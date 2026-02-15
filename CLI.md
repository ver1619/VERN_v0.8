# v0.8 CLI Documentation

This document provides a detailed guide on how to use the VERN-KV v0.8 Command-Line Interface (CLI).

## Overview

`VERN-CLI` is an **interactive shell** that allows you to interact with the ***VERN Key-Value Store***.<br>

The CLI connects directly to the VernKV engine and allows users to:<br>
- Initialize a new database or Open existing database<br>
- Perform CRUD operations (Put, Get, Delete)<br>
- Execute range or prefix scans<br>

## Starting the CLI

To start the CLI, run the compiled binary:

```python
./bin/vern-cli
```

VernKV CLI will start operating in interactive mode and will display the following prompt:

```python
Starting VernKV CLI...
Initializing environment...
Preparing storage runtime...
CLI ready.

Type HELP for available commands.
(VERN) >  
```

## Commands

### OPEN

**Syntax :** `OPEN <path>`

**Description :** Opens a database at the specified path. If the database does not exist, it creates one.

**Example :**
```python
(VERN) > OPEN ./db
Opening database at ./db
Creating data directory...
Initializing storage engine...
Server is ready.
(VERN) > 
```

**Note :** The CLI operates over a **single database directory** (only one database can be opened per session). You can switch between databases by using the `OPEN` command.

### HELP

**Syntax :** `HELP`

**Description :** Displays the list of available commands and their usage.

**Example :**
```python
(VERN) > HELP
Available commands:<br>

  CLEAR                    - Clear the terminal screen
  DELETE <key>             - Delete a key-value pair
  EXIT                     - Exit the CLI
  GET <key>                - Retrieve the value for a key
  HELP                     - Display available commands
  OPEN <path>              - Open a database at the specified path
  PUT <key> <value>        - Insert or update a key-value pair
  SCAN <keyN> <keyM>       - Range scan from keyN to keyM
  SCAN -pre <key>          - Prefix scan for keys starting with prefix
(VERN) > 
```

### PUT

**Syntax :** `PUT <key> <value>`

**Description :** Inserts or updates a key-value pair.

**Example :**
```python
(VERN) > PUT user:101 {"name": "John", "age": 30}
OK
(VERN) > 
```

### GET

**Syntax :** `GET <key>`

**Description :** Retrieve the latest visible value for a key.

**Example :**
```python
(VERN) > GET user:101
{"name": "John", "age": 30}
(VERN) > 
```

**Output if key is not found:**
```python
NOT FOUND
(VERN) > 
```

### DELETE

**Syntax :** `DELETE <key>`

**Description :** Delete a key.

**Example :**
```python
(VERN) > DELETE user:101
OK
(VERN) > 
```

### SCAN (Range Scan)

**Syntax :** `SCAN <start_key> <end_key>`

**Description :** Scan keys in sorted order.

**Example :**
```python
(VERN) > SCAN user:100 user:200
user:101 {"name": "John", "age": 30}
user:102 {"name": "Jane", "age": 25}
END
(VERN) > 
```

### SCAN (Prefix Scan)

**Syntax :** `SCAN -pre <prefix>`

**Description :** Scan keys matching a prefix.

**Example :**
```python
(VERN) > SCAN -pre user:
user:101 {"name": "John", "age": 30}
user:102 {"name": "Jane", "age": 25}
END
(VERN) > 
```

### CLEAR

**Syntax :** `CLEAR`

**Description :** Clear the terminal screen.

**Example :**
```python
(VERN) > CLEAR
(VERN) > 
```

### EXIT

**Syntax :** `EXIT`

**Description :** Exit the CLI.

**Example :**
```python
(VERN) > EXIT
Shutting down...
```

## Keyboard Shortcuts

The CLI supports standard terminal interactions:
- **Up/Down Arrows ⬆️ / ⬇️**: Navigate through command history.
- **Ctrl+C**: Interrupt/Exit.











