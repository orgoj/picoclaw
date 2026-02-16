# PicoClaw Subagent Orchestration Enhancement Plan

## Overview

This document proposes enhancements to PicoClaw's subagent system to provide better orchestration capabilities similar to nanobot's subagent tools. The goal is to enable full lifecycle management of background subagents.

## Current State Analysis

### Existing Components

#### `SubagentManager` (`pkg/tools/subagent.go`)
- Manages subagent tasks in a map with mutex protection
- Supports async spawning via `Spawn()` method
- Runs subagent tasks in goroutines via `runTask()`
- Uses `RunToolLoop()` for LLM iteration
- Publishes completion results via message bus

#### `SubagentTask` Structure
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
}
```

#### Current Limitations
1. **No conversation history storage** - subagent messages are not preserved
2. **No mid-execution messaging** - cannot send guidance to running subagents
3. **No cancellation mechanism** - context is passed but not stored for cancellation
4. **Limited status visibility** - basic status string, no progress tracking

---

## Proposed Enhancements

### 1. Enhanced SubagentTask Structure

```go
// SubagentStatus represents the current state of a subagent
type SubagentStatus string

const (
    StatusPending   SubagentStatus = "pending"
    StatusRunning   SubagentStatus = "running"
    StatusCompleted SubagentStatus = "completed"
    StatusFailed    SubagentStatus = "failed"
    StatusCancelled SubagentStatus = "cancelled"
)

// SubagentTaskEnhanced extends SubagentTask with full orchestration support
type SubagentTaskEnhanced struct {
    ID            string            `json:"id"`
    Task          string            `json:"task"`
    Label         string            `json:"label"`
    OriginChannel string            `json:"origin_channel"`
    OriginChatID  string            `json:"origin_chat_id"`
    Status        SubagentStatus    `json:"status"`
    Result        string            `json:"result,omitempty"`
    Created       int64             `json:"created"`
    Started       int64             `json:"started,omitempty"`
    Completed     int64             `json:"completed,omitempty"`
    Iterations    int               `json:"iterations"`
    Progress      string            `json:"progress,omitempty"` // Current activity description
    
    // Conversation history (for subagent_history tool)
    Messages      []providers.Message `json:"messages,omitempty"`
    
    // Cancellation support
    CancelFunc    context.CancelFunc `json:"-"`
    CancelReason  string             `json:"cancel_reason,omitempty"`
    
    // Message queue for mid-execution guidance
    PendingGuidance []GuidanceMessage `json:"pending_guidance,omitempty"`
}

// GuidanceMessage represents a message sent to a running subagent
type GuidanceMessage struct {
    ID        string `json:"id"`
    Content   string `json:"content"`
    Timestamp int64  `json:"timestamp"`
    Read      bool   `json:"read"`
}
```

### 2. subagent_status Tool

**Purpose:** List all background subagents with their current status.

```go
// SubagentStatusTool lists all subagents and their status
type SubagentStatusTool struct {
    manager *SubagentManager
}

func (t *SubagentStatusTool) Name() string {
    return "subagent_status"
}

func (t *SubagentStatusTool) Description() string {
    return "List all background subagents with their current status, progress, and execution time."
}

func (t *SubagentStatusTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "id": map[string]interface{}{
                "type":        "string",
                "description": "Optional specific subagent ID to get detailed status",
            },
            "include_history": map[string]interface{}{
                "type":        "boolean",
                "description": "Include message count from conversation history",
            },
        },
    }
}

func (t *SubagentStatusTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    id, _ := args["id"].(string)
    includeHistory, _ := args["include_history"].(bool)
    
    if id != "" {
        // Return detailed status for specific subagent
        task, ok := t.manager.GetTaskEnhanced(id)
        if !ok {
            return ErrorResult(fmt.Sprintf("Subagent %s not found", id))
        }
        
        result := t.formatDetailedStatus(task, includeHistory)
        return UserResult(result)
    }
    
    // Return summary of all subagents
    tasks := t.manager.ListTasksEnhanced()
    if len(tasks) == 0 {
        return UserResult("No subagents currently running or recently completed.")
    }
    
    var sb strings.Builder
    sb.WriteString("## Subagent Status\n\n")
    
    for _, task := range tasks {
        duration := ""
        if task.Started > 0 {
            if task.Completed > 0 {
                duration = fmt.Sprintf(" (took %s)", formatDuration(task.Completed - task.Started))
            } else {
                duration = fmt.Sprintf(" (running for %s)", formatDuration(time.Now().UnixMilli() - task.Started))
            }
        }
        
        statusIcon := getStatusIcon(task.Status)
        sb.WriteString(fmt.Sprintf("%s **%s** `%s`\n", statusIcon, task.Label, task.ID))
        sb.WriteString(fmt.Sprintf("   Status: %s%s\n", task.Status, duration))
        if task.Progress != "" {
            sb.WriteString(fmt.Sprintf("   Progress: %s\n", task.Progress))
        }
        if includeHistory && len(task.Messages) > 0 {
            sb.WriteString(fmt.Sprintf("   Messages: %d\n", len(task.Messages)))
        }
        sb.WriteString("\n")
    }
    
    return UserResult(sb.String())
}

