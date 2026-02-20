# TODO for picoclaw agents system

## FIX

- [ ] Nedošla mi message na kanály při stisku CTRL C. 
2026/02/20 09:21:08 [2026-02-20T08:21:08Z] [INFO] tool: Tool execution started {tool=subagent_status, args=map[]}
^C
Shutting down...
[Fri Feb 20 09:31:02 CET 2026] ERROR Execution error editMessageText: request call: context canceled
[Fri Feb 20 09:31:02 CET 2026] ERROR Execution error sendMessage: request call: context canceled
2026/02/20 09:31:02 [2026-02-20T08:31:02Z] [INFO] devices: Device event service stopped
2026/02/20 09:31:02 [2026-02-20T08:31:02Z] [ERROR] telegram: HTML parse failed, falling back to plain text {error=telego: sendMessage: internal execution: request call: context canceled}
[Fri Feb 20 09:31:02 CET 2026] ERROR Execution error sendMessage: request call: context canceled
2026/02/20 09:31:02 [2026-02-20T08:31:02Z] [ERROR] channels: Error sending message to channel {error=telego: sendMessage: internal execution: request call: context canceled, channel=telegram}
2026/02/20 09:31:02 [2026-02-20T08:31:02Z] [INFO] channels: Outbound dispatcher stopped
2026/02/20 09:31:02 [2026-02-20T08:31:02Z] [INFO] channels: Stopping all channels
2026/02/20 09:31:02 [2026-02-20T08:31:02Z] [INFO] channels: Stopping channel {channel=telegram}
2026/02/20 09:31:02 [2026-02-20T08:31:02Z] [INFO] telegram: Stopping Telegram bot...
2026/02/20 09:31:02 [2026-02-20T08:31:02Z] [INFO] channels: All channels stopped
✓ Gateway stopped
2026/02/20 09:31:11 [2026-02-20T08:31:11Z] [INFO] agent: ZAI Search MCP client connected {endpoint=https://api.z.ai/api/mcp/web_search_prime/mcp}
To se sakra naučí, jak má být formatovaná message pro telegramu, ale to už je zase chybávalo s posílání message na telegramu. Telegram. To máš mi zapsané v AGENTS.md. 

## Memory tooling (future)

- [ ] Zavedeni `memory_*` toolu misto ad-hoc write/read
  - `memory_append(name, note, tags?)`
  - `memory_search(name, query, limit?)`
  - `memory_consolidate(name)` pro slouceni dennich poznamek do dlouhodobe memory
- [ ] Idle worker pro memory maintenance
  - to by mel byt nejaky prompt, ktery by se poustel na memory dir kazdeho (sub)agenta
  - periodicky spoustet `memory_consolidate` jen pri idle
  - detekce duplicit a sumarizace starsich zaznamu
  - zachovat audit trail (co bylo slouceno a kdy)

## Upstream backlog (2026-02-20)

- [ ] proverit upstream patch pro `max_completion_tokens` u GPT-5 v `pkg/providers/http_provider.go`
  - upstream referencni commit: `bb0424e`
  - u nas uz existuje vetveni pro `glm`/`o1`; rozhodnout jestli rozsirit i o `gpt-5`
- [ ] zvazit `channel session key routing` metadata (`peer_kind`, `peer_id`) pro non-telegram kanaly
  - upstream referencni commit: `4adafa8`
  - overit prinos pro nase aktualni session/routing modely a DM/group oddeleni
- [ ] legacy config migrace bez `agents.defaults.provider` (jen kdyz budeme chtit zpetnou kompatibilitu)
  - upstream referencni commit: `58b5e21`
  - zatim low priority, pokud necilime na stare konfigurace
