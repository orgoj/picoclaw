# Report: Max Concurrent Subagents Limit Implementation

**Date**: 2025-01-21
**Task**: Implement configurable limit on concurrent subagents

## Summary

Implemented a `max_concurrent_subagents` configuration option to limit the number of simultaneously running subagents. This is necessary because the ZAI API limits concurrent sessions (currently 3 = 1 main + 2 subagents).

## Changes Made

### 1. Config Changes (`pkg/config/config.go`)

- Added `MaxConcurrentSubagents int` field to `AgentDefaults` struct
- Default value: **2**
- Supports environment variable override: `PICOCLAW_AGENTS_DEFAULTS_MAX_CONCURRENT_SUBAGENTS`

```go
type AgentDefaults struct {
    // ... existing fields ...
    MaxConcurrentSubagents    int     `json:"max_concurrent_subagents" env:"PICOCLAW_AGENTS_DEFAULTS_MAX_CONCURRENT_SUBAGENTS"`
    // ... remaining fields ...
}
```

### 2. Subagent Manager (`pkg/tools/subagent.go`)

- Added `maxConcurrentSubagents int` field to `SubagentManager` struct
- Added `countRunningTasks()` helper method to count running subagents
- Modified `Spawn()` method to check limit before spawning new subagents
- Returns clear error message when limit is reached:

```
maximum concurrent subagents limit reached (2/2 running). Please wait for existing subagents to complete
```

### 3. Tests (`pkg/config/config_test.go`)

Added two new tests:
- `TestDefaultConfig_MaxConcurrentSubagents` - verifies default value is 2
- `TestDefaultConfig_MaxConcurrentSubagents_Parsing` - verifies config parsing

All tests pass:
```
=== RUN   TestDefaultConfig_MaxConcurrentSubagents
--- PASS: TestDefaultConfig_MaxConcurrentSubagents (0.00s)
=== RUN   TestDefaultConfig_MaxConcurrentSubagents_Parsing
--- PASS: TestDefaultConfig_MaxConcurrentSubagents_Parsing (0.00s)
PASS
```

### 4. Documentation Updates

#### README.md
Updated two configuration tables to include:
```
| `max_concurrent_subagents` | 2 | Max number of subagents that can run simultaneously |
```

#### config/config.example.json
Added the new option to the example config:
```json
{
  "agents": {
    "defaults": {
      "max_concurrent_subagents": 2,
      // ... other options
    }
  }
}
```

## Usage

### Configuration

In `~/.picoclaw/config.json`:
```json
{
  "agents": {
    "defaults": {
      "max_concurrent_subagents": 2
    }
  }
}
```

### Environment Variable

```bash
export PICOCLAW_AGENTS_DEFAULTS_MAX_CONCURRENT_SUBAGENTS=2
```

## Behavior

When the limit is reached and a new subagent is spawned:

1. The `Spawn()` method checks the count of currently running subagents
2. If count >= `max_concurrent_subagents`, the spawn is rejected
3. Clear error message is returned to the caller
4. The main agent can inform the user to wait for existing tasks to complete

## Files Modified

1. `pkg/config/config.go` - Added field and default value
2. `pkg/tools/subagent.go` - Added limit check logic
3. `pkg/config/config_test.go` - Added tests
4. `README.md` - Updated documentation (2 tables + 1 example)
5. `config/config.example.json` - Added example config entry

## Verification

```bash
# Run tests
go test ./pkg/config/... -v

# Build
go build ./...
```

Both pass successfully.
