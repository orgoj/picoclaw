# PICOCLAW TOOL PROMPTS

This document contains all tool definitions extracted from the PicoClaw codebase.

---

## Tool: append_file

**Description:** Append content to the end of a file

**Parameters:**
- `path`: string (required) - The file path to append to
- `content`: string (required) - The content to append

**Implementation:**
- Uses `validatePath()` to ensure path is within workspace if restriction is enabled
- Opens file with `os.O_APPEND|os.O_CREATE|os.O_WRONLY` flags
- Creates file if it doesn't exist
- Returns silent result on success

---

## Tool: cron

**Description:** Schedule reminders, tasks, or system commands. IMPORTANT: When user asks to be reminded or scheduled, you MUST call this tool. Use 'at_seconds' for one-time reminders (e.g., 'remind me in 10 minutes' → at_seconds=600). Use 'every_seconds' ONLY for recurring tasks (e.g., 'every 2 hours' → every_seconds=7200). Use 'cron_expr' for complex recurring schedules. Use 'command' to execute shell commands directly.

**Parameters:**
- `action`: string (required) - Action to perform. Use 'add' when user wants to schedule a reminder or task. Options: `add`, `list`, `remove`, `enable`, `disable`
- `message`: string - The reminder/task message to display when triggered. If 'command' is used, this describes what the command does.
- `command`: string - Optional: Shell command to execute directly (e.g., 'df -h'). If set, the agent will run this command and report output instead of just showing the message. 'deliver' will be forced to false for commands.
- `at_seconds`: integer - One-time reminder: seconds from now when to trigger (e.g., 600 for 10 minutes later). Use this for one-time reminders like 'remind me in 10 minutes'.
- `every_seconds`: integer - Recurring interval in seconds (e.g., 3600 for every hour). Use this ONLY for recurring tasks like 'every 2 hours' or 'daily reminder'.
- `cron_expr`: string - Cron expression for complex recurring schedules (e.g., '0 9 * * *' for daily at 9am). Use this for complex recurring schedules.
- `job_id`: string - Job ID (for remove/enable/disable)
- `deliver`: boolean - If true, send message directly to channel. If false, let agent process message (for complex tasks). Default: true

**Implementation:**
- Actions: `add`, `list`, `remove`, `enable`, `disable`
- Requires session context (channel/chatID) for adding jobs
- Schedule priority: `at_seconds` > `every_seconds` > `cron_expr`
- Supports shell command execution via `command` parameter
- Uses `CronService` for job management
- Jobs can be delivered directly to chat or processed through agent
- Job names are truncated to 30 chars max

---

## Tool: edit_file

**Description:** Edit a file by replacing old_text with new_text. The old_text must exist exactly in the file.

**Parameters:**
- `path`: string (required) - The file path to edit
- `old_text`: string (required) - The exact text to find and replace
- `new_text`: string (required) - The text to replace with

**Implementation:**
- Validates path with `validatePath()` for workspace restriction
- Checks if file exists before editing
- Returns error if `old_text` not found in file
- Returns error if `old_text` appears multiple times (requires unique context)
- Uses exact string matching (case-sensitive)
- Writes file with mode 0644

---

## Tool: exec

**Description:** Execute a shell command and return its output. Use with caution.

**Parameters:**
- `command`: string (required) - The shell command to execute
- `working_dir`: string - Optional working directory for the command

**Implementation:**
- Default timeout: 60 seconds
- Uses `sh -c` on Unix, `powershell -NoProfile -NonInteractive -Command` on Windows
- **Safety guard with deny patterns:**
  - `rm -rf`, `del /f`, `rmdir /s`
  - `format`, `mkfs`, `diskpart`
  - `dd if=`
  - Writes to `/dev/sd[a-z]`
  - `shutdown`, `reboot`, `poweroff`
  - Fork bomb pattern
