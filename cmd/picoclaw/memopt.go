package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/llm"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type memoptOptions struct {
	Workspace   string
	Apply       bool
	UseLLM      bool
	LLMMaxChars int
}

type memoptStats struct {
	FilesScanned int
	FilesChanged int
	BeforeTokens int
	AfterTokens  int
}

func memoptCmd() {
	for _, arg := range os.Args[2:] {
		if arg == "--help" || arg == "-h" {
			memoptHelp()
			return
		}
	}

	defaultWorkspace := ""
	cfg, cfgErr := loadConfig()
	if cfgErr == nil {
		defaultWorkspace = cfg.WorkspacePath()
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			defaultWorkspace = filepath.Join(home, ".picoclaw", "workspace")
		}
	}

	opts, err := parseMemoptArgs(defaultWorkspace, os.Args[2:])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		memoptHelp()
		os.Exit(1)
	}
	if opts.LLMMaxChars <= 0 {
		fmt.Println("Error: --llm-max-chars must be > 0")
		os.Exit(1)
	}

	fmt.Printf("%s memopt workspace: %s\n", logo, opts.Workspace)
	if opts.Apply {
		fmt.Println("Mode: apply")
	} else {
		fmt.Println("Mode: dry-run")
	}
	fmt.Printf("LLM pass: %t\n", opts.UseLLM)

	stats, err := runDeterministicMemopt(opts)
	if err != nil {
		fmt.Printf("Deterministic pass failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Deterministic: files=%d changed=%d tokens=%d->%d saved=%d\n",
		stats.FilesScanned, stats.FilesChanged, stats.BeforeTokens, stats.AfterTokens, stats.BeforeTokens-stats.AfterTokens)

	if !opts.UseLLM {
		return
	}
	if cfgErr != nil {
		fmt.Printf("LLM pass needs valid config (~/.picoclaw/config.json): %v\n", cfgErr)
		os.Exit(1)
	}
	if cfg == nil {
		fmt.Println("LLM pass needs valid config (~/.picoclaw/config.json)")
		os.Exit(1)
	}

	if err := runLLMMemopt(cfg, opts); err != nil {
		fmt.Printf("LLM pass failed: %v\n", err)
		os.Exit(1)
	}
}

func parseMemoptArgs(defaultWorkspace string, args []string) (memoptOptions, error) {
	opts := memoptOptions{
		Workspace:   defaultWorkspace,
		Apply:       false,
		UseLLM:      false,
		LLMMaxChars: 12000,
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--workspace":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--workspace requires a value")
			}
			opts.Workspace = strings.TrimSpace(args[i+1])
			i++
		case "--apply":
			opts.Apply = true
		case "--dry-run":
			opts.Apply = false
		case "--llm":
			opts.UseLLM = true
		case "--no-llm":
			opts.UseLLM = false
		case "--llm-max-chars":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--llm-max-chars requires a value")
			}
			var n int
			if _, err := fmt.Sscanf(args[i+1], "%d", &n); err != nil {
				return opts, fmt.Errorf("invalid --llm-max-chars: %v", err)
			}
			opts.LLMMaxChars = n
			i++
		default:
			return opts, fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	if strings.TrimSpace(opts.Workspace) == "" {
		return opts, fmt.Errorf("workspace cannot be empty")
	}
	return opts, nil
}

func memoptHelp() {
	fmt.Println("\nMemory optimization")
	fmt.Println()
	fmt.Println("Usage: picoclaw memopt [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --workspace PATH       Target workspace (default: config workspace)")
	fmt.Println("  --dry-run              Report only (default)")
	fmt.Println("  --apply                Write changes")
	fmt.Println("  --llm                  Enable optional LLM rewrite on MEMORY.md files")
	fmt.Println("  --no-llm               Disable LLM rewrite (default)")
	fmt.Println("  --llm-max-chars N      Max chars sent per MEMORY.md file to LLM (default: 12000)")
	fmt.Println()
	fmt.Println("Scope:")
	fmt.Println("  - <workspace>/memory/**/*.md")
	fmt.Println("  - <workspace>/agents/*/memory/**/*.md")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  picoclaw memopt --workspace ~/.picoclaw/workspace")
	fmt.Println("  picoclaw memopt --workspace ~/.picoclaw/workspace --apply")
	fmt.Println("  picoclaw memopt --workspace ~/.picoclaw/workspace --apply --llm")
}

func runDeterministicMemopt(opts memoptOptions) (memoptStats, error) {
	var stats memoptStats
	dirs, err := collectMemoryDirs(opts.Workspace)
	if err != nil {
		return stats, err
	}
	if len(dirs) == 0 {
		return stats, fmt.Errorf("no memory directories found under %s", opts.Workspace)
	}

	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".md" {
				return nil
			}

			originalBytes, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			original := string(originalBytes)
			optimized := deterministicCompressMarkdown(original)

			before := estimateTextTokens(original)
			after := estimateTextTokens(optimized)
			if before > 0 && after > before {
				optimized = original
				after = before
			}
			stats.BeforeTokens += before
			stats.AfterTokens += after
			stats.FilesScanned++

			if optimized != original {
				stats.FilesChanged++
				if opts.Apply {
					if err := os.WriteFile(path, []byte(optimized), 0644); err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			return stats, err
		}
	}

	return stats, nil
}