func getStatusIcon(status SubagentStatus) string {
    switch status {
    case StatusPending: return "⏳"
    case StatusRunning: return "🔄"
    case StatusCompleted: return "✅"
    case StatusFailed: return "❌"
    case StatusCancelled: return "🚫"
    default: return "❓"
    }
}
```

### 3. subagent_history Tool

**Purpose:** View subagent conversation history for debugging and monitoring.

```go
// SubagentHistoryTool retrieves conversation history from a subagent
type SubagentHistoryTool struct {
    manager *SubagentManager
}

func (t *SubagentHistoryTool) Name() string {
    return "subagent_history"
}

func (t *SubagentHistoryTool) Description() string {
    return "View the conversation history of a running or completed subagent. Useful for debugging and monitoring subagent progress."
}

func (t *SubagentHistoryTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "id": map[string]interface{}{
                "type":        "string",
                "description": "Subagent ID to view history for",
            },
            "limit": map[string]interface{}{
                "type":        "integer",
                "description": "Maximum number of messages to return (default: 20)",
            },
            "include_tool_results": map[string]interface{}{
                "type":        "boolean",
                "description": "Include tool call results in history (default: true)",
            },
        },
        "required": []string{"id"},
    }
}

func (t *SubagentHistoryTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    id, ok := args["id"].(string)
    if !ok {
        return ErrorResult("id parameter is required")
    }
    
    limit, _ := args["limit"].(int)
    if limit == 0 {
        limit = 20
    }
    includeToolResults, _ := args["include_tool_results"].(bool)
    if includeToolResults == false {
        // Default to true unless explicitly set to false
        if _, exists := args["include_tool_results"]; !exists {
            includeToolResults = true
        }
    }
    
    task, ok := t.manager.GetTaskEnhanced(id)
    if !ok {
        return ErrorResult(fmt.Sprintf("Subagent %s not found", id))
    }
    
    if len(task.Messages) == 0 {
        return UserResult(fmt.Sprintf("Subagent %s has no conversation history yet.", id))
    }
    
    messages := task.Messages
    if len(messages) > limit {
        messages = messages[len(messages)-limit:]
    }
    
    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("## Conversation History: %s\n\n", task.Label))
    sb.WriteString(fmt.Sprintf("**Task:** %s\n\n", task.Task))
    sb.WriteString("---\n\n")
    
    for _, msg := range messages {
        switch msg.Role {
        case "system":
            sb.WriteString("### 🤖 System\n")
            sb.WriteString(fmt.Sprintf("%s\n\n", truncate(msg.Content, 500)))
        case "user":
            sb.WriteString("### 👤 User\n")
            sb.WriteString(fmt.Sprintf("%s\n\n", msg.Content))
        case "assistant":
            sb.WriteString("### 🤖 Assistant\n")
            if msg.Content != "" {
                sb.WriteString(fmt.Sprintf("%s\n\n", msg.Content))
            }
            if len(msg.ToolCalls) > 0 {
                for _, tc := range msg.ToolCalls {
                    sb.WriteString(fmt.Sprintf("*Tool Call: `%s`*\n\n", tc.Function.Name))
                }
            }
        case "tool":
            if includeToolResults {
                sb.WriteString("### ⚙️ Tool Result\n")
                sb.WriteString(fmt.Sprintf("%s\n\n", truncate(msg.Content, 300)))
            }
        }
    }
    
    return UserResult(sb.String())
}
```

### 4. subagent_message Tool

**Purpose:** Send guidance/corrections to a running subagent.

```go
// SubagentMessageTool sends a message to a running subagent
type SubagentMessageTool struct {
    manager *SubagentManager
}

func (t *SubagentMessageTool) Name() string {
    return "subagent_message"
}

