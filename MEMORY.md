# PicoClaw Project Memory

## Project Info
- **Repo:** https://github.com/sipeed/picoclaw
- **Branch:** `bot` (NE main!)
- **Jazyk:** Go

## Deployment (CRITICAL!!!)
- **JÁ:** commit + push na `bot`
- **USER:** restart, install, deploy (NIKDY NE dělat já!)
- Po code review APPROVED → commit → **ČEKÁM NA USER RESTART**

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
- FIX: escape před truncate (`128ea8f`)

## Lekce
- **Telegram HTML** = `<s>text</s>`, NE `~~text~~` (markdown)
- Regex pro multiline = `(?s)` flag (DOTALL)
- **Escape PŘED truncate** - ne naopak!
- Vždy testy na edge cases

---

*Updated: 2026-02-19*
