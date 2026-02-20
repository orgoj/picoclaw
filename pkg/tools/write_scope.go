package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type writeScopeContextKey struct{}

var writeScopeKey writeScopeContextKey

// WithWriteScope stores allowed write roots for the current execution context.
func WithWriteScope(ctx context.Context, roots []string) context.Context {
	cleaned := make([]string, 0, len(roots))
	for _, root := range roots {
		if root == "" {
			continue
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		cleaned = append(cleaned, filepath.Clean(absRoot))
	}
	if len(cleaned) == 0 {
		return ctx
	}
	return context.WithValue(ctx, writeScopeKey, cleaned)
}

func extractWriteScope(ctx context.Context) ([]string, bool) {
	val := ctx.Value(writeScopeKey)
	roots, ok := val.([]string)
	if !ok || len(roots) == 0 {
		return nil, false
	}
	return roots, true
}

func validateWriteScope(ctx context.Context, absPath string) error {
	roots, ok := extractWriteScope(ctx)
	if !ok {
		return nil
	}

	// Prevent writing to a mirrored ".picoclaw/workspace/agents/*/memory" tree
	// nested under a project directory. Named-agent memory must always target the
	// canonical workspace root memory path.
	if isNestedWorkspaceAgentMemoryPath(absPath) {
		hasCanonicalMemoryRoot := false
		for _, root := range roots {
			if isAgentMemoryRoot(root) {
				hasCanonicalMemoryRoot = true
				if pathWithinRoot(absPath, root) {
					return nil
				}
			}
		}
		if hasCanonicalMemoryRoot {
			return fmt.Errorf("write scope denied: %s is not under canonical agent memory root", absPath)
		}
	}

	for _, root := range roots {
		if pathWithinRoot(absPath, root) {
			return nil
		}
	}

	return fmt.Errorf("write scope denied: %s is outside allowed roots (%s)", absPath, strings.Join(roots, ", "))
}

func pathWithinRoot(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}

func isNestedWorkspaceAgentMemoryPath(path string) bool {
	p := filepath.ToSlash(filepath.Clean(path))
	return strings.Contains(p, "/.picoclaw/workspace/agents/") && strings.Contains(p, "/memory/")
}

func isAgentMemoryRoot(root string) bool {
	r := filepath.ToSlash(filepath.Clean(root))
	parts := strings.Split(r, "/")
	if len(parts) < 3 {
		return false
	}
	n := len(parts)
	return parts[n-3] == "agents" && parts[n-2] != "" && parts[n-1] == "memory"
}