func (t *SubagentMessageTool) Description() string {
    return "Send guidance or corrections to a running subagent. The message will be injected into the subagent's conversation context."
}

func (t *SubagentMessageTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "id": map[string]interface{}{
                "type":        "string",
                "description": "Subagent ID to send message to",
            },
            "message": map[string]interface{}{
                "type":        "string",
                "description": "Guidance message to send to the subagent",
            },
            "priority": map[string]interface{}{
                "type":        "boolean",
                "description": "If true, message is processed immediately (interrupts current iteration)",
            },
        },
        "required": []string{"id", "message"},
    }
}

func (t *SubagentMessageTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    id, ok := args["id"].(string)
    if !ok {
        return ErrorResult("id parameter is required")
    }
    
    message, ok := args["message"].(string)
    if !ok {
        return ErrorResult("message parameter is required")
    }
    
    priority, _ := args["priority"].(bool)
    
    task, ok := t.manager.GetTaskEnhanced(id)
    if !ok {
        return ErrorResult(fmt.Sprintf("Subagent %s not found", id))
    }
    
    if task.Status != StatusRunning {
        return ErrorResult(fmt.Sprintf("Subagent %s is not running (status: %s)", id, task.Status))
    }
    
    // Send guidance to subagent
    err := t.manager.SendGuidance(id, message, priority)
    if err != nil {
        return ErrorResult(fmt.Sprintf("Failed to send message: %v", err))
    }
    
    result := fmt.Sprintf("Guidance sent to subagent %s: %s", id, message)
    return UserResult(result)
}
```

### 5. subagent_cancel Tool

**Purpose:** Cancel a running subagent.

```go
// SubagentCancelTool cancels a running subagent
type SubagentCancelTool struct {
    manager *SubagentManager
}

func (t *SubagentCancelTool) Name() string {
    return "subagent_cancel"
}

func (t *SubagentCancelTool) Description() string {
    return "Cancel a running subagent. The subagent will stop at the next safe point and report partial results."
}

func (t *SubagentCancelTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "id": map[string]interface{}{
                "type":        "string",
                "description": "Subagent ID to cancel",
            },
            "reason": map[string]interface{}{
                "type":        "string",
                "description": "Reason for cancellation (optional, for logging)",
            },
        },
        "required": []string{"id"},
    }
}

func (t *SubagentCancelTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    id, ok := args["id"].(string)
    if !ok {
        return ErrorResult("id parameter is required")
    }
    
    reason, _ := args["reason"].(string)
    if reason == "" {
        reason = "User requested cancellation"
    }
    
    task, ok := t.manager.GetTaskEnhanced(id)
    if !ok {
        return ErrorResult(fmt.Sprintf("Subagent %s not found", id))
    }
    
    if task.Status != StatusRunning && task.Status != StatusPending {
        return ErrorResult(fmt.Sprintf("Subagent %s cannot be cancelled (status: %s)", id, task.Status))
    }
    
    // Cancel the subagent
    err := t.manager.Cancel(id, reason)
    if err != nil {
        return ErrorResult(fmt.Sprintf("Failed to cancel subagent: %v", err))
    }
    
    result := fmt.Sprintf("Subagent %s cancelled. Reason: %s", id, reason)
    return UserResult(result)
}
```

---

## Enhanced SubagentManager

### New Methods

```go
// Enhanced SubagentManager with orchestration support
type SubagentManagerEnhanced struct {
    tasks         map[string]*SubagentTaskEnhanced
    mu            sync.RWMutex
    provider      providers.LLMProvider
    defaultModel  string
    bus           *bus.MessageBus
    workspace     string
    tools         *ToolRegistry
    maxIterations int
    nextID        int
    
    // Callbacks for async notifications
    onStatusChange func(task *SubagentTaskEnhanced)
    onProgress     func(task *SubagentTaskEnhanced, progress string)
}

// GetTaskEnhanced returns a task with full details
func (sm *SubagentManagerEnhanced) GetTaskEnhanced(id string) (*SubagentTaskEnhanced, bool) {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    task, ok := sm.tasks[id]
    return task, ok
}

// ListTasksEnhanced returns all tasks with enhanced details
func (sm *SubagentManagerEnhanced) ListTasksEnhanced() []*SubagentTaskEnhanced {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    
    tasks := make([]*SubagentTaskEnhanced, 0, len(sm.tasks))
    for _, task := range sm.tasks {
        tasks = append(tasks, task)
    }
    
    // Sort by creation time, newest first
    sort.Slice(tasks, func(i, j int) bool {
        return tasks[i].Created > tasks[j].Created
    })
    
    return tasks
}

