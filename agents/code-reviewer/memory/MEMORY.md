# Code Reviewer Memory

## Long-term Learnings

### PicoClaw Project Context
- **Stack**: Go, clean architecture
- **Build**: `make build`, `make test`
- **Key packages**: `pkg/channels/`, `pkg/tools/`, `pkg/agent/`

### Regex Patterns in PicoClaw
- `(?s)` DOTALL flag used in: `pkg/channels/telegram.go`, `pkg/skills/loader.go`, `pkg/tools/agent_registry.go`
- Standard pattern for multiline content: `(?s)delimiter(.+?)delimiter`

### Telegram Channel
- `markdownToTelegramHTML()` converts markdown to Telegram HTML
- HTML escaping happens BEFORE markdown replacements to prevent XSS
- Code blocks extracted before escaping, re-inserted after

---

*Last updated: 2025-01-19*
