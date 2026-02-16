# PicoClaw Subagent Orchestration - Code Review

## Summary

This review analyzes the proposed subagent orchestration enhancement plan in `PICOCLAW_SUBAGENT_PLAN.md`. The plan is well-structured and comprehensive, but several issues and edge cases need addressing before implementation.

---

## Critical Issues

### 1. Race Condition in Guidance Message Processing

**Location:** `runEnhancedToolLoop()` - Guidance injection

**Problem:**
```go
sm.mu.Lock()
if len(task.PendingGuidance) > 0 {
    for _, g := range task.PendingGuidance {
        if !g.Read {
            // ...
            g.Read = true  // This modifies a COPY, not the original!
        }
    }
    task.PendingGuidance = nil
}
sm.mu.Unlock()
```

The loop variable `g` is a copy of the struct, so `g.Read = true` doesn't update the original slice element.

**Fix:**
```go
for i := range task.PendingGuidance {
    if !task.PendingGuidance[i].Read {
        // Process guidance...
        task.PendingGuidance[i].Read = true
    }
}
```

---

### 2. JSON Tag Omission for CancelFunc

**Location:** `SubagentTaskEnhanced` struct

**Problem:**
```go
CancelFunc context.CancelFunc `json:"-"`
```

While this correctly prevents serialization, attempting to serialize this struct (e.g., for debugging or API responses) will silently omit the field. This is correct behavior but should be documented.

**Recommendation:** Add a comment explaining why `CancelFunc` is excluded from JSON.

---

### 3. Missing Mutex Protection in Tool Loop

**Location:** `runEnhancedToolLoop()`

**Problem:**
```go
task.Progress = fmt.Sprintf("Iteration %d/%d", iteration, maxIter)
task.Iterations = iteration
```

These fields are written without holding the lock, but may be read concurrently by `subagent_status` tool.

**Fix:**
```go
sm.mu.Lock()
task.Progress = fmt.Sprintf("Iteration %d/%d", iteration, maxIter)
task.Iterations = iteration
sm.mu.Unlock()
```

---

### 4. Memory Leak: Unbounded Task Storage

**Location:** `SubagentManagerEnhanced.tasks` map

**Problem:** Tasks are never removed from the map. Long-running systems will accumulate completed tasks indefinitely.

**Fix:** Implement cleanup mechanism:
```go
func (sm *SubagentManagerEnhanced) CleanupOldTasks(maxAge time.Duration) int {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    cutoff := time.Now().UnixMilli() - maxAge.Milliseconds()
    removed := 0
    
    for id, task := range sm.tasks {
        if task.Completed > 0 && task.Completed < cutoff {
            delete(sm.tasks, id)
            removed++
        }
    }
    return removed
}
```

Also consider:
- Max task limit (e.g., keep last 100 tasks)
- Configurable retention policy
- Persistence option before cleanup

---

## Edge Cases

### 5. Double Cancellation

**Scenario:** User calls `subagent_cancel` twice on the same task.

**Current Behavior:** Second call returns error "subagent cannot be cancelled" because status already changed.

**Recommendation:** Return idempotent success:
```go
if task.Status == StatusCancelled {
    return nil // Already cancelled, success
}
```

---

### 6. Message to Completed Subagent

**Scenario:** User tries to send guidance to a completed subagent.

**Current Behavior:** Returns error "subagent is not running".

**Issue:** User might not realize task finished and lose the guidance.

**Recommendation:** Include task status in error:
```go
return fmt.Errorf("subagent %s is not running (status: %s, completed at %s)", 
    id, task.Status, time.UnixMilli(task.Completed).Format(time.RFC3339))
```

---

### 7. Empty Message Handling

**Location:** `SubagentMessageTool.Execute()`

**Problem:** Empty or whitespace-only messages are accepted.

**Fix:**
```go
message := strings.TrimSpace(args["message"].(string))
if message == "" {
    return ErrorResult("message cannot be empty")
}
```

---

### 8. Iteration Limit Reached Without Completion

**Scenario:** Subagent hits max iterations without completing.

**Current Behavior:** Loop exits with whatever `finalContent` is.

**Problem:** `finalContent` may be empty if the last response was a tool call.

**Fix:**
```go
if iteration >= maxIter {
    return nil, fmt.Errorf("max iterations (%d) reached without completion", maxIter)
}
```

---

### 9. Context Cancellation During LLM Call

**Scenario:** Context cancelled while waiting for LLM response.

**Current Behavior:** Not explicitly handled.

**Recommendation:** Ensure LLM provider respects context cancellation and has timeout:
```go
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
```

---

### 10. Guidance Accumulation Overflow

**Scenario:** Main agent sends many guidance messages faster than subagent processes them.

**Problem:** `PendingGuidance` slice grows unbounded.

**Fix:** Implement guidance queue limit:
```go
const maxPendingGuidance = 10

func (sm *SubagentManagerEnhanced) SendGuidance(id, message string, priority bool) error {
    // ...
    if len(task.PendingGuidance) >= maxPendingGuidance {
        // Drop oldest non-priority guidance
        task.PendingGuidance = task.PendingGuidance[1:]
    }
    // ...
}
```

---

## API Design Issues

### 11. Inconsistent Boolean Parameter Handling

**Location:** `SubagentHistoryTool.Execute()`