// Spawn creates and starts a new subagent with cancellation support
func (sm *SubagentManagerEnhanced) Spawn(ctx context.Context, task, label, originChannel, originChatID string, callback AsyncCallback) (string, error) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    taskID := fmt.Sprintf("subagent-%d", sm.nextID)
    sm.nextID++
    
    // Create cancellable context
    cancelCtx, cancelFunc := context.WithCancel(ctx)
    
    subagentTask := &SubagentTaskEnhanced{
        ID:              taskID,
        Task:            task,
        Label:           label,
        OriginChannel:   originChannel,
        OriginChatID:    originChatID,
        Status:          StatusPending,
        Created:         time.Now().UnixMilli(),
        CancelFunc:      cancelFunc,
        Messages:        make([]providers.Message, 0),
        PendingGuidance: make([]GuidanceMessage, 0),
    }
    sm.tasks[taskID] = subagentTask
    
    // Start task in background
    go sm.runTaskEnhanced(cancelCtx, subagentTask, callback)
    
    if label != "" {
        return fmt.Sprintf("Spawned subagent '%s' (ID: %s) for task: %s", label, taskID, task), nil
    }
    return fmt.Sprintf("Spawned subagent (ID: %s) for task: %s", taskID, task), nil
}

// SendGuidance sends a message to a running subagent
func (sm *SubagentManagerEnhanced) SendGuidance(id, message string, priority bool) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    task, ok := sm.tasks[id]
    if !ok {
        return fmt.Errorf("subagent not found")
    }
    
    if task.Status != StatusRunning {
        return fmt.Errorf("subagent is not running")
    }
    
    guidance := GuidanceMessage{
        ID:        fmt.Sprintf("guidance-%d", time.Now().UnixNano()),
        Content:   message,
        Timestamp: time.Now().UnixMilli(),
        Read:      false,
    }
    
    if priority {
        // Prepend to process immediately
        task.PendingGuidance = append([]GuidanceMessage{guidance}, task.PendingGuidance...)
    } else {
        task.PendingGuidance = append(task.PendingGuidance, guidance)
    }
    
    return nil
}

// Cancel cancels a running subagent
func (sm *SubagentManagerEnhanced) Cancel(id, reason string) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    task, ok := sm.tasks[id]
    if !ok {
        return fmt.Errorf("subagent not found")
    }
    
    if task.Status != StatusRunning && task.Status != StatusPending {
        return fmt.Errorf("subagent cannot be cancelled")
    }
    
    task.CancelReason = reason
    task.CancelFunc()
    
    return nil
}

// runTaskEnhanced runs the subagent with enhanced monitoring
func (sm *SubagentManagerEnhanced) runTaskEnhanced(ctx context.Context, task *SubagentTaskEnhanced, callback AsyncCallback) {
    sm.updateStatus(task, StatusRunning)
    task.Started = time.Now().UnixMilli()
    
    // Build system prompt for subagent
    systemPrompt := `You are a subagent. Complete the given task independently and report the result.
You have access to tools - use them as needed to complete your task.
After completing the task, provide a clear summary of what was done.

**IMPORTANT:** You may receive guidance messages from the main agent. 
When you receive guidance in <SupervisorGuidance> tags, adjust your approach accordingly.
Stay focused on completing your assigned task efficiently.`
    
    task.Messages = append(task.Messages, providers.Message{
        Role:    "system",
        Content: systemPrompt,
    })
    
    task.Messages = append(task.Messages, providers.Message{
        Role:    "user",
        Content: task.Task,
    })
    
    // Check if context is already cancelled
    select {
    case <-ctx.Done():
        sm.handleCancellation(task, "Cancelled before execution")
        return
    default:
    }
    
    // Run enhanced tool loop with guidance injection
    sm.mu.RLock()
    tools := sm.tools
    maxIter := sm.maxIterations
    sm.mu.RUnlock()
    
    loopResult, err := sm.runEnhancedToolLoop(ctx, task, tools, maxIter)
    
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    var result *ToolResult
    defer func() {
        // Call callback if provided
        if callback != nil && result != nil {
            callback(context.Background(), result)
        }
    }()
    
    if err != nil {
        if ctx.Err() != nil {
            sm.handleCancellationLocked(task, task.CancelReason)
            result = &ToolResult{
                ForLLM:  fmt.Sprintf("Subagent %s cancelled: %s", task.ID, task.CancelReason),
                Silent:  false,
                IsError: false,
                Async:   false,
            }
        } else {
            task.Status = StatusFailed
            task.Result = fmt.Sprintf("Error: %v", err)
            task.Completed = time.Now().UnixMilli()
            result = &ToolResult{
                ForLLM:  task.Result,
                Silent:  false,
                IsError: true,
                Async:   false,
                Err:     err,
            }
        }
    } else {
        task.Status = StatusCompleted
        task.Result = loopResult.Content
        task.Completed = time.Now().UnixMilli()
        task.Iterations = loopResult.Iterations
        result = &ToolResult{
            ForLLM:  fmt.Sprintf("Subagent '%s' completed (iterations: %d): %s", task.Label, loopResult.Iterations, loopResult.Content),
            ForUser: loopResult.Content,
            Silent:  false,
            IsError: false,
            Async:   false,
        }
    }
    
    // Send completion notification
    sm.notifyCompletion(task)
}

