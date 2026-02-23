package logger

import (
	"strings"
	"testing"
)

func TestLogLevelFiltering(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	SetLevel(WARN)

	tests := []struct {
		name      string
		level     LogLevel
		shouldLog bool
	}{
		{"DEBUG message", DEBUG, false},
		{"INFO message", INFO, false},
		{"WARN message", WARN, true},
		{"ERROR message", ERROR, true},
		{"FATAL message", FATAL, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.level {
			case DEBUG:
				Debug(tt.name)
			case INFO:
				Info(tt.name)
			case WARN:
				Warn(tt.name)
			case ERROR:
				Error(tt.name)
			case FATAL:
				if tt.shouldLog {
					t.Logf("FATAL test skipped to prevent program exit")
				}
			}
		})
	}

	SetLevel(INFO)
}

func TestLoggerWithComponent(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	SetLevel(DEBUG)

	tests := []struct {
		name      string
		component string
		message   string
		fields    map[string]interface{}
	}{
		{"Simple message", "test", "Hello, world!", nil},
		{"Message with component", "discord", "Discord message", nil},
		{"Message with fields", "telegram", "Telegram message", map[string]interface{}{
			"user_id": "12345",
			"count":   42,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch {
			case tt.fields == nil && tt.component != "":
				InfoC(tt.component, tt.message)
			case tt.fields != nil:
				InfoF(tt.message, tt.fields)
			default:
				Info(tt.message)
			}
		})
	}

	SetLevel(INFO)
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
		want  string
	}{
		{"DEBUG level", DEBUG, "DEBUG"},
		{"INFO level", INFO, "INFO"},
		{"WARN level", WARN, "WARN"},
		{"ERROR level", ERROR, "ERROR"},
		{"FATAL level", FATAL, "FATAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if logLevelNames[tt.level] != tt.want {
				t.Errorf("logLevelNames[%d] = %s, want %s", tt.level, logLevelNames[tt.level], tt.want)
			}
		})
	}
}

func TestSetGetLevel(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	tests := []LogLevel{DEBUG, INFO, WARN, ERROR, FATAL}

	for _, level := range tests {
		SetLevel(level)
		if GetLevel() != level {
			t.Errorf("SetLevel(%v) -> GetLevel() = %v, want %v", level, GetLevel(), level)
		}
	}
}

func TestLoggerHelperFunctions(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	SetLevel(INFO)

	Debug("This should not log")
	Info("This should log")
	Warn("This should log")
	Error("This should log")

	InfoC("test", "Component message")
	InfoF("Fields message", map[string]interface{}{"key": "value"})

	WarnC("test", "Warning with component")
	ErrorF("Error with fields", map[string]interface{}{"error": "test"})

	SetLevel(DEBUG)
	DebugC("test", "Debug with component")
	WarnF("Warning with fields", map[string]interface{}{"key": "value"})
}

func TestExtractSubagentID(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]interface{}
		wantID string
		wantOK bool
	}{
		{
			name:   "from run_id",
			fields: map[string]interface{}{"run_id": "subagent-7"},
			wantID: "subagent-7",
			wantOK: true,
		},
		{
			name:   "from task_id",
			fields: map[string]interface{}{"task_id": "subagent-9"},
			wantID: "subagent-9",
			wantOK: true,
		},
		{
			name:   "from sender_id",
			fields: map[string]interface{}{"sender_id": "subagent:subagent-11"},
			wantID: "subagent-11",
			wantOK: true,
		},
		{
			name:   "no subagent id",
			fields: map[string]interface{}{"run_id": "telegram_1"},
			wantID: "",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotID, gotOK := extractSubagentID(tc.fields)
			if gotOK != tc.wantOK || gotID != tc.wantID {
				t.Fatalf("extractSubagentID() = (%q,%v), want (%q,%v)", gotID, gotOK, tc.wantID, tc.wantOK)
			}
		})
	}
}

func TestTruncateConsoleString(t *testing.T) {
	short := "short"
	if got := truncateConsoleString(short); got != short {
		t.Fatalf("short string changed: got %q", got)
	}

	long := strings.Repeat("a", consoleStringLimit+25)
	got := truncateConsoleString(long)
	if got == long {
		t.Fatal("expected long string to be truncated")
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected ellipsis suffix, got %q", got)
	}
}

func TestTrimFieldsForConsole(t *testing.T) {
	long := strings.Repeat("x", consoleStringLimit+10)
	fields := map[string]interface{}{
		"s": long,
		"nested": map[string]interface{}{
			"inner": long,
		},
		"arr": []string{long},
	}

	trimmed := trimFieldsForConsole(fields)
	if trimmed["s"] == long {
		t.Fatal("expected top-level string to be truncated")
	}
	nested, ok := trimmed["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested map, got %T", trimmed["nested"])
	}
	if nested["inner"] == long {
		t.Fatal("expected nested string to be truncated")
	}
	arr, ok := trimmed["arr"].([]string)
	if !ok || len(arr) != 1 {
		t.Fatalf("expected []string[1], got %T len=%d", trimmed["arr"], len(arr))
	}
	if arr[0] == long {
		t.Fatal("expected slice string to be truncated")
	}
}
