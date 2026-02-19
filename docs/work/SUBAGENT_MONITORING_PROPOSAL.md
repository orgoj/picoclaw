# Subagent Monitoring Proposal for Picoclaw

## Executive Summary

This document analyzes the current async subagent implementation in picoclaw and proposes solutions for monitoring, logging, and managing spawned subagents.

**Key Finding**: Monitoring tools (`subagent_status`, `subagent_history`, `subagent_message`, `subagent_cancel`) already exist in `pkg/tools/subagent_tools.go` but are **NOT registered** in the main agent's tool registry. This is the primary reason subagents appear "invisible."

---

## 1. Current State Analysis

### 1.1 Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        AgentLoop                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    ToolRegistry                          │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────┐  │    │
│  │  │  spawn   │ │subagent  │ │ readFile │ │   ...      │  │    │
│  │  │  (async) │ │ (sync)   │ │          │ │            │  │    │
│  │  └──────────┘ └──────────┘ └──────────┘ └────────────┘  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                 SubagentManager                          │    │
│  │  tasks: map[string]*SubagentTask                         │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │ subagent-1: {running, "task description"...}    │    │    │
│  │  │ subagent-2: {completed, "result"...}            │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│                              ▼ (via MessageBus)                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              processSystemMessage()                      │    │
│  │  Handles subagent completion notifications               │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Code Locations

| Component | File | Purpose |
|-----------|------|---------|
| `SpawnTool` | `pkg/tools/spawn.go` | Async subagent spawning |
| `SubagentTool` | `pkg/tools/subagent.go` | Sync subagent execution |
| `SubagentManager` | `pkg/tools/subagent.go` | Task lifecycle management |
| Monitoring Tools | `pkg/tools/subagent_tools.go` | status/history/message/cancel |
| `AgentLoop` | `pkg/agent/loop.go` | Main agent orchestration |
| Tool Registration | `pkg/agent/loop.go:58-97` | `createToolRegistry()` |

### 1.3 SubagentTask Structure

```go
type SubagentTask struct {
    ID            string
    Task          string
    Label         string
    OriginChannel string
    OriginChatID  string
    Status        string    // "running", "completed", "failed", "cancelled"
    Result        string
    Created       int64     // Unix timestamp in milliseconds
    PendingMsgs   []string  // Queued guidance messages (NOT CONSUMED!)
}
```

### 1.4 Current Problems

#### Problem 1: Monitoring Tools Not Registered ❌
**Location**: `pkg/agent/loop.go:58-97`

The `createToolRegistry()` function does NOT register the monitoring tools:
```go
func createToolRegistry(...) *tools.ToolRegistry {
    registry := tools.NewToolRegistry()
    // ... registers spawn, subagent, files, shell, web, etc.
    // BUT MISSING:
    // - subagent_status
    // - subagent_history  
    // - subagent_message
    // - subagent_cancel
}
```

**Impact**: Main agent cannot query subagent status. LLM doesn't know these tools exist.

#### Problem 2: No Real-Time Logging ❌
**Location**: `pkg/tools/subagent.go:109-174` (`runTask()`)

Subagent runs in isolation with no log streaming:
```go
func (sm *SubagentManager) runTask(ctx context.Context, task *SubagentTask, callback AsyncCallback) {
    // ... runs RunToolLoop but no intermediate updates
    loopResult, err := RunToolLoop(ctx, ...)
    // Only updates status after completion
}
```

**Impact**: No visibility into what subagent is doing until it finishes.

#### Problem 3: Context Cancellation Not Working ❌
**Location**: `pkg/tools/subagent.go:196-207` (`Cancel()`)

```go
func (sm *SubagentManager) Cancel(taskID string) error {
    task, ok := sm.tasks[taskID]
    // ...
    task.Status = "cancelled"  // Only sets flag!
    task.Result = "Cancelled by user"
    return nil
}
```

The running goroutine has no way to detect cancellation. The `runTask()` function checks `ctx.Done()` only **before** starting, not during execution.

#### Problem 4: Pending Messages Not Consumed ❌
**Location**: `pkg/tools/subagent.go:184-194` (`SendMessage()`)

```go
func (sm *SubagentManager) SendMessage(taskID, message string) error {
    task.PendingMsgs = append(task.PendingMsgs, message)
    return nil
}
```

Messages are queued but `runTask()` never reads from `task.PendingMsgs`.

---

## 2. Proposed Solution

