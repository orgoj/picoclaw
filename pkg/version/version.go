// Package version provides build version information
package version

import "runtime"

var (
	// Version is the semantic version (set via ldflags)
	Version = "dev"
	// GitCommit is the short git commit hash (set via ldflags)
	GitCommit = ""
	// BuildTime is the build timestamp (set via ldflags)
	BuildTime = ""
	// GoVersion is the Go version used to build (set via ldflags)
	GoVersion = ""
)

// Set initializes version info (called from main)
func Set(v, commit, build, gov string) {
	Version = v
	GitCommit = commit
	BuildTime = build
	GoVersion = gov
}

// Format returns version with optional git commit
func Format() string {
	if GitCommit != "" {
		return Version + " (" + GitCommit + ")"
	}
	return Version
}

// GetGoVersion returns Go version (falls back to runtime.Version)
func GetGoVersion() string {
	if GoVersion != "" {
		return GoVersion
	}
	return runtime.Version()
}