```go
includeToolResults, _ := args["include_tool_results"].(bool)
if includeToolResults == false {
    if _, exists := args["include_tool_results"]; !exists {
        includeToolResults = true
    }
}
```

**Problem:** This logic is convoluted. JSON unmarshaling `false` vs. missing key is indistinguishable.

**Fix:** Use pointer or explicit defaults:
```go
includeToolResults := true // default
if v, ok := args["include_tool_results"]; ok {
    includeToolResults = v.(bool)
}
```

---

### 12. Missing Input Validation

**Location:** All tool Execute methods

**Problem:** ID parameters not validated for format/length.

**Risk:** Malicious IDs could cause issues in logs or storage.

**Fix:**
```go
func validateTaskID(id string) error {
    if len(id) > 64 {
        return fmt.Errorf("task ID too long")
    }
    if !taskIDRegex.MatchString(id) {
        return fmt.Errorf("invalid task ID format")
    }
    return nil
}
```

---

## Missing Features

### 13. No Task Priority Support

**Recommendation:** Add priority levels for task scheduling:
```go
type TaskPriority int
const (
    PriorityLow TaskPriority = iota
    PriorityNormal
    PriorityHigh
)
```

---

### 14. No Task Result Truncation

**Problem:** Completed task results could be very large.

**Recommendation:** Truncate stored results:
```go
const maxResultLength = 10000

func truncateResult(result string) string {
    if len(result) > maxResultLength {
        return result[:maxResultLength] + "\n...[truncated]"
    }
    return result
}
```

---

### 15. No Subagent-to-Subagent Communication Prevention

**Scenario:** A subagent could call `subagent_spawn` creating uncontrolled nesting.

**Recommendation:** Add context flag to prevent nested subagents:
```go
type subagentCtxKey struct{}
func ContextWithSubagent(ctx context.Context) context.Context {
    return context.WithValue(ctx, subagentCtxKey{}, true)
}
func IsSubagentContext(ctx context.Context) bool {
    _, ok := ctx.Value(subagentCtxKey{}).(bool)
    return ok
}
```

---

## Documentation Issues

### 16. Callback Threading Not Documented

**Problem:** `onStatusChange` and `onProgress` callbacks are called with locks held.

**Risk:** Deadlock if callback tries to access manager.

**Fix:** Call callbacks without lock:
```go
func (sm *SubagentManagerEnhanced) updateStatus(task *SubagentTaskEnhanced, status SubagentStatus) {
    var callback func(*SubagentTaskEnhanced)
    sm.mu.Lock()
    task.Status = status
    callback = sm.onStatusChange
    sm.mu.Unlock()
    
    if callback != nil {
        callback(task)
    }
}
```

---

### 17. Error Handling Not Specified

**Problem:** Tool result `ForLLM` vs `ForUser` fields not consistently used.

**Recommendation:** Document when to use each field:
- `ForLLM`: Internal representation, includes IDs and technical details
- `ForUser`: User-facing summary, omits technical details

---

## Security Considerations

### 18. Guidance Message Injection

**Risk:** Malicious guidance could trick subagent into unwanted actions.

**Recommendation:** Mark guidance clearly in system prompt and consider:
```go
Content: fmt.Sprintf("<SupervisorGuidance priority=\"%t\">%s</SupervisorGuidance>", priority, message),
```

---

### 19. Resource Exhaustion

**Risk:** Creating many subagents could exhaust resources.

**Fix:** Implement concurrent task limit:
```go
const maxConcurrentTasks = 10

func (sm *SubagentManagerEnhanced) Spawn(...) (string, error) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    running := 0
    for _, t := range sm.tasks {
        if t.Status == StatusRunning || t.Status == StatusPending {
            running++
        }
    }
    if running >= maxConcurrentTasks {
        return "", fmt.Errorf("maximum concurrent subagents (%d) reached", maxConcurrentTasks)
    }
    // ...
}
```

---

## Minor Issues

### 20. Timestamp Precision Inconsistency

**Problem:** Some timestamps use `UnixMilli()`, others might use `Unix()`.

**Fix:** Standardize on `UnixMilli()` throughout and document.

---

### 21. Missing Unit Test Recommendations

The plan should include test coverage for:
- Concurrent access patterns
- Cancellation scenarios
- Guidance injection
- Edge cases listed above

---

## Recommendations Summary

### Must Fix Before Implementation
1. Race condition in guidance processing (#1)
2. Missing mutex for Progress/Iterations (#3)
3. Memory leak from unbounded task storage (#4)
4. Callback deadlock risk (#16)

### Should Fix
5. Double cancellation handling (#5)
6. Empty message validation (#7)
7. Iteration limit handling (#8)
8. Guidance overflow protection (#10)
9. Concurrent task limit (#19)

### Nice to Have
10. Input validation (#12)
11. Task priority (#13)
12. Result truncation (#14)
13. Nested subagent prevention (#15)

---

## Conclusion

The proposed enhancement plan is well-architected and addresses real needs for subagent orchestration. However, the identified race conditions, memory management issues, and edge cases must be addressed before production use. The recommendations above will improve robustness, security, and maintainability.

**Estimated Implementation Impact:** Adding these fixes will increase development time by approximately 20-30% but will prevent significant debugging and production issues.