- **Path traversal protection** when `restrictToWorkspace` is enabled
- Output truncated to 10,000 chars max
- Returns both stdout and stderr
- Reports timeout errors clearly

---

## Tool: i2c

**Description:** Interact with I2C bus devices for reading sensors and controlling peripherals. Actions: detect (list buses), scan (find devices on a bus), read (read bytes from device), write (send bytes to device). Linux only.

**Parameters:**
- `action`: string (required) - Action to perform: `detect`, `scan`, `read`, `write`
- `bus`: string - I2C bus number (e.g. "1" for /dev/i2c-1). Required for scan/read/write.
- `address`: integer - 7-bit I2C device address (0x03-0x77). Required for read/write.
- `register`: integer - Register address to read from or write to. If set, sends register byte before read/write.
- `data`: array of integers - Bytes to write (0-255 each). Required for write action.
- `length`: integer - Number of bytes to read (1-256). Default: 1. Used with read action.
- `confirm`: boolean - Must be true for write operations. Safety guard to prevent accidental writes.

**Implementation:**
- **Linux only** - requires `/dev/i2c-*` device files
- `detect`: Scans `/dev/i2c-*` via glob pattern
- `scan`: Finds devices on specified bus
- `read`: Reads bytes from device register
- `write`: Sends bytes to device (requires `confirm=true`)
- Bus ID validation (must be numeric to prevent path injection)
- Address validation (0x03-0x77 range for valid 7-bit addresses)

---

## Tool: list_dir

**Description:** List files and directories in a path

**Parameters:**
- `path`: string (required) - Path to list

**Implementation:**
- Defaults to "." if no path specified
- Validates path with `validatePath()` for workspace restriction
- Prefixes output with "DIR:" for directories, "FILE:" for files
- Returns error if directory doesn't exist or can't be read

---

## Tool: message

**Description:** Send a message to user on a chat channel. Use this when you want to communicate something.

**Parameters:**
- `content`: string (required) - The message content to send
- `channel`: string - Optional: target channel (telegram, whatsapp, etc.)
- `chat_id`: string - Optional: target chat/user ID

**Implementation:**
- Uses default channel/chatID from context if not specified
- Requires `SendCallback` to be configured
- Tracks if message was sent in current round (`sentInRound` flag)
- Returns silent result (user receives message directly)
- Error if no target channel/chat specified

---

## Tool: read_file

**Description:** Read the contents of a file

**Parameters:**
- `path`: string (required) - Path to the file to read

**Implementation:**
- Validates path with `validatePath()` for workspace restriction
- Returns full file contents as string
- Returns error if file doesn't exist or can't be read

---

## Tool: spawn

**Description:** Spawn a subagent to handle a task in the background. Use this for complex or time-consuming tasks that can run independently. The subagent will complete the task and report back when done.

**Parameters:**
- `task`: string (required) - The task for subagent to complete
- `label`: string - Optional short label for the task (for display)

