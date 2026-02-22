# TODO for picoclaw agents system

## JEDEN VELKY REFACTOR (udelat najednou, ne po kouskach)

### REF-4: Upstream parity (active in full REF)

- [NOW] proverit upstream patch pro `max_completion_tokens` u GPT-5 v `pkg/providers/http_provider.go`
  - upstream referencni commit: `bb0424e`
- [NOW] zvazit `channel session key routing` metadata (`peer_kind`, `peer_id`) pro non-telegram kanaly
  - upstream referencni commit: `4adafa8`

## HARD RULE PRO NOVOU VERZI

- [MUST] Bez zpetne kompatibility: cte se jen nove schema/logy/API; zadne fallback parsery a zadne migracni vetve pro stare formaty.

## DEFINITION OF DONE (pro "hotovy dashboard bez telegramu")

- [ ] Cely provoz je ovladatelny z dashboardu bez Telegramu.
- [ ] Feed v dashboardu odpovida tomu, co by prislo do channelu + explicitni SYS/AUTO eventy. TY eventy jsou doufam na vsech aktivnich chanels
- [ ] Zadny pravidelny full repaint; pouze incremental patch.
- [ ] Timeline ordering je stabilni a reprodukovatelny.
- [ ] Kazdy agent ma vlastni log stream a rychle API nacteni.
- [ ] `make vet` + `make test` prochazi.
- [ ] CHANGELOG + README + VERSION + TODO aktualizovane ve stejnem passu.


## BACKLOG ZACHOVAN (udelat potom)

### LLM

- [LATER] LLM autotune context/retry podle chyb - pro konretniho provedera a model separatne
- [LATER] autotune max context podle error
- [LATER] autotune retry podle error
- [LATER] barvicky ve WEBUI agentu, aby bylo videt retry
- [LATER] proverit stream mode z.ai: https://docs.z.ai/guides/capabilities/streaming

### Memory tooling (future)

- [LATER] zavedeni `memory_*` toolu misto ad-hoc write/read
  - `memory_append(name, note, tags?)`
  - `memory_search(name, query, limit?)`
  - `memory_consolidate(name)` pro slouceni denich poznamek do dlouhodobe memory
- [LATER] idle worker pro memory maintenance
  - periodicky spoustet `memory_consolidate` jen pri idle po nastavitejne dobe idle a v povolenem casovem rozsahu (default idle 30m v case 02-04h)
  - detekce duplicit a sumarizace starsich zaznamu
  - zachovat audit trail (co bylo slouceno a kdy)
  - zamezit opakovanemu consolidate po nastavidelne dobe

### Upstream backlog (2026-02-20)

- [LATER] legacy config migrace bez `agents.defaults.provider` (jen pokud budeme chtit zpetnou kompatibilitu)
  - upstream referencni commit: `58b5e21`

### Config parity

- [LATER] sjednotit datovou strukturu `agents.defaults` a `agents.subagents` (plus `agents.<name>` override) do plne parity
  - subagent runtime ma mit stejne konfigurovatelne limity jako main agent (timeout/retries/backoff/context apod.)
  - zavest jeden resolver/runtime profil pro main i subagent beh
  - `config/config.example.json` drzet 1:1 se skutecnym runtime chovanim po dodelani parity

### Ideas

- [LATER] komunikace primo s agentem
  - injekce message agentovi
  - moznost ho spustit primo na channel a komunikovat s nim (`+agent_chat PROMPT`, ukonceni po konci nebo `+kill`)
- [LATER] project panel ve webui
- [LATER] Multi-dashboard pro vice picoclaw instanci.
- [LATER] Memory tooling (`memory_append/search/consolidate`) + idle maintenance.
- [LATER] Workspace file manager ve web UI (+ pozdeji editace).
