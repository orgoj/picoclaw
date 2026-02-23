package tools

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SubagentStatusTool lists all subagent tasks and their status.
type SubagentStatusTool struct {
	manager *SubagentManager
}

func NewSubagentStatusTool(manager *SubagentManager) *SubagentStatusTool {
	return &SubagentStatusTool{
		manager: manager,
	}
}

func (t *SubagentStatusTool) Name() string {
	return "subagent_status"
}

func (t *SubagentStatusTool) Description() string {
	return "List all subagent tasks and their current status. Shows task ID, label, status, and creation time."
}

func (t *SubagentStatusTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"status": map[string]interface{}{
				"type":        "string",
				"description": "Filter by status (running, completed, failed, cancelled). Optional - if empty, shows all.",
			},
		},
		"required": []string{},
	}
}

func (t *SubagentStatusTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	if t.manager == nil {
		return ErrorResult("Subagent manager not configured")
	}

	statusFilter, _ := args["status"].(string)
	statusFilter = strings.ToLower(strings.TrimSpace(statusFilter))

	tasks := t.manager.ListTasks()

	var filtered []*SubagentTask
	for _, task := range tasks {
		if statusFilter == "" || strings.ToLower(task.Status) == statusFilter {
			filtered = append(filtered, task)
		}
	}

	if len(filtered) == 0 {
		if statusFilter != "" {
			return UserResult(fmt.Sprintf("No subagents with status '%s'", statusFilter))
		}
		return UserResult("No subagents found")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d subagent(s):\n\n", len(filtered)))
	for _, task := range filtered {
		created := time.UnixMilli(task.Created).Format("15:04:05")
		label := task.Label
		if label == "" {
			label = "no-label"
		}
		agentInfo := ""
		if task.Name != "" {
			agentInfo = fmt.Sprintf(" [agent: %s]", task.Name)
		}
		dirInfo := ""
		if task.Directory != "" {
			dirInfo = fmt.Sprintf(" @ %s", task.Directory)
		}
		sb.WriteString(fmt.Sprintf("- **%s** [%s] - %s%s%s (created: %s)\n", task.ID, task.Status, label, agentInfo, dirInfo, created))
	}

	return UserResult(sb.String())
}

// SubagentHistoryTool shows the history of a specific subagent task.
type SubagentHistoryTool struct {
	manager *SubagentManager
}

func NewSubagentHistoryTool(manager *SubagentManager) *SubagentHistoryTool {
	return &SubagentHistoryTool{
		manager: manager,
	}
}

func (t *SubagentHistoryTool) Name() string {
	return "subagent_history"
}

func (t *SubagentHistoryTool) Description() string {
	return "View details and result of a specific subagent task."
}

func (t *SubagentHistoryTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task_id": map[string]interface{}{
				"type":        "string",
				"description": "The ID of the subagent task to view",
			},
		},
		"required": []string{"task_id"},
	}
}

func (t *SubagentHistoryTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	taskID, ok := args["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return ErrorResult("task_id is required and cannot be empty")
	}

	if t.manager == nil {
		return ErrorResult("Subagent manager not configured")
	}

	task, found := t.manager.GetTask(taskID)
	if !found {
		return ErrorResult(fmt.Sprintf("Subagent task '%s' not found", taskID))
	}

	label := task.Label
	if label == "" {
		label = "no-label"
	}

	created := time.UnixMilli(task.Created).Format("2006-01-02 15:04:05")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Subagent: %s**\n", task.ID))
	sb.WriteString(fmt.Sprintf("- Label: %s\n", label))
	if task.Name != "" {
		sb.WriteString(fmt.Sprintf("- Agent: %s\n", task.Name))
	}
	sb.WriteString(fmt.Sprintf("- Status: %s\n", task.Status))
	sb.WriteString(fmt.Sprintf("- Created: %s\n", created))
	if task.Directory != "" {
		sb.WriteString(fmt.Sprintf("- Directory: %s\n", task.Directory))
	}
	sb.WriteString(fmt.Sprintf("- Task: %s\n", task.Task))

	if task.Result != "" {
		sb.WriteString(fmt.Sprintf("\n**Result:**\n%s\n", task.Result))
	}

	return UserResult(sb.String())
}

// SubagentMessageTool sends guidance to a running subagent.
type SubagentMessageTool struct {
	manager *SubagentManager
}

func NewSubagentMessageTool(manager *SubagentManager) *SubagentMessageTool {
	return &SubagentMessageTool{
		manager: manager,
	}
}

func (t *SubagentMessageTool) Name() string {
	return "subagent_message"
}

func (t *SubagentMessageTool) Description() string {
	return "Send guidance or instructions to a running subagent task."
}

func (t *SubagentMessageTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task_id": map[string]interface{}{
				"type":        "string",
				"description": "The ID of the subagent task",
			},
			"message": map[string]interface{}{
				"type":        "string",
				"description": "The guidance message to send to the subagent",
			},
		},
		"required": []string{"task_id", "message"},
	}
}

func (t *SubagentMessageTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	taskID, ok := args["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return ErrorResult("task_id is required and cannot be empty")
	}

	message, ok := args["message"].(string)
	if !ok || strings.TrimSpace(message) == "" {
		return ErrorResult("message is required and cannot be empty")
	}

	if t.manager == nil {
		return ErrorResult("Subagent manager not configured")
	}

	err := t.manager.SendMessage(taskID, message)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to send message: %v", err))
	}

	return UserResult(fmt.Sprintf("Message sent to subagent '%s'", taskID))
}

// SubagentCancelTool cancels a running subagent task.
type SubagentCancelTool struct {
	manager *SubagentManager
}

func NewSubagentCancelTool(manager *SubagentManager) *SubagentCancelTool {
	return &SubagentCancelTool{
		manager: manager,
	}
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
		"properties": map[string]interface{}{
			"task_id": map[string]interface{}{
				"type":        "string",
				"description": "The ID of the subagent task to cancel",
			},
		},
		"required": []string{"task_id"},
	}
}

func (t *SubagentCancelTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	taskID, ok := args["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return ErrorResult("task_id is required and cannot be empty")
	}

	if t.manager == nil {
		return ErrorResult("Subagent manager not configured")
	}

	err := t.manager.CancelWithSource(taskID, "llm_tool:subagent_cancel")
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to cancel subagent: %v", err))
	}

	return UserResult(fmt.Sprintf("Subagent '%s' cancelled", taskID))
}
