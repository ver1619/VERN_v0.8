package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/term"
	"vern_kv0.8/engine"
)

var (
	db     *engine.DB
	isOpen bool
)

// ANSI color codes for CLI styling.
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
)

// Command history management
var (
	history        []string
	historyIndex   int
	maxHistorySize = 1000
)

// Terminal state for raw mode toggling
var termState *term.State

func main() {
	// Catch interrupt signals for graceful shutdown.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		// Restore terminal before exiting
		if termState != nil {
			term.Restore(int(os.Stdin.Fd()), termState)
		}
		fmt.Println("\nCaught signal, shutting down...")
		if isOpen && db != nil {
			db.Close()
		}
		os.Exit(0)
	}()

	// CLI startup sequence.
	fmt.Println("Starting VernKV CLI...")
	fmt.Println("Initializing environment...")
	fmt.Println("Preparing storage runtime...")
	fmt.Println("CLI ready.")
	fmt.Println("")
	fmt.Println("Type HELP for available commands.")

	// Initialize history
	history = make([]string, 0, maxHistorySize)
	historyIndex = 0

	// Main input loop with history support
	runInputLoop()
}

// Switch the terminal to raw mode for key-by-key input.
func enterRawMode() error {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	termState = oldState
	return nil
}

// Restore the terminal to cooked (normal) mode for clean output.
func exitRawMode() {
	if termState != nil {
		term.Restore(int(os.Stdin.Fd()), termState)
	}
}

func runInputLoop() {
	// Enter raw mode for input
	if err := enterRawMode(); err != nil {
		// Fallback to simple mode without history if terminal setup fails
		fmt.Printf("Warning: Failed to enable advanced input mode: %v\n", err)
		runSimpleInputLoop()
		return
	}

	rawPrintPrompt()

	var inputBuffer []rune
	historyIndex = len(history)
	tempInput := ""

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			exitRawMode()
			break
		}

		if n == 1 {
			b := buf[0]

			switch b {
			case 3: // Ctrl+C
				exitRawMode()
				fmt.Println("\n^C")
				if isOpen && db != nil {
					db.Close()
				}
				os.Exit(0)
			case 4: // Ctrl+D (EOF)
				if len(inputBuffer) == 0 {
					exitRawMode()
					fmt.Println("\nExiting...")
					if isOpen && db != nil {
						db.Close()
					}
					os.Exit(0)
				}
			case 13: // Enter
				// Move to next line in raw mode
				fmt.Print("\r\n")
				line := strings.TrimSpace(string(inputBuffer))

				// Exit raw mode so command output prints normally
				exitRawMode()

				if line != "" {
					addToHistory(line)
					handleCommand(line)
				}

				// Re-enter raw mode for next input
				if err := enterRawMode(); err != nil {
					fmt.Println("Fatal: failed to re-enter raw mode")
					os.Exit(1)
				}

				inputBuffer = inputBuffer[:0]
				historyIndex = len(history)
				tempInput = ""
				rawPrintPrompt()
			case 127: // Backspace
				if len(inputBuffer) > 0 {
					inputBuffer = inputBuffer[:len(inputBuffer)-1]
					// Clear line and redraw
					fmt.Print("\r\033[K")
					rawPrintPrompt()
					fmt.Print(string(inputBuffer))
				}
			default:
				if b >= 32 && b < 127 { // Printable characters
					inputBuffer = append(inputBuffer, rune(b))
					fmt.Printf("%c", b)
				}
			}
		} else if n == 3 && buf[0] == 27 && buf[1] == 91 {
			// Arrow keys
			switch buf[2] {
			case 65: // Up arrow
				if historyIndex > 0 {
					if historyIndex == len(history) {
						tempInput = string(inputBuffer)
					}
					historyIndex--
					inputBuffer = []rune(history[historyIndex])
					fmt.Print("\r\033[K")
					rawPrintPrompt()
					fmt.Print(string(inputBuffer))
				}
			case 66: // Down arrow
				if historyIndex < len(history) {
					historyIndex++
					if historyIndex == len(history) {
						inputBuffer = []rune(tempInput)
					} else {
						inputBuffer = []rune(history[historyIndex])
					}
					fmt.Print("\r\033[K")
					rawPrintPrompt()
					fmt.Print(string(inputBuffer))
				}
			}
		}
	}
}

