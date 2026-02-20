package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

var freeTextFlagValuePatterns = []*regexp.Regexp{
	regexp.MustCompile(`--description\s*=\s*("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|[^\s]+)`),
	regexp.MustCompile(`--description\s+("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|[^\s]+)`),
	regexp.MustCompile(`--body\s*=\s*("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|[^\s]+)`),
	regexp.MustCompile(`--body\s+("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|[^\s]+)`),
	regexp.MustCompile(`--message\s*=\s*("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|[^\s]+)`),
	regexp.MustCompile(`--message\s+("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|[^\s]+)`),
	regexp.MustCompile(`--title\s*=\s*("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|[^\s]+)`),
	regexp.MustCompile(`--title\s+("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|[^\s]+)`),
	regexp.MustCompile(`-m\s+("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|[^\s]+)`),
}

type ExecTool struct {
	workingDir          string
	timeout             time.Duration
	denyPatterns        []*regexp.Regexp
	allowPatterns       []*regexp.Regexp
	restrictToWorkspace bool
}

func NewExecTool(workingDir string, restrict bool) *ExecTool {
	denyPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\brm\s+-[rf]{1,2}\s+/`), // Block rm -rf /
		regexp.MustCompile(`\brm\s+-[rf]{1,2}\s+\$HOME\b`),
		regexp.MustCompile(`\brm\s+-[rf]{1,2}\s+~\b`),
		regexp.MustCompile(`\bdel\s+/[fq]\b`),
		regexp.MustCompile(`\brmdir\s+/s\b`),
		regexp.MustCompile(`\b(format|mkfs|diskpart)\b\s`), // Match disk wiping commands (must be followed by space/args)
		regexp.MustCompile(`\bdd\s+if=`),
		regexp.MustCompile(`>\s*/dev/sd[a-z]\b`), // Block writes to disk devices (but allow /dev/null)
		regexp.MustCompile(`\b(shutdown|reboot|poweroff)\b`),
		regexp.MustCompile(`:\(\)\s*\{.*\};\s*:`),
	}

	return &ExecTool{
		workingDir:          workingDir,
		timeout:             60 * time.Second,
		denyPatterns:        denyPatterns,
		allowPatterns:       nil,
		restrictToWorkspace: restrict,
	}
}

func (t *ExecTool) Name() string {
	return "exec"
}

func (t *ExecTool) Description() string {
	return "Execute a shell command and return its output. Use with caution."
}

func (t *ExecTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The shell command to execute",
			},
			"working_dir": map[string]interface{}{
				"type":        "string",
				"description": "Optional working directory for the command",
			},
		},
		"required": []string{"command"},
	}
}

func (t *ExecTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	command, ok := args["command"].(string)
	if !ok {
		return ErrorResult("command is required")
	}

	cwd := t.workingDir
	if scopedWD, ok := extractWorkingDirectory(ctx); ok {
		cwd = scopedWD
	}
	if wd, ok := args["working_dir"].(string); ok && wd != "" {
		cwd = wd
	}

	if cwd == "" {
		wd, err := os.Getwd()
		if err == nil {
			cwd = wd
		}
	}

	if guardError := t.guardCommand(command, cwd); guardError != "" {
		return ErrorResult(guardError)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(cmdCtx, "powershell", "-NoProfile", "-NonInteractive", "-Command", command)
	} else {
		cmd = exec.CommandContext(cmdCtx, "sh", "-c", command)
	}
	if cwd != "" {
		cmd.Dir = cwd
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR:\n" + stderr.String()
	}

	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			msg := fmt.Sprintf("Command timed out after %v", t.timeout)
			return &ToolResult{
				ForLLM:  msg,
				ForUser: msg,
				IsError: true,
			}
		}
		output += fmt.Sprintf("\nExit code: %v", err)
	}

	if output == "" {
		output = "(no output)"
	}

	maxLen := 10000
	if len(output) > maxLen {
		output = output[:maxLen] + fmt.Sprintf("\n... (truncated, %d more chars)", len(output)-maxLen)
	}

	if err != nil {
		return &ToolResult{
			ForLLM:  output,
			ForUser: output,
			IsError: true,
		}
	}

	return &ToolResult{
		ForLLM:  output,
		ForUser: output,
		IsError: false,
	}
}

func (t *ExecTool) guardCommand(command, cwd string) string {
	cmd := strings.TrimSpace(command)
	lower := strings.ToLower(cmd)

	for _, pattern := range t.denyPatterns {
		if pattern.MatchString(lower) {
			logger.WarnCF("exec", "Command blocked by dangerous pattern", map[string]interface{}{
				"command": command,
				"pattern": pattern.String(),
			})
			return "Command blocked by safety guard (dangerous pattern detected)"
		}
	}

	if len(t.allowPatterns) > 0 {
		allowed := false
		for _, pattern := range t.allowPatterns {
			if pattern.MatchString(lower) {
				allowed = true
				break
			}
		}
		if !allowed {
			logger.WarnCF("exec", "Command blocked by allowlist", map[string]interface{}{
				"command": command,
			})
			return "Command blocked by safety guard (not in allowlist)"
		}
	}

	if t.restrictToWorkspace {
		// Only block explicit path traversal with ..
		if strings.Contains(cmd, "..\\") || strings.Contains(cmd, "../") {
			logger.WarnCF("exec", "Command blocked by path traversal", map[string]interface{}{
				"command": command,
			})
			return "Command blocked by safety guard (path traversal detected)"
		}

		// Get absolute workspace path
		workspacePath, err := filepath.Abs(t.workingDir)
		if err != nil || workspacePath == "" {
			workspacePath, _ = filepath.Abs(cwd)
		}
		if workspacePath == "" {
			return "" // No workspace restriction if we can't determine it
		}

		// Match only ABSOLUTE paths: /something or C:\something
		// Must be preceded by start of string, whitespace, =, or quotes
		// EXCLUSION: Ignore paths that look like URLs (http://, https://) or contain only localhost/IPs without leading slash
		absPathPattern := regexp.MustCompile(`(?:^|[\s="'` + "`" + `])([A-Za-z]:\\[^\s\\\"']+|/(?:[^\s\"'/][^\s\"']*))`)
		matches := absPathPattern.FindAllStringSubmatchIndex(cmd, -1)

		for _, match := range matches {
			if len(match) < 4 {
				continue
			}
			rawStart := match[2]
			rawEnd := match[3]
			raw := cmd[rawStart:rawEnd]

			if isWithinFreeTextFlagValue(cmd, rawStart) {
				continue
			}

			// Skip if it looks like a URL (contains ://)
			if strings.Contains(raw, "://") {
				continue
			}

			// Skip common safe system paths/devices
			if raw == "/dev/null" || raw == "/dev/stdout" || raw == "/dev/stderr" ||
				raw == "/dev/stdin" || raw == "/dev/zero" || raw == "/dev/random" || raw == "/dev/urandom" {
				continue
			}

			p, err := filepath.Abs(raw)
			if err != nil {
				continue
			}

			// Check if absolute path is inside workspace
			if !strings.HasPrefix(p+string(filepath.Separator), workspacePath+string(filepath.Separator)) && p != workspacePath {
				logger.WarnCF("exec", "Command blocked by path restriction", map[string]interface{}{
					"command":   command,
					"path":      p,
					"raw":       raw,
					"workspace": workspacePath,
				})
				return "Command blocked by safety guard (path outside working dir)"
			}
		}
	}

	return ""
}

func isWithinFreeTextFlagValue(command string, pathStart int) bool {
	for _, pattern := range freeTextFlagValuePatterns {
		matches := pattern.FindAllStringSubmatchIndex(command, -1)
		for _, match := range matches {
			if len(match) < 4 {
				continue
			}
			valueStart := match[2]
			valueEnd := match[3]
			if pathStart >= valueStart && pathStart < valueEnd {
				return true
			}
		}
	}
	return false
}

func (t *ExecTool) SetTimeout(timeout time.Duration) {
	t.timeout = timeout
}

func (t *ExecTool) SetRestrictToWorkspace(restrict bool) {
	t.restrictToWorkspace = restrict
}

func (t *ExecTool) SetAllowPatterns(patterns []string) error {
	t.allowPatterns = make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return fmt.Errorf("invalid allow pattern %q: %w", p, err)
		}
		t.allowPatterns = append(t.allowPatterns, re)
	}
	return nil
}
