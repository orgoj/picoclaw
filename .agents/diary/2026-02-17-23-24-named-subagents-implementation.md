# Session Diary

**Date**: 2026-02-17 23:24
**Agent**: Claude-Code
**Project**: /home/michael/projects/picoclaw

## Task Summary

Implementace pojmenovaných subagentů s persistentní identitou a pamětí. Subagenty nyní přijímají volitelný parametr `name`, který načte jejich identitu a paměť z `workspace/agents/<name>/`. Hlavní agent dostane seznam dostupných agentů do systémového promptu.

## Work Done

- Přidán `Name string` field do `SubagentTask`
- `buildSubagentSystemPrompt()` rozšířen o načítání `AGENTS.md` (identita) + `MEMORY.md` (paměť) + instrukce k uložení
- Bezpečnostní validace: jména s `/\\.` jsou tiše ignorována (path traversal)
- `Spawn()` rozšířen o `name` parametr, zobrazuje `[agent: name]` v návratové zprávě
- Announce message do busu obsahuje `[agent: name]` pokud nastaven
- `subagent_status` a `subagent_history` zobrazují agent name
- `SubagentTool` a `SpawnTool` parametry rozšířeny o `name`
- Nový soubor `agent_registry.go`: `LoadAvailableAgents()` skenuje `workspace/agents/*/AGENTS.md` pro YAML frontmatter
- `context.go`: `buildNamedAgentsSummary()` přidána do `BuildSystemPrompt()` (za Skills, před Memory)
- Testy: check `name` parametru, `WithName`, `InvalidName`, registry testy
- README: sekce "Named Sub-agents" s formátem AGENTS.md frontmatteru
- CODEBASE-MAP.md aktualizována

## Mistakes & Corrections (CRITICAL)

### Where I Made Errors:
- Žádné zásadní chyby — implementace proběhla podle plánu

### What Caused the Mistakes:
- N/A

## Lessons Learned

### Technical:
- `write_file` v `filesystem.go` dělá `MkdirAll` automaticky — agent nemusí volat mkdir před prvním zápisem paměti
- `pkg/agent` importuje `pkg/tools` → `LoadAvailableAgents()` šlo dát do `pkg/tools` bez circular dependency
- YAML frontmatter parsing je sdílená logika — v `skills/loader.go` existuje identická implementace; hodí se mít ji i v `pkg/tools/agent_registry.go` jako standalone (žádná závislost na `pkg/skills`)
- Vzor skills (frontmatter → discovery → seznam v promptu) je dobrý vzor pro rozšiřování systému

### Process:
- Plán byl detailní a funkční — stačilo ho následovat krok po kroku
- Při implementaci vždy ověřit skutečné názvy toolů (`write_file`, `append_file`) před zápisem do promptů
- Kontrola existujících importů před přidáváním závislostí ušetří čas

## Skills Used

| Skill | Issue/Observation | Action |
|-------|-------------------|--------|
| commit | Hladce fungoval | - |
| agents-diary | Použit na závěr session | - |
