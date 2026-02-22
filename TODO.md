# TODO for picoclaw agents system

## FIX

### WEBUI

- info v head o hlavnim agentovi, doba behu, stav contextu, pocet message a tool call, celkove a od compactu
- filter na history abych mohl skyt rychle debug a info
- ta history je spatna - to musi byt casove jak to slo za sebou (ted je best-effort, doplnit 100% timeline i pro starsi data bez timestamp)
- to web ui se asi porad prekresluje, nejde ani oznacit text na copy, kurva to musi menit jen zmeny a ne cele predrbavat porad
- u agenta potrebuji videt cas spusteni a cas konce, pocet message, pocet toolcall, context size aktualni. a co je tam pending u neho sama nula 0 proc to tam je?
- injekce message agentovi

- asi nejak celkove review, oprava UX, toto musi byt pouzitelne jako primarni ovladaci panel, abych to rozjel i bez telegram a toto byl primarni chanell

- pres to web ui by melo by dostupny i cely workspace file manager (do budoucna i editace file)


## LLM

- autotune max context podle error
- autotune restry podle error
- barvicky ve WEBUI agentu, abych videl ze je retry 


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

## IDEAS

- komunikace primo s agentem
  - injekce message agentovi
  - moznost ho spustit primo na chanell a komunikovat s nim, nejake `+agent_chat PROMPT` by ho pustilo  a vse co pise davalo na chanell, a konec kdyz skonci a nebo `+kill`

- multi dashboard vice picoclaw pres API
- project panel ve webui
