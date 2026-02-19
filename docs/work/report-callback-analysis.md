# Callback Analysis Report

**Task:** Opravit async callback pro subagent completion  
**Date:** 2026-02-19  
**Status:** ANALÝZA DOKONČENA (čeká na implementaci)
**Self-rating:** 4/5

---

## Jak to funguje teď

### Current Flow

1. **spawn tool** volá `subagentManager.Spawn()` s AsyncCallback
2. **SubagentManager** spustí `runTask()` v gorutině
3. Po dokončení `runTask()`:
   - Nastaví status na "completed"/"failed"
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
   - Momentálně: Jen loguje "Subagent completed"
   - **PROBLÉM:** Vrací prázdný string `""`
   - **VÝSLEDEK:** Main agent NEVÍ že subagent skončil!

### Problém

`processSystemMessage()` vrací `""` → main agent nemá žádnou informaci o dokončení.

---

## Co chybí

1. ❌ Main agent se nedozví o dokončení subagenta
2. ⚠️ User notifikace (subagent má `message` tool)

---

## Řešení (NENAVRŽENÁ IMPLEMENTACE)

### Správné řešení: VRÁTIT content

```go
func (al *AgentLoop) processSystemMessage(ctx context.Context, msg bus.InboundMessage) (string, error) {
    // ... existing code ...
    
    // VRÁTIT content, ne ""
    return fmt.Sprintf("📢 Subagent %s completed:\n\n%s", msg.SenderID, content), nil
}
```

**Proč tohle řešení:**
- Main agent UVIDÍ zprávu "Subagent X completed"
- Může se ROZHODNOUT co s ní (zapnout do historie, ignorovat, poslat userovi)
- Neautomatické chování - flexibilita

---

## ŠPATNÉ řešení (zamítnuto)

```go
// ❌ ŠPATNĚ: Automaticky append do CURRENT_TASK.md
if directory != "" {
    taskFile := filepath.Join(directory, "CURRENT_TASK.md")
    // ... append automaticky ...
}
return "", nil  // ❌ Stále vrací prázdný string
```

**Proč je to špatně:**
- Hardcoded chování
- Main agent nemá kontrolu
- Neřeší základní problém (main agent neví o dokončení)

---

## Další kroky

1. **Implementovat** návrat content v `processSystemMessage()`
2. **Rozhodnout** co main agent dělá s notifikací
3. **Testnout** E2E: spawn → completion → main agent vidí zprávu

---

## Soubory k úpravě

- `pkg/agent/loop.go` - `processSystemMessage()` - změnit `return "", nil` na `return content, nil`

---

## Analýza vytvořena

- **Kým:** subagent-12 (picoclaw-self-update)
- **Kdy:** 2026-02-19 06:51
- **Status:** Analýza, čeká na implementaci
