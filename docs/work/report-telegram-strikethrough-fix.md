# Task Report: Fix /status Telegram Strikethrough Error

## Problem
```
Bad Request: can't parse entities: Can't find end of Strikethrough entity at byte offset 262
```

## Root Cause
In `/status` command (`pkg/channels/telegram_commands.go:Status()`):
```go
// BUGGY ORDER:
taskPreview := truncateStr(t.Task, 30)  // Step 1: truncate (may break ~~)
sb.WriteString(fmt.Sprintf("  Task: %s\n", escapeMD(taskPreview)))  // Step 2: escape (too late!)
```

When `t.Task` contains `~~text~~` and truncation cuts INSIDE the strikethrough markers:
- Input: `"Fix bug ~~deprecated code~~ in module"`
- After truncate(15): `"Fix bug ~~deprec..."`
- Result: Unclosed `~~` → Telegram parse error!

## Fix
```go
// CORRECT ORDER:
taskPreview := truncateStr(escapeMD(t.Task), 30)  // Escape FIRST, then truncate
sb.WriteString(fmt.Sprintf("  Task: %s\n", taskPreview))
```

Now `~~` becomes `\~\~` before truncation, so it can never be broken.

## Files Changed
- `pkg/channels/telegram_commands.go` - 3 lines

## Verification
- `go build` ✅
- `go fmt ./...` ✅
- `go vet ./...` ✅
- Manual test with strikethrough text ✅

## Deployment
- Committed: `0e618c5`
- Pushed to: `origin/bot`
- Production binary rebuilt: `/home/nanobot/picoclaw-production/picoclaw`
- **⚠️ REQUIRES MANUAL RESTART** - picoclaw running in terminal pts/6

## Next Steps
User must restart picoclaw process to apply the fix.
