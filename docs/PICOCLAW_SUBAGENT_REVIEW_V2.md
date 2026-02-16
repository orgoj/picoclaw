# PicoClaw Subagent Enhancement Review (KISS)

**Review Date:** 2026-02-16  
**Document Reviewed:** PICOCLAW_SUBAGENT_PLAN.md  
**Scope:** 4 new tools: `subagent_status`, `subagent_history`, `subagent_message`, `subagent_cancel`

---

## Summary

The plan is over-engineered. The core idea (4 orchestration tools) is solid, but the implementation adds unnecessary complexity. This review focuses on **what's wrong with the changes themselves**, not circumstances.

---

## Critical Issues

### 1. Unnecessary `Enhanced` Suffix Everywhere

**Problem:** `SubagentTaskEnhanced`, `SubagentManagerEnhanced`, `GetTaskEnhanced()`, `ListTasksEnhanced()`

**Why it's wrong:**
- Creates parallel structures (`SubagentTask` vs `SubagentTaskEnhanced`)
- Forces callers to choose between old/new APIs
- Will cause confusion and tech debt

**Fix:** Just extend `SubagentTask` and `SubagentManager` directly. Migration is inevitable - do it once, cleanly.

---

### 2. `Messages []providers.Message` Stored in Task struct

**Problem:** Storing full conversation history in the task struct.

**Why it's wrong:**
- Memory grows unbounded during long-running tasks
- Large messages are copied on every status check
- JSON serialization of `Messages` in `SubagentTaskEnhanced` will be slow
- The struct has `json:"messages,omitempty"` but these can be huge

**Fix:** Store messages separately (by reference or in a dedicated slice), not in the main task struct. Or store only message count for status, fetch full history on demand.

---

### 3. `CancelFunc` in JSON-serialized struct

```go
type SubagentTaskEnhanced struct {
    ...
    CancelFunc    context.CancelFunc `json:"-"`  // ← This
```

**Why it's wrong:**
- It works, but it's a code smell
- Mixing serializable data with runtime state
- The `json:"-"` is a band-aid

**Fix:** Separate concerns:
```go
type SubagentTask struct { /* serializable fields */ }
type subagentRuntime struct {
    task       *SubagentTask
    cancelFunc context.CancelFunc
    messages   []providers.Message
}
```

---

### 4. Guidance Injection is Over-Complicated

**Problem:** The `PendingGuidance []GuidanceMessage` with `Read` flag, priority queue logic, and injection in tool loop.

**Why it's wrong:**
- `Read` flag is never used meaningfully
- Priority flag creates two insertion paths for no clear benefit
- The injection loop in `runEnhancedToolLoop` has a race condition (locks/unlocks mid-iteration)

```go
sm.mu.Lock()
if len(task.PendingGuidance) > 0 {
    for _, g := range task.PendingGuidance {
        if !g.Read {  // ← Always true, we just added it
            ...
            g.Read = true  // ← Doesn't modify the original slice element!
        }
    }
    task.PendingGuidance = nil
}
sm.mu.Unlock()
```

**Fix:** Simple channel-based approach:
```go
type SubagentManager struct {
    ...
    guidance chan guidanceMsg  // buffered
}

func (sm *SubagentManager) SendGuidance(id, msg string) error {
    // Just send to channel, let the tool loop drain it
}
```

---

### 5. `include_tool_results` Logic Bug

```go
includeToolResults, _ := args["include_tool_results"].(bool)
if includeToolResults == false {
    if _, exists := args["include_tool_results"]; !exists {
        includeToolResults = true
    }
}
```

**Why it's wrong:**
- Over-complicated for "default true"
- `args["include_tool_results"]` from JSON will be `bool` or `nil`, never "explicit false" distinguishable from "missing"

**Fix:**
```go
includeToolResults := true
if v, ok := args["include_tool_results"]; ok {
    includeToolResults = v.(bool)
}
```

---

### 6. Tool Results Return Inconsistent Types

**Problem:** Tools return `UserResult()` or `ErrorResult()` which are presumably helper functions.

**Why it's wrong:**
- Not shown in the plan what these do
- Inconsistent with standard Go error handling
- Makes testing harder

**Fix:** Just return `*ToolResult` directly:
```go
return &ToolResult{
    ForLLM:  "...",
    IsError: true,
    Err:     fmt.Errorf("subagent not found"),
}
```

---

### 7. Missing: Task Cleanup

**Problem:** Plan mentions cleanup in "Phase 4" but no mechanism is designed.