func collectMemoryDirs(workspace string) ([]string, error) {
	info, err := os.Stat(workspace)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("workspace is not a directory: %s", workspace)
	}

	dirs := make([]string, 0, 16)
	mainMemory := filepath.Join(workspace, "memory")
	if fi, err := os.Stat(mainMemory); err == nil && fi.IsDir() {
		dirs = append(dirs, mainMemory)
	}

	agentsRoot := filepath.Join(workspace, "agents")
	entries, err := os.ReadDir(agentsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return dirs, nil
		}
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		memDir := filepath.Join(agentsRoot, entry.Name(), "memory")
		if fi, err := os.Stat(memDir); err == nil && fi.IsDir() {
			dirs = append(dirs, memDir)
		}
	}
	return dirs, nil
}

func deterministicCompressMarkdown(input string) string {
	normalized := strings.ReplaceAll(input, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	out := make([]string, 0, len(lines))

	prevLine := ""
	blankCount := 0
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			blankCount++
			if blankCount > 1 {
				continue
			}
			out = append(out, "")
			prevLine = ""
			continue
		}
		blankCount = 0
		if line == prevLine {
			continue
		}
		out = append(out, line)
		prevLine = line
	}

	result := strings.Join(out, "\n")
	result = strings.TrimSpace(result)
	if result == "" {
		return ""
	}
	return result + "\n"
}

func estimateTextTokens(text string) int {
	if text == "" {
		return 0
	}
	return utf8.RuneCountInString(text) / 3
}

func runLLMMemopt(cfg *config.Config, opts memoptOptions) error {
	provider, err := providers.CreateProvider(cfg)
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}

	files := make([]string, 0, 16)
	mainMemory := filepath.Join(opts.Workspace, "memory", "MEMORY.md")
	if _, err := os.Stat(mainMemory); err == nil {
		files = append(files, mainMemory)
	}
	agentMemoryFiles, err := filepath.Glob(filepath.Join(opts.Workspace, "agents", "*", "memory", "MEMORY.md"))
	if err == nil {
		files = append(files, agentMemoryFiles...)
	}
	if len(files) == 0 {
		fmt.Println("LLM: no MEMORY.md files found.")
		return nil
	}

	for _, path := range files {
		originalBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		original := string(originalBytes)
		if strings.TrimSpace(original) == "" {
			continue
		}

		truncated := original
		if len(truncated) > opts.LLMMaxChars {
			truncated = truncated[:opts.LLMMaxChars]
		}

		prompt := fmt.Sprintf(`Rewrite this memory file to be concise and token-efficient while preserving all facts, decisions, constraints, paths, and unresolved TODOs.
Rules:
- Keep markdown structure.
- Do not invent information.
- Preserve names/dates/paths exactly.
- Remove redundancy and filler.
- Output only rewritten markdown.

FILE: %s
CONTENT:
%s`, path, truncated)

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Agents.Defaults.LLMTimeout)*time.Second)
		resp, err := llm.CallWithRetry(ctx, llm.CallConfig{
			Provider: provider,
			Messages: []providers.Message{{Role: "user", Content: prompt}},
			Model:    cfg.Agents.Defaults.Model,
			Options: map[string]any{
				"max_tokens":  cfg.Agents.Defaults.MaxTokens,
				"temperature": 0.2,
			},
			Retry: llm.RetryConfig{
				MaxRetries:              cfg.Agents.Defaults.LLMMaxRetries,
				RetryBackoffSeconds:     cfg.Agents.Defaults.LLMRetryBackoffSeconds,
				RetryMaxBackoffSeconds:  cfg.Agents.Defaults.LLMRetryMaxBackoff,
				RateLimitBackoffSeconds: cfg.Agents.Defaults.LLMRateLimitBackoff,
				RateLimitMaxBackoffSecs: cfg.Agents.Defaults.LLMRateLimitMaxBackoff,
				RetryMaxElapsedSeconds:  cfg.Agents.Defaults.LLMRetryMaxElapsed,
			},
			LogComponent: "memopt",
			LogFields: map[string]any{
				"file": path,
			},
		})
		cancel()
		if err != nil {
			return fmt.Errorf("llm rewrite %s: %w", path, err)
		}

		candidate := strings.TrimSpace(resp.Content)
		if candidate == "" {
			fmt.Printf("LLM: empty output, skipped %s\n", path)
			continue
		}
		candidate += "\n"

		before := estimateTextTokens(original)
		after := estimateTextTokens(candidate)
		if after > before {
			fmt.Printf("LLM: skipped %s (tokens %d -> %d)\n", path, before, after)
			continue
		}

		if opts.Apply {
			if err := os.WriteFile(path, []byte(candidate), 0644); err != nil {
				return err
			}
			fmt.Printf("LLM: applied %s (%d -> %d tokens)\n", path, before, after)
		} else {
			fmt.Printf("LLM: dry-run %s (%d -> %d tokens)\n", path, before, after)
		}
	}
	return nil
}