// runEnhancedToolLoop runs the tool loop with guidance injection support
func (sm *SubagentManagerEnhanced) runEnhancedToolLoop(ctx context.Context, task *SubagentTaskEnhanced, tools *ToolRegistry, maxIter int) (*ToolLoopResult, error) {
    iteration := 0
    var finalContent string
    messages := task.Messages
    
    for iteration < maxIter {
        iteration++
        
        // Check for pending guidance and inject into messages
        sm.mu.Lock()
        if len(task.PendingGuidance) > 0 {
            for _, g := range task.PendingGuidance {
                if !g.Read {
                    guidanceMsg := providers.Message{
                        Role:    "user",
                        Content: fmt.Sprintf("<SupervisorGuidance>%s</SupervisorGuidance>", g.Content),
                    }
                    messages = append(messages, guidanceMsg)
                    g.Read = true
                }
            }
            task.PendingGuidance = nil // Clear processed guidance
        }
        sm.mu.Unlock()
        
        // Update progress
        task.Progress = fmt.Sprintf("Iteration %d/%d", iteration, maxIter)
        task.Iterations = iteration
        
        // Check for cancellation
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }
        
        // Run single LLM iteration (same as RunToolLoop but with message tracking)
        // ... (similar to existing RunToolLoop but saves messages to task.Messages)
    }
    
    return &ToolLoopResult{
        Content:    finalContent,
        Iterations: iteration,
    }, nil
}

// updateStatus safely updates task status
func (sm *SubagentManagerEnhanced) updateStatus(task *SubagentTaskEnhanced, status SubagentStatus) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    task.Status = status
    if sm.onStatusChange != nil {
        sm.onStatusChange(task)
    }
}

// handleCancellation handles task cancellation
func (sm *SubagentManagerEnhanced) handleCancellation(task *SubagentTaskEnhanced, reason string) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    sm.handleCancellationLocked(task, reason)
}

func (sm *SubagentManagerEnhanced) handleCancellationLocked(task *SubagentTaskEnhanced, reason string) {
    task.Status = StatusCancelled
    task.CancelReason = reason
    task.Result = fmt.Sprintf("Task cancelled: %s", reason)
    task.Completed = time.Now().UnixMilli()
}

// notifyCompletion sends completion notification via message bus
func (sm *SubagentManagerEnhanced) notifyCompletion(task *SubagentTaskEnhanced) {
    if sm.bus != nil {
        var content string
        if task.Status == StatusCancelled {
            content = fmt.Sprintf("Task '%s' cancelled: %s", task.Label, task.CancelReason)
        } else {
            content = fmt.Sprintf("Task '%s' %s.\n\nResult:\n%s", task.Label, task.Status, task.Result)
        }
        
        sm.bus.PublishInbound(bus.InboundMessage{
            Channel:  "system",
            SenderID: fmt.Sprintf("subagent:%s", task.ID),
            ChatID:   fmt.Sprintf("%s:%s", task.OriginChannel, task.OriginChatID),
            Content:  content,
        })
    }
}
```

---

## Async Callbacks for Results

### Callback Types

```go
// SubagentEventCallback is called when subagent status changes
type SubagentEventCallback func(event SubagentEvent)

// SubagentEvent represents a subagent lifecycle event
type SubagentEvent struct {
    Type        string                 `json:"type"` // "started", "progress", "completed", "failed", "cancelled"
    TaskID      string                 `json:"task_id"`
    Label       string                 `json:"label"`
    Timestamp   int64                  `json:"timestamp"`
    Data        map[string]interface{} `json:"data,omitempty"`
}

