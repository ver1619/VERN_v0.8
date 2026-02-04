package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"vern_kv0.8/engine"
)

var (
	db     *engine.DB
	isOpen bool
)

// ANSI Color Codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
)

func main() {
	// Setup Signal Handling for Graceful Shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nCaught signal, shutting down...")
		if isOpen && db != nil {
			db.Close()
		}
		os.Exit(0)
	}()

	// 5. Startup Responsibilities
	fmt.Println("Starting VernKV CLI...")
	fmt.Println("Initializing environment...")
	fmt.Println("Preparing storage runtime...")
	fmt.Println("CLI ready.")
	fmt.Println("")
	fmt.Println("Type HELP for available commands.")

	scanner := bufio.NewScanner(os.Stdin)
	printPrompt()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			handleCommand(line)
		}
		printPrompt()
	}

	if err := scanner.Err(); err != nil {
		printError(fmt.Sprintf("Error reading input: %v", err))
		os.Exit(1)
	}
}

func printPrompt() {
	fmt.Printf("%s(VERN) > %s", ColorYellow, ColorReset)
}

func printError(msg string) {
	fmt.Printf("%s%s%s\n", ColorRed, msg, ColorReset)
}

func printSuccess(msg string) {
	fmt.Printf("%s%s%s\n", ColorBlue, msg, ColorReset)
}

func handleCommand(line string) {
	// Split by whitespace, but be careful with PUT value
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToUpper(parts[0])

	switch cmd {
	case "OPEN":
		execOpen(parts)
	case "PUT":
		execPut(line, parts)
	case "GET":
		execGet(parts)
	case "DELETE":
		execDelete(parts)
	case "SCAN":
		execScan(parts)
	case "CLEAR":
		execClear()
	case "HELP":
		execHelp()
	case "EXIT":
		execExit()
	default:
		printError(fmt.Sprintf("ERROR: unknown command '%s'", cmd))
	}
}

// Command 1: OPEN
func execOpen(parts []string) {
	if isOpen {
		printError("ERROR: database already opened for this session")
		return
	}
	if len(parts) < 2 {
		printError("ERROR: missing argument <path>")
		return
	}
	path := parts[1]
	if path == "" || path == "\"\"" {
		printError("ERROR: invalid path")
		return
	}

	dataDir := filepath.Join(path, "data")

	fmt.Printf("Opening database at %s\n", path)

	// Check if directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		fmt.Println("Creating data directory...")
		// Ensure parent is writable/exists handled by MkdirAll usually, or engine.Open
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			printError(fmt.Sprintf("ERROR: cannot create data directory (%v)", err))
			return
		}
	}

	fmt.Println("Initializing storage engine...")
	var err error
	db, err = engine.Open(dataDir)
	if err != nil {
		printError(fmt.Sprintf("ERROR: failed to open database: %v", err))
		return
	}

	isOpen = true
	fmt.Println("Server is ready.")
}

// Command 2: PUT
func execPut(line string, parts []string) {
	if !ensureOpen() {
		return
	}
	// Syntax: PUT <key> <value>
	// Value can contain spaces (JSON, text).
	if len(parts) < 2 {
		printError("ERROR: missing argument <key>")
		return
	}

	key := parts[1]

	// We need to extract the value part from the original line
	// PUT <key> <value...>
	// parts[0] is PUT
	// parts[1] is key
	// value is everything after parts[1]

	if len(parts) < 3 {
		printError("ERROR: missing argument <value>")
		return
	}

	// Robust way to get the rest of the string:
	// Find the first occurrence of the key in the line (after PUT)
	// Then find the first non-space character after the key.

	rest := strings.TrimSpace(line[len(parts[0]):]) // remove PUT
	rest = strings.TrimSpace(rest[len(parts[1]):])  // remove Key

	value := rest

	if value == "" {
		// Should be caught by len(parts) < 3 check above, but double check
		printError("ERROR: missing argument <value>")
		return
	}

	// CLI does not validate content, just opaque string
	err := db.Put([]byte(key), []byte(value))
	if err != nil {
		printError(fmt.Sprintf("ERROR: write failed (%v)", err))
		return
	}
	printSuccess("OK")
}

