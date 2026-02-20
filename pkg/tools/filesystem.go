package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// validatePath ensures the given path is within the workspace if restrict is true.
func validatePath(path, workspace string, restrict bool, denyPatterns []string) (string, error) {
	if workspace == "" {
		cleanPath := filepath.Clean(path)
		if matched, pattern := isPathDenied(cleanPath, "", denyPatterns); matched {
			return "", fmt.Errorf("access denied: path matches denied pattern %q", pattern)
		}
		return cleanPath, nil
	}

	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace path: %w", err)
	}

	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath, err = filepath.Abs(filepath.Join(absWorkspace, path))
		if err != nil {
			return "", fmt.Errorf("failed to resolve file path: %w", err)
		}
	}

	if restrict && !strings.HasPrefix(absPath, absWorkspace) {
		return "", fmt.Errorf("access denied: path is outside the workspace")
	}
	if matched, pattern := isPathDenied(absPath, absWorkspace, denyPatterns); matched {
		return "", fmt.Errorf("access denied: path matches denied pattern %q", pattern)
	}

	return absPath, nil
}

func isPathDenied(absPath, workspace string, patterns []string) (bool, string) {
	if len(patterns) == 0 {
		return false, ""
	}

	absSlash := filepath.ToSlash(filepath.Clean(absPath))
	candidates := []string{
		absSlash,
		strings.TrimPrefix(absSlash, "/"),
	}

	if workspace != "" {
		rel, err := filepath.Rel(workspace, absPath)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			relSlash := filepath.ToSlash(rel)
			candidates = append(candidates, relSlash)
			if relSlash != "." {
				candidates = append(candidates, "./"+relSlash)
			}
		}
	}

	for _, pattern := range patterns {
		normalizedPattern := strings.TrimSpace(filepath.ToSlash(pattern))
		if normalizedPattern == "" {
			continue
		}
		for _, candidate := range candidates {
			if globMatchPath(normalizedPattern, candidate) {
				return true, pattern
			}
		}
	}

	return false, ""
}

func globMatchPath(pattern, candidate string) bool {
	rePattern := strings.Builder{}
	rePattern.Grow(len(pattern) * 2)
	rePattern.WriteString("^")

	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		if ch == '*' {
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				rePattern.WriteString(".*")
				i++
				continue
			}
			rePattern.WriteString(`[^/]*`)
			continue
		}
		if ch == '?' {
			rePattern.WriteString(`[^/]`)
			continue
		}
		if strings.ContainsRune(`.+()|[]{}^$\\`, rune(ch)) {
			rePattern.WriteByte('\\')
		}
		rePattern.WriteByte(ch)
	}
	rePattern.WriteString("$")

	re := regexp.MustCompile(rePattern.String())
	if re.MatchString(candidate) {
		return true
	}
	// Allow patterns ending with "/**" to match the directory itself.
	if strings.HasSuffix(pattern, "/**") {
		return re.MatchString(candidate + "/")
	}
	return false
}

type ReadFileTool struct {
	workspace        string
	restrict         bool
	denyPathPatterns []string
}

func NewReadFileTool(workspace string, restrict bool, denyPatterns ...string) *ReadFileTool {
	return &ReadFileTool{workspace: workspace, restrict: restrict, denyPathPatterns: denyPatterns}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Read the contents of a file"
}

func (t *ReadFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to read",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return ErrorResult("path is required")
	}

	resolvedPath, err := validatePath(path, t.workspace, t.restrict, t.denyPathPatterns)
	if err != nil {
		return ErrorResult(err.Error())
	}

	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read file: %v", err))
	}

	return NewToolResult(string(content))
}

type WriteFileTool struct {
	workspace        string
	restrict         bool
	denyPathPatterns []string
}

func NewWriteFileTool(workspace string, restrict bool, denyPatterns ...string) *WriteFileTool {
	return &WriteFileTool{workspace: workspace, restrict: restrict, denyPathPatterns: denyPatterns}
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "Write content to a file"
}

func (t *WriteFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to write",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return ErrorResult("path is required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return ErrorResult("content is required")
	}

	resolvedPath, err := validatePath(path, t.workspace, t.restrict, t.denyPathPatterns)
	if err != nil {
		return ErrorResult(err.Error())
	}
	if err := validateWriteScope(ctx, resolvedPath); err != nil {
		return ErrorResult(err.Error())
	}

	dir := filepath.Dir(resolvedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ErrorResult(fmt.Sprintf("failed to create directory: %v", err))
	}

	if err := os.WriteFile(resolvedPath, []byte(content), 0644); err != nil {
		return ErrorResult(fmt.Sprintf("failed to write file: %v", err))
	}

	return SilentResult(fmt.Sprintf("File written: %s", path))
}

type ListDirTool struct {
	workspace        string
	restrict         bool
	denyPathPatterns []string
}

func NewListDirTool(workspace string, restrict bool, denyPatterns ...string) *ListDirTool {
	return &ListDirTool{workspace: workspace, restrict: restrict, denyPathPatterns: denyPatterns}
}

func (t *ListDirTool) Name() string {
	return "list_dir"
}

func (t *ListDirTool) Description() string {
	return "List files and directories in a path"
}

func (t *ListDirTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to list",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ListDirTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		path = "."
	}

	resolvedPath, err := validatePath(path, t.workspace, t.restrict, t.denyPathPatterns)
	if err != nil {
		return ErrorResult(err.Error())
	}

	entries, err := os.ReadDir(resolvedPath)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read directory: %v", err))
	}

	result := ""
	for _, entry := range entries {
		if entry.IsDir() {
			result += "DIR:  " + entry.Name() + "\n"
		} else {
			result += "FILE: " + entry.Name() + "\n"
		}
	}

	return NewToolResult(result)
}