// SetEventCallback registers a callback for subagent events
func (sm *SubagentManagerEnhanced) SetEventCallback(cb SubagentEventCallback) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    sm.onStatusChange = func(task *SubagentTaskEnhanced) {
        cb(SubagentEvent{
            Type:      string(task.Status),
            TaskID:    task.ID,
            Label:     task.Label,
            Timestamp: time.Now().UnixMilli(),
        })
    }
}

// SetProgressCallback registers a callback for progress updates
func (sm *SubagentManagerEnhanced) SetProgressCallback(cb func(taskID, progress string)) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    sm.onProgress = cb
}
```

### Integration with AgentLoop

```go
// In AgentLoop initialization:
func (al *AgentLoop) setupSubagentCallbacks() {
    al.subagentManager.SetEventCallback(func(event SubagentEvent) {
        logger.InfoCF("subagent", "Subagent event",
            map[string]interface{}{
                "type":   event.Type,
                "task_id": event.TaskID,
                "label":  event.Label,
            })
        
        // Optionally notify user of important events
        if event.Type == "completed" || event.Type == "failed" {
            // Send notification via bus
            // (handled by existing completion notification)
        }
    })
}
```

---

## Implementation Roadmap

### Phase 1: Core Enhancements (Priority)
1. ✅ Extend `SubagentTask` with conversation history
2. ✅ Add cancellation support via context
3. ✅ Implement `subagent_status` tool
4. ✅ Implement `subagent_cancel` tool

### Phase 2: Monitoring & Debugging
1. ✅ Implement `subagent_history` tool
2. ✅ Add progress tracking
3. ✅ Add event callbacks for status changes

### Phase 3: Advanced Orchestration
1. ✅ Implement `subagent_message` tool
2. ✅ Add guidance message injection
3. ✅ Support priority guidance

### Phase 4: Polish
1. Add task cleanup for old completed tasks
2. Add task persistence for crash recovery
3. Add metrics collection (execution time, token usage)

---

## File Changes Summary

| File | Changes |
|------|---------|
| `pkg/tools/subagent.go` | Add `SubagentTaskEnhanced`, `SubagentManagerEnhanced`, new methods |
| `pkg/tools/subagent_status.go` | **NEW** - `SubagentStatusTool` implementation |
| `pkg/tools/subagent_history.go` | **NEW** - `SubagentHistoryTool` implementation |
| `pkg/tools/subagent_message.go` | **NEW** - `SubagentMessageTool` implementation |
| `pkg/tools/subagent_cancel.go` | **NEW** - `SubagentCancelTool` implementation |
| `pkg/agent/loop.go` | Register new tools, setup callbacks |

---

## Usage Examples

### Check subagent status
```
User: What subagents are running?
Agent: [uses subagent_status]
## Subagent Status

🔄 **file-organizer** `subagent-5`
   Status: running (running for 2m30s)
   Progress: Iteration 3/10

✅ **web-research** `subagent-4`
   Status: completed (took 1m15s)
```

### View subagent history
```
User: Show me what the file-organizer subagent is doing
Agent: [uses subagent_history id="subagent-5"]

## Conversation History: file-organizer

**Task:** Organize files in the downloads folder by type

---
### 🤖 System
You are a subagent...

### 👤 User
Organize files in the downloads folder by type

### 🤖 Assistant
I'll start by listing the files in the downloads folder.

*Tool Call: `list_dir`*

### ⚙️ Tool Result
Found 47 files...
```

### Send guidance to subagent
```
User: Tell the file-organizer to skip any .tmp files
Agent: [uses subagent_message id="subagent-5" message="Skip any .tmp files, they are temporary and don't need organizing"]
Guidance sent to subagent subagent-5: Skip any .tmp files
```

### Cancel subagent
```
User: Cancel the file-organizer task
Agent: [uses subagent_cancel id="subagent-5" reason="User requested - no longer needed"]
Subagent subagent-5 cancelled. Reason: User requested - no longer needed
```

---

## Conclusion

This enhancement plan provides PicoClaw with comprehensive subagent orchestration capabilities:

1. **Visibility**: Full insight into running subagents via `subagent_status`
2. **Debuggability**: Conversation history access via `subagent_history`
3. **Control**: Mid-execution guidance via `subagent_message`
4. **Safety**: Graceful cancellation via `subagent_cancel`
5. **Extensibility**: Event callbacks for custom integrations

The implementation follows PicoClaw's existing patterns and integrates seamlessly with the current architecture.