**Why it's wrong:**
- Memory leak: completed tasks are never removed
- `ListTasksEnhanced()` will grow unbounded

**Fix:** Add cleanup to the core design:
```go
type SubagentManager struct {
    maxHistoryAge  time.Duration  // e.g., 1 hour
}

// Called periodically or on each status check
func (sm *SubagentManager) cleanup() {
    // Remove completed/failed/cancelled tasks older than maxHistoryAge
}
```

---

### 8. Callback System is Over-Engineered

```go
onStatusChange func(task *SubagentTaskEnhanced)
onProgress     func(task *SubagentTaskEnhanced, progress string)

func (sm *SubagentManagerEnhanced) SetEventCallback(cb SubagentEventCallback) { ... }
func (sm *SubagentManagerEnhanced) SetProgressCallback(cb func(taskID, progress string)) { ... }
```

**Why it's wrong:**
- Two different callback mechanisms (`onStatusChange` vs `SetEventCallback`)
- `SubagentEvent` type adds complexity for simple notifications
- Callbacks are stored on manager but called from goroutines - potential races

**Fix:** One simple event channel:
```go
type SubagentEvent struct {
    Type   string  // "status", "progress"
    TaskID string
    Status string  // for status events
    Msg    string  // for progress events
}

func (sm *SubagentManager) Events() <-chan SubagentEvent {
    return sm.events
}
```

---

### 9. `runEnhancedToolLoop` is Incomplete

The plan shows:
```go
// ... (similar to existing RunToolLoop but saves messages to task.Messages)
```

**Why it's wrong:**
- The complex part is hand-waved away
- Message tracking, guidance injection, and cancellation all happen here
- Without this implementation, the plan is incomplete

**Fix:** Show the actual implementation or reference the existing code clearly.

---

### 10. File Structure: 4 New Files for 4 Simple Tools

```
pkg/tools/subagent_status.go    # NEW
pkg/tools/subagent_history.go   # NEW
pkg/tools/subagent_message.go   # NEW
pkg/tools/subagent_cancel.go    # NEW
```

**Why it's wrong:**
- Each tool is ~50-80 lines
- They all depend on `SubagentManager`
- Creates maintenance overhead for simple CRUD operations

**Fix:** One file `pkg/tools/subagent_orchestration.go` with all 4 tools. They're tightly coupled anyway.

---

## Minor Issues

### 11. Status Icons in Tool Output

```go
func getStatusIcon(status SubagentStatus) string {
    switch status {
    case StatusPending: return "⏳"
    ...
}
```

**Why it's wrong:** Icons may not render correctly in all terminals/channels. Use text.

### 12. `truncate()` Function Not Defined

Used in `subagent_history` but not shown.

### 13. `formatDuration()` Function Not Defined

Used in `subagent_status` but not shown.

---

## Recommended Minimal Implementation

```go
// pkg/tools/subagent.go - extend existing

type SubagentTask struct {
    ID            string
    Task          string
    Label         string
    Status        string  // "running", "completed", "failed", "cancelled"
    Result        string
    Created       int64
    Started       int64
    Completed     int64
    Iterations    int
    CancelReason  string
    
    // Runtime state (not serialized)
    cancelFunc    context.CancelFunc
    messages      []providers.Message
    guidanceQueue []string
}

type SubagentManager struct {
    tasks    map[string]*SubagentTask
    mu       sync.RWMutex
    events   chan SubagentEvent  // optional, for listeners
    ...
}

// 4 tools - simple implementations:

func (sm *SubagentManager) Status(id string) (*SubagentTask, error) { ... }
func (sm *SubagentManager) ListAll() []*SubagentTask { ... }
func (sm *SubagentManager) History(id string, limit int) ([]providers.Message, error) { ... }
func (sm *SubagentManager) SendGuidance(id, msg string) error { ... }
func (sm *SubagentManager) Cancel(id, reason string) error { ... }
```

One file for tools: `pkg/tools/subagent_tools.go` (all 4 tools, ~150 lines total).

---

## Verdict

| Aspect | Rating |
|--------|--------|
| Core concept | ✅ Good |
| Data structures | ⚠️ Over-complicated |
| API design | ⚠️ Inconsistent |
| Error handling | ⚠️ Incomplete |
| Implementation detail | ❌ Missing key parts |

**Recommendation:** Simplify before implementing. The plan has too many abstractions for what is essentially:
1. List/map of tasks
2. Get status
3. Get history
4. Send message
5. Cancel

These are CRUD operations. Don't over-engineer them.
