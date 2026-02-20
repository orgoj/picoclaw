package tools

import (
	"context"
	"path/filepath"
)

type workingDirectoryContextKey struct{}

var workingDirectoryKey workingDirectoryContextKey

// WithWorkingDirectory stores the effective working directory for tool calls.
// Relative directory values are resolved against workspace when provided.
func WithWorkingDirectory(ctx context.Context, workspace, directory string) context.Context {
	if directory == "" {
		return ctx
	}

	resolved := directory
	if !filepath.IsAbs(resolved) && workspace != "" {
		resolved = filepath.Join(workspace, resolved)
	}

	absDir, err := filepath.Abs(resolved)
	if err != nil {
		return ctx
	}

	return context.WithValue(ctx, workingDirectoryKey, filepath.Clean(absDir))
}

func extractWorkingDirectory(ctx context.Context) (string, bool) {
	val := ctx.Value(workingDirectoryKey)
	dir, ok := val.(string)
	if !ok || dir == "" {
		return "", false
	}
	return dir, true
}

func resolvePathFromContext(ctx context.Context, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	if dir, ok := extractWorkingDirectory(ctx); ok {
		return filepath.Join(dir, path)
	}
	return path
}
