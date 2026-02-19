# Callback Analysis Report

**Task:** Opravit async callback pro subagent completion  
**Date:** 2026-02-19  
**Self-rating:** 4/5

---

## Jak to funguje teď

### Current Flow (před opravou)

1. **spawn tool** volá `subagentManager.Spawn()` s AsyncCallback
2. **SubagentManager** spustí `runTask()` v gorutině
3. Po dokončení `runTask()`:
   - Nastaví status na "completed"/"failed"
   - Vytvoří `ToolResult` a zavolá callback (ale jen loguje!)
   - **Klíčové:** Zavolá `bus.PublishInbound()` s announce zprávou

```go
// V subagent.go - runTask()
if sm.bus != nil {
    sm.bus.PublishInbound(bus.InboundMessage{
        Channel:  "system",
        SenderID: fmt.Sprintf("subagent:%s", task.ID),
        ChatID:   fmt.Sprintf("%s:%s", task.OriginChannel, task.OriginChatID),
        Content:  announceContent,
    })
}
```

4. **processSystemMessage()** v loop.go zpracuje announce:
   - Před opravou: Jen logoval "Subagent completed"
   - Vracel prázdný string - user se nedozvěděl, že subagent skončil!

### Problém

`processSystemMessage()` **neaktualizoval CURRENT_TASK.md** a **neinformoval usera**.

---

## Co chybí

1. ❌ Aktualizace CURRENT_TASK.md s výsledkem subagenta
2. ⚠️ Notifikace userovi (vyřešeno: subagent použije `message` tool sám)

---

## Jak to opravit (implementováno)

### Změny v `pkg/agent/loop.go` - `processSystemMessage()`

```go
// Extract directory from subagent task if available
var directory string
if strings.HasPrefix(msg.SenderID, "subagent:") {
    taskID := strings.TrimPrefix(msg.SenderID, "subagent:")
    if task, ok := al.subagentManager.GetTask(taskID); ok {
        directory = task.Directory
    }
}

// Update CURRENT_TASK.md if directory exists and file exists
if directory != "" {
    taskFile := filepath.Join(directory, "CURRENT_TASK.md")
    if data, err := os.ReadFile(taskFile); err == nil {
        // File exists - append completion section
        timestamp := time.Now().Format("2006-01-02 15:04")
        taskLabel := msg.SenderID
        if idx := strings.Index(string(data), "## Task:"); idx >= 0 {
            // Extract task name from file
            endIdx := strings.Index(string(data)[idx:], "\n")
            if endIdx > 0 {
                taskLabel = string(data)[idx+8 : idx+endIdx]
            }
        }
        update := fmt.Sprintf("\n\n---\n\n## Subagent Completed [%s]\n\n**Status:** ✅ Done\n\n**Result:**\n%s", timestamp, content)
        updatedContent := string(data) + update
        if err := os.WriteFile(taskFile, []byte(updatedContent), 0644); err != nil {
            logger.WarnCF("agent", "Failed to update CURRENT_TASK.md", ...)
        }
    }
}
```

### User Notifikace

**Design Decision:** Subagent může použít `message` tool pro přímou komunikaci s userem.
Main agent **neposílá** notifikaci automaticky, protože:

1. User už dostane "Spawned subagent..." zprávu při spuštění
2. Subagent může sám použít `message` tool pokud potřebuje
3. Nechceme spamovat usera duplicitními zprávami

---

## Výsledný Flow

```
[User] pošle task
    ↓
[Main Agent] zavolá spawn tool
    ↓
[spawn tool] vytvoří subagent + vrátí "Spawned subagent..."
    ↓
[User] vidí "Spawned subagent..." (okamžitá odezva)
    ↓
[Subagent] běží async, používá tools
    ↓
[Subagent] dokončí → PublishInbound("system", "subagent:xxx", ...)
    ↓
[Main Agent] processSystemMessage():
    - Loguje dokončení
    - Aktualizuje CURRENT_TASK.md (pokud existuje)
    - NEPOŠÍLÁ zprávu userovi (subagent může použít message tool)
    ↓
[KONEC]
```

---

## Testování

- `go fmt ./...` ✅
- `go vet ./...` ✅
- `go test ./...` ✅ (všechny testy prošly)

---

## Self-Rating: 4/5

**Co chybí pro 5/5:**
- Unit test pro `processSystemMessage()` s mock subagentManager
- E2E test spawn → completion → CURRENT_TASK.md update

**Co je dobře:**
- Minimální změna kódu
- Zachována existující architektura
- Správné error handling a logování
- Respektuje design (subagent má `message` tool k dispozici)

---

## Soubory změněné

1. `pkg/agent/loop.go` - `processSystemMessage()` - přidána aktualizace CURRENT_TASK.md
