package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Event struct {
	Timestamp string                 `json:"timestamp"`
	Type      string                 `json:"type"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

var (
	mu      sync.Mutex
	enabled bool
	file    *os.File
)

func Enable(logDir string) error {
	mu.Lock()
	defer mu.Unlock()

	if file != nil {
		file.Close()
		file = nil
	}
	enabled = false
	if logDir == "" {
		return nil
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}
	path := filepath.Join(logDir, "audit.jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	file = f
	enabled = true
	return nil
}

func Disable() {
	mu.Lock()
	defer mu.Unlock()
	enabled = false
	if file != nil {
		file.Close()
		file = nil
	}
}

func Record(eventType string, fields map[string]interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if !enabled || file == nil {
		return
	}

	entry := Event{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Type:      eventType,
		Fields:    fields,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_, _ = file.Write(append(data, '\n'))
}
