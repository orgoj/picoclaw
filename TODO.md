# TODO for picoclaw agents system

## WEBUI

- kliknu na agenta a udela se mi tam duplicine
- potrebuji videt so dostane jako ukol agent pri startu
- kliknu na egenta a  vleho ma nejaky filtrovany telegram a ne jeho log!!!
  - kdyz kurva kliknu na agenta tak vlevo musim videt jeho sesssion a kurva nic jineho!

### LLM

- [LATER] LLM autotune context/retry podle chyb - pro konretniho provedera a model separatne
- [LATER] autotune max context podle error
- [LATER] autotune retry podle error
- [LATER] barvicky ve WEBUI agentu, aby bylo videt retry
- [LATER] proverit stream mode z.ai: https://docs.z.ai/guides/capabilities/streaming

### Config parity

- [LATER] legacy config migrace bez `agents.defaults.provider` (jen pokud budeme chtit zpetnou kompatibilitu), toto je co?
  - upstream referencni commit: `58b5e21`

- [LATER] sjednotit datovou strukturu `agents.defaults` a `agents.subagents` (plus `agents.<name>` override) do plne parity
  - subagent runtime ma mit stejne konfigurovatelne limity jako main agent (timeout/retries/backoff/context apod.)
  - zavest jeden resolver/runtime profil pro main i subagent beh
  - `config/config.example.json` drzet 1:1 se skutecnym runtime chovanim po dodelani parity

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

### Ideas

- [LATER] komunikace primo s agentem
  - injekce message agentovi
  - moznost ho spustit primo na channel a komunikovat s nim (`+agent_chat PROMPT`, ukonceni po konci nebo `+kill`)
- [LATER] project panel ve webui
- [LATER] Multi-dashboard pro vice picoclaw instanci.
- [LATER] Memory tooling (`memory_append/search/consolidate`) + idle maintenance.
- [LATER] Workspace file manager ve web UI (+ pozdeji editace).

### Analyze for inspiration

- https://github.com/logiccrafterdz/Droidclaw
- https://github.com/sker65/picoclaw/tree/webui
-