func runSimpleInputLoop() {
	// Fallback simple input loop without terminal control
	var inputBuffer string
	for {
		_, err := fmt.Scanln(&inputBuffer)
		if err != nil {
			break
		}
		line := strings.TrimSpace(inputBuffer)
		if line != "" {
			handleCommand(line)
		}
		printPrompt()
	}
}

func addToHistory(cmd string) {
	// avoid consecutive duplicates
	if len(history) > 0 && history[len(history)-1] == cmd {
		return
	}

	history = append(history, cmd)

	// Trim history if it exceeds max size
	if len(history) > maxHistorySize {
		history = history[1:]
	}
}

// Print the prompt while in raw terminal mode (no translation needed).
func rawPrintPrompt() {
	fmt.Printf("%s(VERN) > %s", ColorYellow, ColorReset)
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

func printInfo(msg string) {
	fmt.Printf("%s%s%s\n", ColorGreen, msg, ColorReset)
}

func handleCommand(line string) {
	// Parse input command and arguments.
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
		printError(fmt.Sprintf("[ERROR] Syntax Error: Unknown command '%s'", cmd))
		fmt.Printf("Type HELP to see available commands.\n")
	}
}

func execOpen(parts []string) {
	if isOpen {
		printInfo("[INFO] Database already opened for this session.")
		return
	}
	if len(parts) < 2 {
		printError("[ERROR] Syntax Error: Path argument required.")
		fmt.Printf("%sUsage:%s OPEN <path>\n", ColorRed, ColorReset)
		return
	}
	path := parts[1]
	if path == "" || path == "\"\"" {
		printError("[ERROR] Syntax Error: Invalid path.")
		fmt.Printf("%sUsage:%s OPEN <path>\n", ColorRed, ColorReset)
		return
	}

	dataDir := filepath.Join(path, "data")

	fmt.Printf("Opening database at %s\n", path)

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		fmt.Println("Creating data directory...")
		// Ensure data directory exists.
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			printError(fmt.Sprintf("[ERROR] System Error: Cannot create data directory (%v)", err))
			return
		}
	}

	fmt.Println("Initializing storage engine...")
	var err error
	db, err = engine.Open(dataDir)
	if err != nil {
		printError(fmt.Sprintf("[ERROR] System Error: Failed to open database: %v", err))
		return
	}

	isOpen = true
	fmt.Println("Server is ready.")
}

func execPut(line string, parts []string) {
	if !ensureOpen() {
		return
	}

	if len(parts) < 2 {
		printError("[ERROR] Syntax Error: Missing key argument.")
		fmt.Printf("%sUsage:%s PUT <key> <value>\n", ColorRed, ColorReset)
		return
	}

	key := parts[1]

	if len(parts) < 3 {
		printError("[ERROR] Syntax Error: Missing value argument.")
		fmt.Printf("%sUsage:%s PUT <key> <value>\n", ColorRed, ColorReset)
		return
	}

	// Extract value while preserving spaces.

	rest := strings.TrimSpace(line[len(parts[0]):]) // remove PUT
	rest = strings.TrimSpace(rest[len(parts[1]):])  // remove Key

	value := rest

	if value == "" {
		printError("[ERROR] Syntax Error: Missing value argument.")
		fmt.Printf("%sUsage:%s PUT <key> <value>\n", ColorRed, ColorReset)
		return
	}

	// Value is treated as an opaque byte slice.
	err := db.Put([]byte(key), []byte(value))
	if err != nil {
		printError(fmt.Sprintf("[ERROR] System Error: Write failed (%v)", err))
		return
	}
	printSuccess("OK")
}

func execGet(parts []string) {
	if !ensureOpen() {
		return
	}
	if len(parts) < 2 {
		printError("[ERROR] Syntax Error: Missing key argument.")
		fmt.Printf("%sUsage:%s GET <key>\n", ColorRed, ColorReset)
		return
	}
	key := parts[1]

	val, err := db.Get([]byte(key))
	if err == engine.ErrNotFound {
		fmt.Println("NOT FOUND")
		return
	}
	if err != nil {
		printError(fmt.Sprintf("[ERROR] System Error: Read failed (%v)", err))
		return
	}

	// Display retrieved value.
	fmt.Println(string(val))
}

