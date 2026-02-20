# Session Diary

**Date**: 2026-02-17 15:15
**Agent**: Pi-Agent (Antigravity)
**Project**: /home/michael/projects/picoclaw

## Task Summary
Refinement of PicoClaw stability, security, and splitting Z.AI MCP endpoints.

## Work Done
- **Subagent Management & Security**: Finalized registration of `subagent_*` tools and improved `exec` security to allow safe `/dev/` devices and URL patterns.
- **Logging Refactoring**: Extrahoval logovací inicializaci do helperu `setupLogging`, opravil ošetření chyb (home dir, mkdir) a nastavil logging jako výchozí `false`.
- **MCP Endpoint Splitting**: Rozdělení Z.AI na dva nezávislé endpointy: `endpoint` pro vyhledávání a `endpoint_fetch` pro stahování stránek. Odstraněn fallback pro striktní oddělení služeb.
- **Verification**: Proveden kompletní `make fmt`, `make vet` a `make test`.

## Mistakes & Corrections (CRITICAL)
### Where I Made Errors:
- **DNS Error Misinterpretation**: Původně jsem chybu `no such host` pro Z.AI endpoint identifikoval jako následek mých změn, přičemž šlo o externí výpadek DNS, který se jen stal viditelným díky lepším logům.
- **Implicit Fallback**: V první verzi rozdělení endpointů jsem nechal automatický fallback (pokud fetch chybí, použije se search), což bylo v rozporu s požadavkem na striktní oddělení dvou různých služeb.

### What Caused the Mistakes:
- **Design Over-engineering**: Snaha o "blbuvzdornost" (fallback) narazila na specifický use-case uživatele, který potřeboval mít služby striktně oddělené.

## Lessons Learned
### Technical:
- **MCP Versatility**: Uživatelé mohou používat různé MCP servery pro různé nástroje (search vs fetch). Architektura agenta na to musí být připravena (samostatní klienti).
- **DNS Visibility**: Lepší logování inicializace (včetně adresy endpointu) výrazně usnadňuje diagnostiku síťových chyb.

### Process:
- **Strict Adherence to Requirements**: Pokud uživatel žádá o rozdělení endpointů, nepředpokládat, že chce "zálohu" v podobě fallbacku.

## Skills Used
| Skill | Issue/Observation | Action |
|-------|-------------------|--------|
| commit | Splitting MCP endpoints | Detailed commit with strict separation note. |
| agents-diary | Session Wrap-up | Documenting the DNS discovery and endpoint logic. |
| edit | Config & Agent loop | Precision edits to separate clients. |
