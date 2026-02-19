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