**Implementation:**
- Runs tasks asynchronously in background goroutine
- Uses `SubagentManager` for task management
- Returns `AsyncResult` immediately (doesn't block)
- Supports async callback notification on completion
- Subagent has access to tool registry
- Max iterations configurable (default: 10)
- Reports back via MessageBus when complete

---

## Tool: spi

**Description:** Interact with SPI bus devices for high-speed peripheral communication. Actions: list (find SPI devices), transfer (full-duplex send/receive), read (receive bytes). Linux only.

**Parameters:**
- `action`: string (required) - Action to perform: `list`, `transfer`, `read`
- `device`: string - SPI device identifier (e.g. "2.0" for /dev/spidev2.0). Required for transfer/read.
- `speed`: integer - SPI clock speed in Hz. Default: 1000000 (1 MHz).
- `mode`: integer - SPI mode (0-3). Default: 0. Mode sets CPOL and CPHA: 0=0,0 1=0,1 2=1,0 3=1,1.
- `bits`: integer - Bits per word. Default: 8.
- `data`: array of integers - Bytes to send (0-255 each). Required for transfer action.
- `length`: integer - Number of bytes to read (1-4096). Required for read action.
- `confirm`: boolean - Must be true for transfer operations. Safety guard to prevent accidental writes.

**Implementation:**
- **Linux only** - requires `/dev/spidev*` device files
- `list`: Scans `/dev/spidev*` via glob pattern
- `transfer`: Full-duplex send/receive (requires `confirm=true`)
- `read`: Receive bytes by sending zeros
- Device ID validation (must be "X.Y" format)
- Speed range: 1 Hz to 125 MHz
- Mode range: 0-3
- Bits per word: 1-32

---

## Tool: subagent

**Description:** Execute a subagent task synchronously and return the result. Use this for delegating specific tasks to an independent agent instance. Returns execution summary to user and full details to LLM.

**Parameters:**
- `task`: string (required) - The task for subagent to complete
- `label`: string - Optional short label for the task (for display)

**Implementation:**
- Runs tasks **synchronously** (blocks until complete)
- Uses `RunToolLoop` for execution with tools
- Returns different content for user vs LLM:
  - `ForUser`: Brief summary (truncated to 500 chars)
  - `ForLLM`: Full execution details with label, iterations, result
- Max iterations configurable (default: 10)
- Subagent system prompt: "You are a subagent. Complete the given task independently and provide a clear, concise result."

---

## Tool: web_fetch

**Description:** Fetch a URL and extract readable content (HTML to text). Use this to get weather info, news, articles, or any web content.

**Parameters:**
- `url`: string (required) - URL to fetch
- `maxChars`: integer - Maximum characters to extract (minimum: 100)

**Implementation:**
- Default max chars: 50,000
- Only allows http/https URLs
- Requires valid domain in URL
- Timeout: 60 seconds
- Max 5 redirects
- User-Agent: Chrome 120
- **Content extraction:**
  - JSON: Pretty-printed
  - HTML: Script/style tags removed, then text extracted
  - Other: Raw content
- Returns metadata: URL, status code, extractor type, truncated flag, length

---

## Tool: web_search

**Description:** Search the web for current information. Returns titles, URLs, and snippets from search results.

**Parameters:**
- `query`: string (required) - Search query
- `count`: integer - Number of results (1-10)

**Implementation:**
- **Search providers (priority order):**
  1. Brave Search API (requires API key)
  2. DuckDuckGo HTML scraping (no API key needed)
- Default max results: 5
- Brave: Uses `api.search.brave.com/res/v1/web/search`
- DuckDuckGo: Uses `html.duckduckgo.com/html/` with regex extraction
- Returns: Title, URL, and description/snippet for each result
- Timeout: 10 seconds

---

## Tool: write_file

**Description:** Write content to a file

**Parameters:**
- `path`: string (required) - Path to the file to write
- `content`: string (required) - Content to write to the file

**Implementation:**
- Validates path with `validatePath()` for workspace restriction
- Creates parent directories if they don't exist (`os.MkdirAll`)
- Writes file with mode 0644
- Overwrites existing files
- Returns silent result on success

---

## Summary

| Tool | Category | Platform |
|------|----------|----------|
| append_file | Filesystem | Cross-platform |
| cron | Scheduling | Cross-platform |
| edit_file | Filesystem | Cross-platform |
| exec | Shell | Cross-platform |
| i2c | Hardware/I2C | Linux only |
| list_dir | Filesystem | Cross-platform |
| message | Communication | Cross-platform |
| read_file | Filesystem | Cross-platform |
| spawn | Subagent | Cross-platform |
| spi | Hardware/SPI | Linux only |
| subagent | Subagent | Cross-platform |
| web_fetch | Web | Cross-platform |
| web_search | Web | Cross-platform |
| write_file | Filesystem | Cross-platform |

**Total: 14 tools**
