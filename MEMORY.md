# PicoClaw Project Memory

## Project Info
- **Repo:** https://github.com/sipeed/picoclaw
- **Branch:** `bot` (NE main!)
- **Jazyk:** Go
- **Deploy:** User spouští ručně (NE PM2!)

## API Config
- **Model:** GLM-5
- **Max tokens:** 4000 (ne 4096!)
- **Concurrent limit:** 3 (main + 2 subagents)

## Stack
- Go 1.25+
- Clean architecture
- SQLite pro state

## Branch Workflow
1. Vždy pracovat na `bot`
2. Push na `origin bot`
3. Merge do main přes PR (manuálně)

## Commity
- P0-1: API Retry Logic (`f67fe85`)
- P1-1: /status enhancement (`7acf286`)
- P1-2: IDLE subagent status (`9f575ef`)
- FIX: Telegram strikethrough (`b238401`)

## Lekce
- **Telegram HTML** = `<s>text</s>`, NE `~~text~~` (markdown)
- Regex pro multiline = `(?s)` flag (DOTALL)
- Vždy testy na edge cases (multiline, empty, special chars)

---

*Updated: 2026-02-19*
