# TODO for picoclaw agents system

## Povedomi o case a message

cil je aby bot chapal casove souvisloti a mohl se podle toho chovat
- aby vedel je dlouho se flaka a mohl pustit nejake memory clean terba po 30 minutach flakani
- aby vedel ze uzivatel tuto zpravu napsal o 3 hodiny po predchozi a kdyz se pta na nejaky stav tak uz je to jinak a musi ho znova zjistovat

- [ ] IDELE metrick
  - pocitat kolik bylo bessage idle v rade za sbou
  - pridavat pocitadlo a cas zacatklu a aktualini cas jako nejaka metadata to idle message (aby na ne podle promptu mohl reagovat)
  - nuloat pocitadlo pri message z chanel od usera
- [ ] k message od usera pridavat casova  metadata (kdy message prisla do fronty)
  - konfigurovatelna hodnota pro minimalni casovy rozestup od predchozi message (default 10minut)
  - pokudd je prekrocena prida pak pri injekci message botovy do proptu metadata s upozornemim jaky cas byl od posledni message)
- [ ] pomohlo by pridava metadata s informaci o poctu message ve fronte pri injekci zpravy ? asi configurovatelne a zkusime to
- [ ] konfigurovatelne, jestli zpravy z queu dostava pri injekci po jedne a nebo najedno
  - to by mozna chtelo i mit moznost to nejak ovlivnit pri psani zpravy (mozna kdyz bude zacinat + ?)
  - takove zpravy by se concatenovaly automaticky bez ohledu na nastaveni

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
