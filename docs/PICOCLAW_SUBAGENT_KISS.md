# PicoClaw Subagent Tools - KISS Plan

## Overview

Add 4 tools for subagent management. Use existing `SubagentManager` methods (`GetTask`, `ListTasks`, `SendMessage`, `Cancel`). No changes to `SubagentTask` struct.

## Files

```
pkg/tools/subagent_tools.go  # ~150 lines, 4 tools
```

## Tools

### 1. subagent_status
List all subagent tasks, optionally filter by status.

```go
type SubagentStatusTool struct {
    manager *SubagentManager
}

func (t *SubagentStatusTool) Name() string {
    return "subagent_status"
}

func (t *SubagentStatusTool) Description() string {
    return "List all background subagent tasks. Optionally filter by status (running, completed, failed, cancelled)."
}

func (t *SubagentStatusTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "status": map[string]interface{}{
                "type":        "string",
                "description": "Filter by status: running, completed, failed, cancelled",
            },
        },
    }
}

func (t *SubagentStatusTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    status, ok := args["status"].(string)
    if !ok || status == "" {
        // List all tasks
        tasks := t.manager.ListTasks()
        var lines []string
        lines = append(lines, fmt.Sprintf("Total tasks: %d", len(tasks)))
        for _, task := range tasks {
            lines = append(lines, fmt.Sprintf("- %s [%s]: %s", task.ID, task.Status, task.Label))
        }
        return UserResult(strings.Join(lines, "\n"))
    }

    // Filter by status
    tasks := t.manager.ListTasks()
    var filtered []*SubagentTask
    for _, task := range tasks {
        if task.Status == status {
            filtered = append(filtered, task)
        }
    }

    var lines []string
    lines = append(lines, fmt.Sprintf("Tasks with status '%s': %d", status, len(filtered)))
    for _, task := range filtered {
        lines = append(lines, fmt.Sprintf("- %s: %s", task.ID, task.Label))
    }
    return UserResult(strings.Join(lines, "\n"))
}
```

### 2. subagent_history
View details and result of a specific subagent task.

```go
type SubagentHistoryTool struct {
    manager *SubagentManager
}

func (t *SubagentHistoryTool) Name() string {
    return "subagent_history"
}

func (t *SubagentHistoryTool) Description() string {
    return "View details and result of a specific subagent task by ID."
}

func (t *SubagentHistoryTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "required": []string{"id"},
        "properties": map[string]interface{}{
            "id": map[string]interface{}{
                "type":        "string",
                "description": "Subagent task ID",
            },
        },
    }
}

func (t *SubagentHistoryTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    taskID, ok := args["id"].(string)
    if !ok || strings.TrimSpace(taskID) == "" {
        return ErrorResult("task_id is required and cannot be empty")
    }

    task, ok := t.manager.GetTask(taskID)
    if !ok {
        return ErrorResult(fmt.Sprintf("Task not found: %s", taskID))
    }

    var lines []string
    lines = append(lines, fmt.Sprintf("ID: %s", task.ID))
    lines = append(lines, fmt.Sprintf("Label: %s", task.Label))
    lines = append(lines, fmt.Sprintf("Status: %s", task.Status))
    lines = append(lines, fmt.Sprintf("Created: %s", time.UnixMilli(task.Created).Format(time.RFC3339)))
    lines = append(lines, fmt.Sprintf("Result: %s", truncate(task.Result, 500)))

    return UserResult(strings.Join(lines, "\n"))
}
```

### 3. subagent_message
Send guidance message to a running subagent task.

```go
type SubagentMessageTool struct {
    manager *SubagentManager
}

func (t *SubagentMessageTool) Name() string {
    return "subagent_message"
}

func (t *SubagentMessageTool) Description() string {
    return "Send guidance message to a running subagent task."
}

func (t *SubagentMessageTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "required": []string{"id", "message"},
        "properties": map[string]interface{}{
            "id": map[string]interface{}{
                "type":        "string",
                "description": "Subagent task ID",
            },
            "message": map[string]interface{}{
                "type":        "string",
                "description": "Guidance message to send",
            },
        },
    }
}

func (t *SubagentMessageTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    taskID, ok := args["id"].(string)
    if !ok || strings.TrimSpace(taskID) == "" {
        return ErrorResult("task_id is required and cannot be empty")
    }

    message, ok := args["message"].(string)
    if !ok || strings.TrimSpace(message) == "" {
        return ErrorResult("message is required and cannot be empty")
    }

    err := t.manager.SendMessage(taskID, message)
    if err != nil {
        return ErrorResult(fmt.Sprintf("Failed to send message: %v", err))
    }

    return UserResult(fmt.Sprintf("Sent guidance to task %s", taskID))
}
```

### 4. subagent_cancel
Cancel a running subagent task.

```go
type SubagentCancelTool struct {
    manager *SubagentManager
}

func (t *SubagentCancelTool) Name() string {
    return "subagent_cancel"
}

func (t *SubagentCancelTool) Description() string {
    return "Cancel a running subagent task."
}

func (t *SubagentCancelTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "required": []string{"id"},
        "properties": map[string]interface{}{
            "id": map[string]interface{}{
                "type":        "string",
                "description": "Subagent task ID",
            },
        },
    }
}

func (t *SubagentCancelTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    taskID, ok := args["id"].(string)
    if !ok || strings.TrimSpace(taskID) == "" {
        return ErrorResult("task_id is required and cannot be empty")
    }

    err := t.manager.Cancel(taskID)
    if err != nil {
        return ErrorResult(fmt.Sprintf("Failed to cancel task: %v", err))
    }

    return UserResult(fmt.Sprintf("Cancelled task %s", taskID))
}
```

## Registration

```go
func RegisterSubagentTools(registry *ToolRegistry, manager *SubagentManager) {
    registry.Register(&SubagentStatusTool{manager})
    registry.Register(&SubagentHistoryTool{manager})
    registry.Register(&SubagentMessageTool{manager})
    registry.Register(&SubagentCancelTool{manager})
}
```

## Integration

Update `pkg/agent/loop.go` to register tools:

```go
// In createToolRegistry():
import "github.com/sipeed/picoclaw/pkg/tools"

// After registering other tools:
RegisterSubagentTools(toolsRegistry, subagentManager)
```

## Notes

- Uses existing `SubagentManager` methods - no changes to manager needed
- No changes to `SubagentTask` struct
- Uses ok pattern for type assertions (safe)
- No race conditions - manager already uses mutex

## Done
