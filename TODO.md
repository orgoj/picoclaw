# TODO for picoclaw agents system

## FIX

- (none)

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

## Config parity

- [ ] sjednotit datovou strukturu `agents.defaults` a `agents.subagents` (plus `agents.<name>` override) do plne parity
  - subagent runtime ma mit stejne konfigurovatelne limity jako main agent (timeout/retries/backoff/context apod.)
  - zavest jeden resolver/runtime profil pro main i subagent beh
  - `config/config.example.json` drzet 1:1 se skutecnym runtime chovanim po dodelani parity
