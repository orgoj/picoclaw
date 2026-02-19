# Picoclaw Self-Update Agent Memory

## Purpose
Standardní workflow pro úpravy a aktualizace picoclaw kódu.

## Key Learnings

### 2026-02-18: Regex Greedy vs Non-Greedy in Go
- **Problem**: Greedy regex `(.*)` captured too much content in YAML frontmatter extraction
- **Solution**: Changed to non-greedy `(.*?)` to capture only the first matching section
- **Impact**: Fixed invalid skill name warnings caused by code examples being parsed as metadata
- **Location**: `pkg/skills/loader.go:extractFrontmatter()`

### 2026-02-19: Telegram MarkdownV2 Truncation Order Bug
- **Problem**: `/status` command crashed with "Can't find end of Strikethrough entity"
- **Root Cause**: `truncateStr()` called BEFORE `escapeMD()` could cut text inside `~~` markers
- **Example**: `"Fix bug ~~deprecated~~..." → truncate → `"Fix bug ~~deprec..."` → broken!
- **Fix**: Always escape BEFORE truncate: `truncateStr(escapeMD(text), maxLen)`
- **Location**: `pkg/channels/telegram_commands.go:Status()`
- **Rule**: When combining truncation + escaping, ALWAYS escape first to prevent breaking entities


### 2026-02-19: Truncation Breaking Markdown Entities
- **Problem**: `escapeMD(truncateStr(x))` order caused strikethrough error
- **Impact**: "Can't find end of Strikethrough entity at byte offset 262" in /status
- **Root Cause**: Truncation cut text in middle of `~~strikethrough~~`, leaving unclosed syntax
- **Solution**: Always `truncateStr(escapeMD(x))` - escape FIRST, then truncate
- **Location**: `pkg/channels/telegram_commands.go:205` (GetFirstQueuedMessage display)
- **Rule**: When escaping + truncating, ALWAYS escape first to preserve entity boundaries