func execDelete(parts []string) {
	if !ensureOpen() {
		return
	}
	if len(parts) < 2 {
		printError("[ERROR] Syntax Error: Missing key argument.")
		fmt.Printf("%sUsage:%s DELETE <key>\n", ColorRed, ColorReset)
		return
	}
	key := parts[1]

	err := db.Delete([]byte(key))
	if err != nil {
		printError(fmt.Sprintf("[ERROR] System Error: Delete failed (%v)", err))
		return
	}
	printSuccess("OK")
}

func execScan(parts []string) {
	if !ensureOpen() {
		return
	}

	if len(parts) < 2 {
		// Validate scan arguments.
		printError("[ERROR] Syntax Error: Missing arguments.")
		fmt.Printf("%sUsage:%s SCAN <keyN> <keyM> OR SCAN -pre <key>\n", ColorRed, ColorReset)
		return
	}

	if strings.HasPrefix(parts[1], "-") {
		if parts[1] == "-pre" {
			// Execute prefix-based scan.
			if len(parts) < 3 {
				printError("[ERROR] Syntax Error: Missing key argument.")
				fmt.Printf("%sUsage:%s SCAN -pre <key>\n", ColorRed, ColorReset)
				return
			}
			prefix := parts[2]
			runPrefixScan(prefix)
		} else {
			printError("[ERROR] Syntax Error: Unknown flag.")
			fmt.Printf("%sUsage:%s SCAN <keyN> <keyM> OR SCAN -pre <key>\n", ColorRed, ColorReset)
		}
		return
	}

	// Execute range-based scan.
	if len(parts) < 3 {
		printError("[ERROR] Syntax Error: Missing keyM argument.")
		fmt.Printf("%sUsage:%s SCAN <keyN> <keyM>\n", ColorRed, ColorReset)
		return
	}

	start := parts[1]
	end := parts[2]

	// Ensure start key is not greater than end key.
	if start > end {
		printError("[ERROR] Syntax Error: Invalid key range (keyN > keyM).")
		fmt.Printf("%sUsage:%s SCAN <keyN> <keyM>\n", ColorRed, ColorReset)
		return
	}

	runRangeScan(start, end)
}

func runPrefixScan(prefix string) {
	it := db.NewPrefixIterator([]byte(prefix), nil)
	defer printEnd()

	for it.SeekToFirst(); it.Valid(); it.Next() {
		fmt.Printf("%s %s\n", string(it.Key()), string(it.Value()))
	}
}

func runRangeScan(start, end string) {
	it := db.NewRangeIterator([]byte(start), []byte(end), nil)
	defer printEnd()

	for it.SeekToFirst(); it.Valid(); it.Next() {
		fmt.Printf("%s %s\n", string(it.Key()), string(it.Value()))
	}
}

func printEnd() {
	fmt.Println("END")
}

func execClear() {
	// Clear the terminal screen completely including scrollback buffer.
	fmt.Print("\033[2J\033[3J\033[H")
}

func execHelp() {
	fmt.Println("Available Commands:")
	fmt.Println()
	fmt.Println("  CLEAR                    - Clear the terminal screen")
	fmt.Println("  DELETE <key>             - Delete a key-value pair")
	fmt.Println("  EXIT                     - Exit the CLI")
	fmt.Println("  GET <key>                - Retrieve the value for a key")
	fmt.Println("  HELP                     - Display available commands")
	fmt.Println("  OPEN <path>              - Open a database at the specified path")
	fmt.Println("  PUT <key> <value>        - Insert or update a key-value pair")
	fmt.Println("  SCAN <keyN> <keyM>       - Range scan from keyN to keyM")
	fmt.Println("  SCAN -pre <key>          - Prefix scan for keys starting with prefix")
}

func execExit() {
	fmt.Println("Shutting down...")
	if isOpen && db != nil {
		db.Close()
	}
	os.Exit(0)
}

func ensureOpen() bool {
	if !isOpen {
		printError("[ERROR] State Error: Database not opened.")
		fmt.Printf("%sUsage:%s OPEN <path>\n", ColorRed, ColorReset)
		return false
	}
	return true
}