// Command 3: GET
func execGet(parts []string) {
	if !ensureOpen() {
		return
	}
	if len(parts) < 2 {
		printError("ERROR: missing argument <key>")
		return
	}
	key := parts[1]

	val, err := db.Get([]byte(key))
	if err == engine.ErrNotFound {
		fmt.Println("NOT FOUND")
		return
	}
	if err != nil {
		printError(fmt.Sprintf("ERROR: read failed (%v)", err))
		return
	}

	// Found
	fmt.Println(string(val))
}

// Command 4: DELETE
func execDelete(parts []string) {
	if !ensureOpen() {
		return
	}
	if len(parts) < 2 {
		printError("ERROR: missing argument <key>")
		return
	}
	key := parts[1]

	err := db.Delete([]byte(key))
	if err != nil {
		printError(fmt.Sprintf("ERROR: delete failed (%v)", err))
		return
	}
	printSuccess("OK")
}

// Command 5: SCAN
func execScan(parts []string) {
	if !ensureOpen() {
		return
	}

	// SCAN -pre <key>
	// SCAN <keyN> <keyM>

	if len(parts) < 2 {
		// Just SCAN? Spec says: "ERROR: missing argument <keyM>" for single arg case
		// But let's check flags first
		printError("ERROR: missing arguments")
		return
	}

	if strings.HasPrefix(parts[1], "-") {
		// Flag?
		if parts[1] == "-pre" {
			// Prefix scan
			if len(parts) < 3 {
				printError("ERROR: missing argument <key>")
				return
			}
			prefix := parts[2]
			runPrefixScan(prefix)
		} else {
			printError("ERROR: unknown flag")
		}
		return
	}

	// Range scan
	if len(parts) < 3 {
		printError("ERROR: missing argument <keyM>")
		return
	}

	start := parts[1]
	end := parts[2]

	// Lexicographical check
	if start > end {
		printError("ERROR: invalid key range")
		return
	}

	runRangeScan(start, end)
}

func runPrefixScan(prefix string) {
	// engine.NewPrefixIterator(prefix, nil)
	// Spec: iterate and print <key> <value> then END
	it := db.NewPrefixIterator([]byte(prefix), nil)
	defer printEnd()

	for it.SeekToFirst(); it.Valid(); it.Next() {
		fmt.Printf("%s %s\n", string(it.Key()), string(it.Value()))
	}
}

func runRangeScan(start, end string) {
	// engine.NewRangeIterator(start, end, nil)
	it := db.NewRangeIterator([]byte(start), []byte(end), nil)
	defer printEnd()

	for it.SeekToFirst(); it.Valid(); it.Next() {
		fmt.Printf("%s %s\n", string(it.Key()), string(it.Value()))
	}
}

func printEnd() {
	fmt.Println("END")
}

// Utility Commands
func execClear() {
	// Clear screen and move cursor to top left
	fmt.Print("\033[2J\033[H")
}

func execHelp() {
	fmt.Println("OPEN <path>")
	fmt.Println("PUT <key> <value>")
	fmt.Println("GET <key>")
	fmt.Println("DELETE <key>")
	fmt.Println("SCAN <keyN> <keyM>")
	fmt.Println("SCAN -pre <key>")
	fmt.Println("CLEAR")
	fmt.Println("HELP")
	fmt.Println("EXIT")
}

func execExit() {
	fmt.Println("Shutting down...")
	if isOpen && db != nil {
		db.Close()
	}
	os.Exit(0)
}

// Helpers
func ensureOpen() bool {
	if !isOpen {
		printError("ERROR: database not opened (run OPEN <path> first)")
		return false
	}
	return true
}