### 2.1 Quick Fix: Register Monitoring Tools (Priority: HIGH)

**File**: `pkg/agent/loop.go`

Add monitoring tools to `createToolRegistry()` or register them separately in `NewAgentLoop()`:

```go
func NewAgentLoop(cfg *config.Config, msgBus *bus.MessageBus, provider providers.LLMProvider) *AgentLoop {
    // ... existing code ...
    
    // Register subagent monitoring tools
    registry.Register(tools.NewSubagentStatusTool(subagentManager))
    registry.Register(tools.NewSubagentHistoryTool(subagentManager))
    registry.Register(tools.NewSubagentMessageTool(subagentManager))
    registry.Register(tools.NewSubagentCancelTool(subagentManager))
    
    // ... rest of initialization ...
}
```

**Estimated effort**: 30 minutes

### 2.2 New Tool: `subagent_logs` (Priority: MEDIUM)

Add a new tool to view real-time/queued logs from running subagents.

**File**: `pkg/tools/subagent_tools.go`

```go
// SubagentLogsTool shows the execution log of a running or completed subagent.
type SubagentLogsTool struct {
    manager *SubagentManager
}

func NewSubagentLogsTool(manager *SubagentManager) *SubagentLogsTool {
    return &SubagentLogsTool{manager: manager}
}

func (t *SubagentLogsTool) Name() string {
    return "subagent_logs"
}

func (t *SubagentLogsTool) Description() string {
    return "View the execution log of a subagent task. Shows recent tool calls, LLM responses, and any errors."
}

func (t *SubagentLogsTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "task_id": map[string]interface{}{
                "type":        "string",
                "description": "The ID of the subagent task",
            },
            "tail": map[string]interface{}{
                "type":        "integer",
                "description": "Number of recent log entries to show (default: 20)",
            },
        },
        "required": []string{"task_id"},
    }
}

func (t *SubagentLogsTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    taskID, _ := args["task_id"].(string)
    tail, _ := args["tail"].(int)
    if tail == 0 {
        tail = 20
    }
    
    task, found := t.manager.GetTask(taskID)
    if !found {
        return ErrorResult(fmt.Sprintf("Subagent task '%s' not found", taskID))
    }
    
    // Return task execution info
    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("**Subagent: %s**\n", task.ID))
    sb.WriteString(fmt.Sprintf("Status: %s\n", task.Status))
    sb.WriteString(fmt.Sprintf("Task: %s\n\n", task.Task))
    
    if len(task.Logs) > 0 {
        sb.WriteString("**Recent Activity:**\n")
        start := 0
        if len(task.Logs) > tail {
            start = len(task.Logs) - tail
        }
        for i := start; i < len(task.Logs); i++ {
            sb.WriteString(task.Logs[i] + "\n")
        }
    }
    
    if task.Result != "" {
        sb.WriteString(fmt.Sprintf("\n**Result:**\n%s\n", task.Result))
    }
    
    return UserResult(sb.String())
}
```

### 2.3 Enhanced SubagentTask with Logging (Priority: MEDIUM)

**File**: `pkg/tools/subagent.go`

Add logging support to track execution:

```go
type SubagentTask struct {
    ID            string
    Task          string
    Label         string
    OriginChannel string
    OriginChatID  string
    Status        string
    Result        string
    Created       int64
    PendingMsgs   []string
    Logs          []string    // NEW: Execution log entries
    Started       int64       // NEW: When execution started
    Completed     int64       // NEW: When execution completed
    Iterations    int         // NEW: Tool loop iterations
    mu            sync.Mutex  // NEW: For thread-safe log access
}

// AddLog appends a log entry to the task
func (t *SubagentTask) AddLog(entry string) {
    t.mu.Lock()
    defer t.mu.Unlock()
    timestamp := time.Now().Format("15:04:05")
    t.Logs = append(t.Logs, fmt.Sprintf("[%s] %s", timestamp, entry))
}
```

### 2.4 Enhanced runTask with Logging (Priority: MEDIUM)

Modify `runTask()` to capture execution progress:

