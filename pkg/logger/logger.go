package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var (
	logLevelNames = map[LogLevel]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
		FATAL: "FATAL",
	}

	currentLevel = INFO
	logger       *Logger
	once         sync.Once
	mu           sync.RWMutex
)

type Logger struct {
	file         *os.File
	debugFile    *os.File
	agentLogsDir string
}

const consoleStringLimit = 220

type LogEntry struct {
	Level     string                 `json:"level"`
	Timestamp string                 `json:"timestamp"`
	Component string                 `json:"component,omitempty"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
}

func init() {
	once.Do(func() {
		logger = &Logger{}
	})
}

func SetLevel(level LogLevel) {
	mu.Lock()
	defer mu.Unlock()
	currentLevel = level
}

func GetLevel() LogLevel {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel
}

func EnableFileLogging(filePath string) error {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	if logger.file != nil {
		logger.file.Close()
	}

	logger.file = file
	log.Println("File logging enabled:", filePath)
	return nil
}

func DisableFileLogging() {
	mu.Lock()
	defer mu.Unlock()

	if logger.file != nil {
		logger.file.Close()
		logger.file = nil
		log.Println("File logging disabled")
	}
}

func EnableDebugFileLogging(filePath string) error {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open debug log file: %w", err)
	}

	if logger.debugFile != nil {
		logger.debugFile.Close()
	}

	logger.debugFile = file
	log.Println("Debug file logging enabled:", filePath)
	return nil
}

func SetAgentLogsDir(dirPath string) error {
	mu.Lock()
	defer mu.Unlock()

	if strings.TrimSpace(dirPath) == "" {
		logger.agentLogsDir = ""
		return nil
	}
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create agent logs dir: %w", err)
	}
	logger.agentLogsDir = dirPath
	log.Println("Agent-scoped file logging enabled:", dirPath)
	return nil
}

func DisableDebugFileLogging() {
	mu.Lock()
	defer mu.Unlock()

	if logger.debugFile != nil {
		logger.debugFile.Close()
		logger.debugFile = nil
		log.Println("Debug file logging disabled")
	}
}

func logMessage(level LogLevel, component string, message string, fields map[string]interface{}) {
	if level < currentLevel {
		return
	}

	entry := LogEntry{
		Level:     logLevelNames[level],
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Component: component,
		Message:   message,
		Fields:    fields,
	}

	if pc, file, line, ok := runtime.Caller(2); ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			entry.Caller = fmt.Sprintf("%s:%d (%s)", file, line, fn.Name())
		}
	}

	if logger.file != nil {
		jsonData, err := json.Marshal(entry)
		if err == nil {
			logger.file.WriteString(string(jsonData) + "\n")
		}
	}
	writeAgentScopedEntry(entry, fields)

	consoleMessage := truncateConsoleString(message)
	consoleFields := trimFieldsForConsole(fields)

	var fieldStr string
	if len(consoleFields) > 0 {
		fieldStr = " " + formatFields(consoleFields)
	}
	flowPrefix := deriveFlowPrefix(fields)

	logLine := fmt.Sprintf("[%s] [%s]%s%s %s%s",
		entry.Timestamp,
		logLevelNames[level],
		flowPrefix,
		formatComponent(component),
		consoleMessage,
		fieldStr,
	)

	log.Println(logLine)
	if logger.debugFile != nil {
		jsonData, err := json.Marshal(entry)
		if err == nil {
			logger.debugFile.WriteString(string(jsonData) + "\n")
		}
	}

	if level == FATAL {
		os.Exit(1)
	}
}

func writeAgentScopedEntry(entry LogEntry, fields map[string]interface{}) {
	dir := strings.TrimSpace(logger.agentLogsDir)
	if dir == "" {
		return
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return
	}
	writeScopedLine := func(filename string) {
		path := filepath.Join(dir, filename)
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return
		}
		_, _ = f.WriteString(string(raw) + "\n")
		_ = f.Close()
	}

	writeScopedLine("main.jsonl")
	if taskID, ok := extractSubagentID(fields); ok {
		writeScopedLine(taskID + ".jsonl")
	}
}

func formatComponent(component string) string {
	if component == "" {
		return ""
	}
	return fmt.Sprintf(" %s:", component)
}

func formatFields(fields map[string]interface{}) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := fields[k]
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
}

func deriveFlowPrefix(fields map[string]interface{}) string {
	if len(fields) == 0 {
		return ""
	}

	if taskID, ok := extractSubagentID(fields); ok {
		return fmt.Sprintf(" [SUBAGENT:%s]", taskID)
	}
	return ""
}

func truncateConsoleString(s string) string {
	r := []rune(s)
	if len(r) <= consoleStringLimit {
		return s
	}
	return string(r[:consoleStringLimit]) + "..."
}

func trimFieldsForConsole(fields map[string]interface{}) map[string]interface{} {
	if len(fields) == 0 {
		return fields
	}
	out := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		out[k] = trimValueForConsole(v)
	}
	return out
}

func trimValueForConsole(v interface{}) interface{} {
	switch tv := v.(type) {
	case string:
		return truncateConsoleString(tv)
	case fmt.Stringer:
		return truncateConsoleString(tv.String())
	case map[string]interface{}:
		return trimFieldsForConsole(tv)
	case []string:
		out := make([]string, len(tv))
		for i, s := range tv {
			out[i] = truncateConsoleString(s)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(tv))
		for i, item := range tv {
			out[i] = trimValueForConsole(item)
		}
		return out
	default:
		return v
	}
}

func extractSubagentID(fields map[string]interface{}) (string, bool) {
	if raw, ok := fields["run_id"].(string); ok && strings.HasPrefix(raw, "subagent-") {
		return raw, true
	}
	if raw, ok := fields["task_id"].(string); ok && strings.HasPrefix(raw, "subagent-") {
		return raw, true
	}
	if raw, ok := fields["sender_id"].(string); ok && strings.HasPrefix(raw, "subagent:") {
		return strings.TrimPrefix(raw, "subagent:"), true
	}
	return "", false
}

func Debug(message string) {
	logMessage(DEBUG, "", message, nil)
}

func DebugC(component string, message string) {
	logMessage(DEBUG, component, message, nil)
}

func DebugF(message string, fields map[string]interface{}) {
	logMessage(DEBUG, "", message, fields)
}

func DebugCF(component string, message string, fields map[string]interface{}) {
	logMessage(DEBUG, component, message, fields)
}

func Info(message string) {
	logMessage(INFO, "", message, nil)
}

func InfoC(component string, message string) {
	logMessage(INFO, component, message, nil)
}

func InfoF(message string, fields map[string]interface{}) {
	logMessage(INFO, "", message, fields)
}

func InfoCF(component string, message string, fields map[string]interface{}) {
	logMessage(INFO, component, message, fields)
}

func Warn(message string) {
	logMessage(WARN, "", message, nil)
}

func WarnC(component string, message string) {
	logMessage(WARN, component, message, nil)
}

func WarnF(message string, fields map[string]interface{}) {
	logMessage(WARN, "", message, fields)
}

func WarnCF(component string, message string, fields map[string]interface{}) {
	logMessage(WARN, component, message, fields)
}

func Error(message string) {
	logMessage(ERROR, "", message, nil)
}

func ErrorC(component string, message string) {
	logMessage(ERROR, component, message, nil)
}

func ErrorF(message string, fields map[string]interface{}) {
	logMessage(ERROR, "", message, fields)
}

func ErrorCF(component string, message string, fields map[string]interface{}) {
	logMessage(ERROR, component, message, fields)
}

func Fatal(message string) {
	logMessage(FATAL, "", message, nil)
}

func FatalC(component string, message string) {
	logMessage(FATAL, component, message, nil)
}

func FatalF(message string, fields map[string]interface{}) {
	logMessage(FATAL, "", message, fields)
}

func FatalCF(component string, message string, fields map[string]interface{}) {
	logMessage(FATAL, component, message, fields)
}