```go
func (sm *SubagentManager) runTask(ctx context.Context, task *SubagentTask, callback AsyncCallback) {
    sm.mu.Lock()
    task.Status = "running"
    task.Started = time.Now().UnixMilli()
    task.AddLog("Task started")
    sm.mu.Unlock()

    // ... build system prompt ...

    // Wrap RunToolLoop with logging
    loopResult, err := RunToolLoopWithLogging(ctx, ToolLoopConfig{...}, messages, 
        func(event string, data map[string]any) {
            switch event {
            case "llm_call":
                task.AddLog(fmt.Sprintf("LLM call (iteration %d)", data["iteration"]))
            case "tool_call":
                task.AddLog(fmt.Sprintf("Tool: %s", data["tool"]))
            case "tool_result":
                if data["is_error"].(bool) {
                    task.AddLog(fmt.Sprintf("Tool error: %s", data["error"]))
                } else {
                    task.AddLog(fmt.Sprintf("Tool completed: %s", data["tool"]))
                }
            }
        }, task.OriginChannel, task.OriginChatID)

    // ... handle result ...
    task.Completed = time.Now().UnixMilli()
    task.Iterations = loopResult.Iterations
    task.AddLog(fmt.Sprintf("Task completed (iterations: %d)", loopResult.Iterations))
}
```

### 2.5 Fix Context Cancellation (Priority: HIGH)

**File**: `pkg/tools/subagent.go`

Store cancel functions and implement proper cancellation:

```go
type SubagentManager struct {
    tasks         map[string]*SubagentTask
    cancels       map[string]context.CancelFunc  // NEW: Store cancel functions
    mu            sync.RWMutex
    // ... other fields ...
}

func (sm *SubagentManager) Spawn(ctx context.Context, task, label, originChannel, originChatID string, callback AsyncCallback) (string, error) {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    taskID := fmt.Sprintf("subagent-%d", sm.nextID)
    sm.nextID++

    // Create cancellable context
    taskCtx, cancel := context.WithCancel(ctx)
    
    subagentTask := &SubagentTask{
        ID:            taskID,
        Task:          task,
        // ...
    }
    sm.tasks[taskID] = subagentTask
    sm.cancels[taskID] = cancel  // Store cancel function

    go sm.runTask(taskCtx, subagentTask, callback)
    // ...
}

func (sm *SubagentManager) Cancel(taskID string) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    task, ok := sm.tasks[taskID]
    if !ok {
        return fmt.Errorf("task not found")
    }
    
    if cancel, ok := sm.cancels[taskID]; ok {
        cancel()  // Actually cancel the context!
        delete(sm.cancels, taskID)
    }
    
    task.Status = "cancelled"
    task.Result = "Cancelled by user"
    return nil
}
```

### 2.6 Process Pending Messages (Priority: LOW)

Modify `runTask()` to check for guidance messages:

```go
func (sm *SubagentManager) runTask(ctx context.Context, task *SubagentTask, callback AsyncCallback) {
    // ... in the tool loop ...

    for iteration < config.MaxIterations {
        // Check for pending guidance messages
        task.mu.Lock()
        if len(task.PendingMsgs) > 0 {
            guidance := strings.Join(task.PendingMsgs, "\n")
            task.PendingMsgs = nil
            task.mu.Unlock()
            
            task.AddLog("Received guidance: " + guidance)
            messages = append(messages, providers.Message{
                Role:    "user",
                Content: "Guidance from supervisor: " + guidance,
            })
        } else {
            task.mu.Unlock()
        }
        
        // ... continue with LLM call ...
    }
}
```

---

## 3. API Summary

### Existing Tools (Already Implemented, Need Registration)

| Tool | Description | Status |
|------|-------------|--------|
| `subagent_status` | List all subagent tasks | ⚠️ Not registered |
| `subagent_history` | View task details and result | ⚠️ Not registered |
| `subagent_message` | Send guidance to running subagent | ⚠️ Not registered |
| `subagent_cancel` | Cancel a running task | ⚠️ Not registered |

### New Tools (Proposed)

| Tool | Description | Priority |
|------|-------------|----------|
| `subagent_logs` | View execution logs | Medium |
| `subagent_wait` | Wait for subagent completion with timeout | Low |

---

## 4. Implementation Notes for Go Developers

### 4.1 Quick Fix (Do First)

**File**: `pkg/agent/loop.go`, function `NewAgentLoop()`

Add after line 85 (after registering spawn and subagent tools):

```go
// Register subagent monitoring tools
al.tools.Register(tools.NewSubagentStatusTool(subagentManager))
al.tools.Register(tools.NewSubagentHistoryTool(subagentManager))
al.tools.Register(tools.NewSubagentMessageTool(subagentManager))
al.tools.Register(tools.NewSubagentCancelTool(subagentManager))
```

### 4.2 Thread Safety

All modifications to `SubagentTask` must be protected by mutex when accessed from multiple goroutines:

```go
type SubagentTask struct {
    // ... fields ...
    mu sync.Mutex
}

func (t *SubagentTask) AddLog(entry string) {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.Logs = append(t.Logs, entry)
}

func (t *SubagentTask) GetLogs() []string {
    t.mu.Lock()
    defer t.mu.Unlock()
    result := make([]string, len(t.Logs))
    copy(result, t.Logs)
    return result
}
```

### 4.3 Memory Management

Long-running subagent logs can consume memory. Consider:

1. Limit log entries per task (e.g., max 100 entries)
2. Rotate old entries
3. Clear logs for completed tasks after N minutes

```go
const MaxLogEntries = 100

func (t *SubagentTask) AddLog(entry string) {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    t.Logs = append(t.Logs, entry)
    
    // Trim old entries
    if len(t.Logs) > MaxLogEntries {
        t.Logs = t.Logs[len(t.Logs)-MaxLogEntries:]
    }
}
```

### 4.4 Testing

Add integration tests in `pkg/agent/loop_test.go`:

```go
func TestSubagentMonitoringTools(t *testing.T) {
    // Setup agent loop
    al := createTestAgentLoop()
    
    // Spawn a subagent
    result := al.tools.Execute(context.Background(), "spawn", map[string]interface{}{
        "task": "Test task",
        "label": "test-1",
    })
    
    // List subagents
    statusResult := al.tools.Execute(context.Background(), "subagent_status", map[string]interface{}{})
    if !strings.Contains(statusResult.ForUser, "test-1") {
        t.Error("subagent_status should show spawned task")
    }
    
    // View history
    historyResult := al.tools.Execute(context.Background(), "subagent_history", map[string]interface{}{
        "task_id": "subagent-1",
    })
    if historyResult.IsError {
        t.Error("subagent_history should find the task")
    }
}
```

### 4.5 Graceful Shutdown

On agent shutdown, cancel all running subagents:

```go
func (sm *SubagentManager) Shutdown() {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    for taskID, cancel := range sm.cancels {
        cancel()
        if task, ok := sm.tasks[taskID]; ok {
            task.Status = "cancelled"
            task.Result = "Agent shutdown"
        }
    }
    sm.cancels = make(map[string]context.CancelFunc)
}
```

---

## 5. Migration Path

### Phase 1: Immediate (1 hour)
1. Register existing monitoring tools in `NewAgentLoop()`
2. Test that `subagent_status` and `subagent_history` work

### Phase 2: Short-term (4 hours)
1. Add `Logs` field to `SubagentTask`
2. Implement `subagent_logs` tool
3. Add logging hooks to `runTask()`
4. Fix context cancellation

### Phase 3: Medium-term (8 hours)
1. Implement pending message consumption
2. Add `subagent_wait` tool
3. Add memory limits for logs
4. Add comprehensive tests

---

## 6. Example Usage After Fix

```
User: Spawn a subagent to analyze the log files and find errors

Agent: [uses spawn tool]
Spawned subagent 'log-analyzer' for task: Analyze log files for errors
Task ID: subagent-3

User: What subagents are running?

Agent: [uses subagent_status tool]
Found 1 subagent(s):

- **subagent-3** [running] - log-analyzer (created: 14:32:05)

User: Show me the logs for subagent-3

Agent: [uses subagent_logs tool]
**Subagent: subagent-3**
Status: running
Task: Analyze log files for errors

**Recent Activity:**
[14:32:05] Task started
[14:32:06] LLM call (iteration 1)
[14:32:07] Tool: read_file
[14:32:08] Tool completed: read_file
[14:32:09] Tool: read_file
[14:32:10] Tool completed: read_file
[14:32:11] LLM call (iteration 2)

User: Tell it to focus on error.log only

Agent: [uses subagent_message tool]
Message sent to subagent 'subagent-3'

User: Cancel subagent-3

Agent: [uses subagent_cancel tool]
Subagent 'subagent-3' cancelled
```

---

## 7. Conclusion

The core monitoring infrastructure already exists in picoclaw but is **not wired up**. The immediate fix is simply registering the existing monitoring tools. For production readiness, adding execution logs and fixing context cancellation are essential.

**Estimated total effort**: 12-16 hours for full implementation including tests.
