package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

const emptySubagentResultMessage = "Subagent finished without textual summary. Check generated files and tool side effects."

type SubagentTask struct {
	ID            string
	Task          string
	Label         string
	Name          string // Named agent identity (optional)
	Directory     string // Project directory passed to subagent
	OriginChannel string
	OriginChatID  string
	Status        string
	Result        string
	Created       int64
	Started       int64    // When the task actually started execution
	Ended         int64    // When the task completed/failed/cancelled
	PendingMsgs   []string // Queued guidance messages from supervisor
	Notified      bool     // Terminal state notification already sent to main agent
}

type SubagentManager struct {
	tasks                   map[string]*SubagentTask
	cancels                 map[string]context.CancelFunc
	mu                      sync.RWMutex
	provider                providers.LLMProvider
	defaultModel            string
	bus                     *bus.MessageBus
	workspace               string
	tools                   *ToolRegistry
	cfg                     *config.Config
	maxIterations           int
	maxTokens               int
	contextLimit            int // Max total chars in subagent message history (0 = no limit)
	historyMessageThreshold int // Max number of messages before trimming (0 = no limit)
	maxConcurrentSubagents  int // Max number of concurrently running subagents
	nextID                  int
}

func NewSubagentManager(provider providers.LLMProvider, cfg *config.Config, workspace string, bus *bus.MessageBus) *SubagentManager {
	// Derive context limit from max output tokens.
	// Rough estimate: 4 chars/token × 20× input/output ratio.
	// E.g. MaxTokensSubagent=4096 → ~81K chars ≈ 20K tokens context.
	contextLimit := cfg.Agents.Defaults.MaxTokensSubagent * 20
	if contextLimit <= 0 {
		contextLimit = 4096 * 20 // fallback default
	}

	// Get default subagent config (for anonymous subagents)
	defaultCfg := cfg.Agents.ResolveAgentConfig("")
	msgThreshold := defaultCfg.HistoryMessageThreshold
	if msgThreshold <= 0 {
		msgThreshold = 100
	}

	return &SubagentManager{
		tasks:                   make(map[string]*SubagentTask),
		cancels:                 make(map[string]context.CancelFunc),
		provider:                provider,
		defaultModel:            cfg.Agents.Defaults.Model,
		bus:                     bus,
		workspace:               workspace,
		tools:                   NewToolRegistry(),
		cfg:                     cfg,
		maxIterations:           defaultCfg.MaxIterations,
		maxTokens:               defaultCfg.MaxTokens,
		contextLimit:            contextLimit,
		historyMessageThreshold: msgThreshold,
		maxConcurrentSubagents:  cfg.Agents.Defaults.MaxConcurrentSubagents,
		nextID:                  1,
	}
}

// SetTools sets the tool registry for subagent execution.
// If not set, subagent will have access to the provided tools.
func (sm *SubagentManager) SetTools(tools *ToolRegistry) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.tools = tools
}

// RegisterTool registers a tool for subagent execution.
func (sm *SubagentManager) RegisterTool(tool Tool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.tools.Register(tool)
}

// countRunningTasks returns the number of subagent tasks with status "running"
func (sm *SubagentManager) countRunningTasks() int {
	count := 0
	for _, task := range sm.tasks {
		if task.Status == "running" {
			count++
		}
	}
	return count
}

func (sm *SubagentManager) Spawn(ctx context.Context, task, label, name, directory, originChannel, originChatID string, callback AsyncCallback) (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check concurrent subagent limit
	runningCount := sm.countRunningTasks()
	if runningCount >= sm.maxConcurrentSubagents {
		return "", fmt.Errorf("maximum concurrent subagents limit reached (%d/%d running). Please wait for existing subagents to complete", runningCount, sm.maxConcurrentSubagents)
	}

	taskID := fmt.Sprintf("subagent-%d", sm.nextID)
	sm.nextID++

	subagentTask := &SubagentTask{
		ID:            taskID,
		Task:          task,
		Label:         label,
		Name:          name,
		Directory:     directory,
		OriginChannel: originChannel,
		OriginChatID:  originChatID,
		Status:        "running",
		Created:       time.Now().UnixMilli(),
	}
	sm.tasks[taskID] = subagentTask

	taskCtx, cancel := context.WithCancel(ctx)
	sm.cancels[taskID] = cancel

	// Start task in background with context cancellation support
	go sm.runTask(taskCtx, subagentTask, callback)

	nameTag := ""
	if name != "" {
		nameTag = fmt.Sprintf(" [agent: %s]", name)
	}
	switch {
	case label != "" && directory != "":
		return fmt.Sprintf("Spawned subagent '%s'%s in '%s' for task: %s", label, nameTag, directory, task), nil
	case label != "":
		return fmt.Sprintf("Spawned subagent '%s'%s for task: %s", label, nameTag, task), nil
	case directory != "":
		return fmt.Sprintf("Spawned subagent%s in '%s' for task: %s", nameTag, directory, task), nil
	default:
		return fmt.Sprintf("Spawned subagent%s for task: %s", nameTag, task), nil
	}
}

// buildSubagentSystemPrompt builds the system prompt for subagents.
// If SUBAGENTS.md exists in workspace, it replaces the hardcoded base prompt.
// If name is set, loads agent identity and memory from workspace/agents/<name>/.
// If directory is set and contains AGENTS.md, it is included as project context.
// Workspace bootstrap files (SOUL.md, USER.md etc.) are NOT included — those are for main agent only.
func (sm *SubagentManager) buildSubagentSystemPrompt(taskLabel, name, directory string) string {
	base := `You are a subagent. Complete the given task independently and report the result.
You have access to tools - use them as needed to complete your task.
After completing the task, provide a clear summary of what was done.`

	if data, err := os.ReadFile(filepath.Join(sm.workspace, "SUBAGENTS.md")); err == nil {
		base = string(data)
	}

	if name != "" {
		// Security: reject names with path separators or dots to prevent traversal
		if strings.ContainsAny(name, "/\\.") {
			logger.WarnCF("subagent", "Invalid agent name ignored (path traversal attempt)",
				map[string]interface{}{"name": name})
			name = ""
		}
	}

	if name != "" {
		agentDir := filepath.Join(sm.workspace, "agents", name)
		// Load agent identity
		if data, err := os.ReadFile(filepath.Join(agentDir, "AGENTS.md")); err == nil {
			base += fmt.Sprintf("\n\n## Agent Identity: %s\n\n%s", name, string(data))
		}
		// Load agent memory
		if data, err := os.ReadFile(filepath.Join(agentDir, "memory", "MEMORY.md")); err == nil {
			base += "\n\n## Your Memory\n\n" + string(data)
		}
		// Memory save instructions
		now := time.Now()
		memoryDir := filepath.Join(sm.workspace, "agents", name, "memory")
		base += fmt.Sprintf("\n\n## Memory Instructions\nYou are a named agent with persistent memory. At the end of this task, save key learnings to your memory using write_file/append_file:\n- Long-term: %s/MEMORY.md\n- Daily notes: %s/%s/%s.md",
			memoryDir, memoryDir, now.Format("200601"), now.Format("20060102"))
	}

	if taskLabel != "" {
		base += fmt.Sprintf("\n\n## Task Label\n%s", taskLabel)
	}

	if directory != "" {
		resolvedDir := directory
		if !filepath.IsAbs(resolvedDir) {
			resolvedDir = filepath.Join(sm.workspace, directory)
		}
		base += fmt.Sprintf("\n\n## Project Directory\n%s", resolvedDir)
		if data, err := os.ReadFile(filepath.Join(resolvedDir, "AGENTS.md")); err == nil {
			base += fmt.Sprintf("\n\n## AGENTS.md\n\n%s", string(data))
		}
	}

	return base
}

func (sm *SubagentManager) runTask(ctx context.Context, task *SubagentTask, callback AsyncCallback) {
	sm.mu.Lock()
	task.Status = "running"
	task.Created = time.Now().UnixMilli()
	task.Started = time.Now().UnixMilli()
	sm.mu.Unlock()

	// Fail-safe: never let a subagent terminate without publishing terminal state.
	defer func() {
		if r := recover(); r != nil {
			sm.mu.Lock()
			if task.Status == "running" || task.Ended == 0 {
				task.Status = "failed"
				task.Result = fmt.Sprintf("Panic in subagent execution: %v", r)
				task.Ended = time.Now().UnixMilli()
			}
			delete(sm.cancels, task.ID)
			sm.mu.Unlock()
			sm.publishTaskUpdate(task)
			logger.ErrorCF("subagent", "Subagent task panicked", map[string]interface{}{
				"task_id": task.ID,
				"panic":   fmt.Sprint(r),
			})
		}
	}()

	logger.InfoCF("subagent", "Starting subagent task",
		map[string]interface{}{
			"task_id":   task.ID,
			"label":     task.Label,
			"name":      task.Name,
			"directory": task.Directory,
		})

	systemPrompt := sm.buildSubagentSystemPrompt(task.Label, task.Name, task.Directory)

	messages := []providers.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: task.Task,
		},
	}

	// Check if context is already cancelled before starting
	select {
	case <-ctx.Done():
		sm.mu.Lock()
		task.Status = "cancelled"
		task.Result = "Task cancelled before execution"
		task.Ended = time.Now().UnixMilli()
		delete(sm.cancels, task.ID)
		sm.mu.Unlock()
		sm.publishTaskUpdate(task)
		return
	default:
	}

	// Resolve config based on agent name
	// Priority: named_agent > subagents > defaults
	resolvedCfg := sm.cfg.Agents.ResolveAgentConfig(task.Name)
	maxIter := resolvedCfg.MaxIterations
	maxTok := resolvedCfg.MaxTokens
	msgThreshold := resolvedCfg.HistoryMessageThreshold
	temperature := resolvedCfg.Temperature

	// Run tool loop with access to tools
	sm.mu.RLock()
	tools := sm.tools
	sm.mu.RUnlock()

	loopResult, err := RunToolLoop(ctx, ToolLoopConfig{
		Provider:      sm.provider,
		Model:         sm.defaultModel,
		Tools:         tools,
		MaxIterations: maxIter,
		RunID:         task.ID,
		PullInjectedMessages: func() []string {
			return sm.drainPendingMessages(task.ID)
		},
		ContextLimit:            sm.contextLimit,
		HistoryMessageThreshold: msgThreshold,
		LLMOptions: map[string]any{
			"max_tokens":  maxTok,
			"temperature": temperature,
		},
	}, messages, task.OriginChannel, task.OriginChatID)

	sm.mu.Lock()
	var result *ToolResult
	task.Ended = time.Now().UnixMilli()

	if err != nil {
		task.Status = "failed"
		task.Result = fmt.Sprintf("Error: %v", err)
		// Check if it was cancelled
		if ctx.Err() != nil {
			task.Status = "cancelled"
			task.Result = "Task cancelled during execution"
		}
		result = &ToolResult{
			ForLLM:  task.Result,
			ForUser: "",
			Silent:  false,
			IsError: true,
			Async:   false,
			Err:     err,
		}
	} else {
		resultContent := strings.TrimSpace(loopResult.Content)
		if resultContent == "" {
			resultContent = emptySubagentResultMessage
		}
		task.Status = "completed"
		task.Result = resultContent
		result = &ToolResult{
			ForLLM:  fmt.Sprintf("Subagent '%s' completed (iterations: %d): %s", task.Label, loopResult.Iterations, resultContent),
			ForUser: resultContent,
			Silent:  false,
			IsError: false,
			Async:   false,
		}
	}

	// IMPORTANT: release manager lock before publish/callback to avoid lock re-entry deadlocks.
	delete(sm.cancels, task.ID)
	sm.mu.Unlock()

	// Send announce message back to main agent.
	sm.publishTaskUpdate(task)

	// Call callback if provided and result is set.
	if callback != nil && result != nil {
		callback(ctx, result)
	}
}

func sanitizeAgentName(name string) string {
	if name == "" {
		return ""
	}
	if strings.ContainsAny(name, "/\\.") {
		logger.WarnCF("subagent", "Invalid agent name ignored (path traversal attempt)",
			map[string]interface{}{"name": name})
		return ""
	}
	return name
}

func (sm *SubagentManager) resolveWriteRoots(name, directory string) []string {
	roots := make([]string, 0, 3)

	if directory != "" {
		resolvedDir := directory
		if !filepath.IsAbs(resolvedDir) {
			resolvedDir = filepath.Join(sm.workspace, resolvedDir)
		}
		if absDir, err := filepath.Abs(resolvedDir); err == nil {
			roots = append(roots, absDir)
		}
	}

	if name != "" {
		memoryDir := filepath.Join(sm.workspace, "agents", name, "memory")
		if absMemory, err := filepath.Abs(memoryDir); err == nil {
			roots = append(roots, absMemory)
		}

		if name == "picoclaw-self-update" {
			projectDir := filepath.Join(sm.workspace, "projects", "picoclaw")
			if absProject, err := filepath.Abs(projectDir); err == nil {
				roots = append(roots, absProject)
			}
		}
	}

	return roots
}

func (sm *SubagentManager) GetTask(taskID string) (*SubagentTask, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	task, ok := sm.tasks[taskID]
	return task, ok
}

func (sm *SubagentManager) ListTasks() []*SubagentTask {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	tasks := make([]*SubagentTask, 0, len(sm.tasks))
	for _, task := range sm.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// CountRunning returns the number of currently running subagent tasks.
func (sm *SubagentManager) CountRunning() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.countRunningTasks()
}

// GetRunningTasks returns all currently running tasks.
func (sm *SubagentManager) GetRunningTasks() []*SubagentTask {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	tasks := make([]*SubagentTask, 0)
	for _, task := range sm.tasks {
		if task.Status == "running" {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// GetRecentTasks returns the most recent non-running tasks, up to limit.
func (sm *SubagentManager) GetRecentTasks(limit int) []*SubagentTask {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Get all non-running tasks
	var completed []*SubagentTask
	for _, task := range sm.tasks {
		if task.Status != "running" {
			completed = append(completed, task)
		}
	}

	// Sort by Ended time (most recent first)
	for i := 0; i < len(completed); i++ {
		for j := i + 1; j < len(completed); j++ {
			if completed[j].Ended > completed[i].Ended {
				completed[i], completed[j] = completed[j], completed[i]
			}
		}
	}

	// Limit results
	if len(completed) > limit {
		completed = completed[:limit]
	}
	return completed
}

// GetMessageQueueCount returns the total count of pending messages across all running subagents.
func (sm *SubagentManager) GetMessageQueueCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	count := 0
	for _, task := range sm.tasks {
		if task.Status == "running" {
			count += len(task.PendingMsgs)
		}
	}
	return count
}

// GetFirstQueuedMessage returns the first queued message from any running subagent.
func (sm *SubagentManager) GetFirstQueuedMessage() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, task := range sm.tasks {
		if task.Status == "running" && len(task.PendingMsgs) > 0 {
			return task.PendingMsgs[0]
		}
	}
	return ""
}

// SendMessage sends a guidance message to a running subagent task.
func (sm *SubagentManager) SendMessage(taskID, message string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	task, ok := sm.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found")
	}
	if task.Status != "running" {
		return fmt.Errorf("task is not running (status: %s)", task.Status)
	}

	// Inject supervisor correction into active loop on next iteration.
	task.PendingMsgs = append(task.PendingMsgs, fmt.Sprintf(
		"<supervisor_message priority=\"high\" source=\"subagent_message\" task_id=\"%s\">\n%s\n</supervisor_message>\nApply this supervisor correction immediately before continuing.",
		taskID,
		message,
	))
	return nil
}

func (sm *SubagentManager) drainPendingMessages(taskID string) []string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	task, ok := sm.tasks[taskID]
	if !ok || len(task.PendingMsgs) == 0 {
		return nil
	}

	items := append([]string(nil), task.PendingMsgs...)
	task.PendingMsgs = task.PendingMsgs[:0]
	return items
}

// Cancel cancels a running subagent task.
func (sm *SubagentManager) Cancel(taskID string) error {
	sm.mu.Lock()

	task, ok := sm.tasks[taskID]
	if !ok {
		sm.mu.Unlock()
		return fmt.Errorf("task not found")
	}
	if task.Status != "running" {
		sm.mu.Unlock()
		return fmt.Errorf("task is not running (status: %s)", task.Status)
	}

	// Mark as cancelled
	task.Status = "cancelled"
	task.Result = "Cancelled by user"
	task.Ended = time.Now().UnixMilli()
	cancel := sm.cancels[taskID]
	sm.mu.Unlock()

	// Trigger actual cancellation for the running task context.
	if cancel != nil {
		cancel()
	}

	// Publish terminal update immediately. This guarantees supervisor visibility
	// even when provider/tool execution does not stop promptly after cancellation.
	sm.publishTaskUpdate(task)
	return nil
}

func (sm *SubagentManager) publishTaskUpdate(task *SubagentTask) {
	if sm.bus == nil || task == nil {
		return
	}

	sm.mu.Lock()
	if task.Notified {
		sm.mu.Unlock()
		return
	}

	status := task.Status
	switch status {
	case "completed", "failed", "cancelled":
	default:
		status = "finished"
	}

	task.Notified = true
	taskID := task.ID
	taskLabel := task.Label
	taskName := task.Name
	taskResult := task.Result
	originChannel := task.OriginChannel
	originChatID := task.OriginChatID
	sm.mu.Unlock()

	var announceContent string
	if taskName != "" {
		announceContent = fmt.Sprintf("Subagent ID: %s\nTask '%s' [agent: %s] %s.\n\nResult:\n%s", taskID, taskLabel, taskName, status, taskResult)
	} else {
		announceContent = fmt.Sprintf("Subagent ID: %s\nTask '%s' %s.\n\nResult:\n%s", taskID, taskLabel, status, taskResult)
	}

	inbound := bus.InboundMessage{
		Channel:  "system",
		SenderID: fmt.Sprintf("subagent:%s", taskID),
		// Format: "original_channel:original_chat_id" for routing back
		ChatID:  fmt.Sprintf("%s:%s", originChannel, originChatID),
		Content: announceContent,
		Metadata: map[string]string{
			"source":      "subagent:terminal",
			"subagent_id": taskID,
			"status":      status,
		},
	}

	if ok := sm.bus.PublishInbound(inbound); !ok {
		logger.WarnCF("subagent", "Subagent completion delayed: inbound queue timeout, using critical fallback", map[string]interface{}{
			"task_id": taskID,
		})
		if criticalOK := sm.bus.PublishInboundCritical(inbound); !criticalOK {
			logger.ErrorCF("subagent", "Subagent completion delivery failed: inbound bus closed", map[string]interface{}{
				"task_id": taskID,
			})
		}
	}
}

// SubagentTool executes a subagent task synchronously and returns the result.
// Unlike SpawnTool which runs tasks asynchronously, SubagentTool waits for completion
// and returns the result directly in the ToolResult.
type SubagentTool struct {
	manager       *SubagentManager
	originChannel string
	originChatID  string
}

func NewSubagentTool(manager *SubagentManager) *SubagentTool {
	return &SubagentTool{
		manager:       manager,
		originChannel: "cli",
		originChatID:  "direct",
	}
}

func (t *SubagentTool) Name() string {
	return "subagent"
}

func (t *SubagentTool) Description() string {
	return "Execute a subagent task synchronously and return the result. Use this for delegating specific tasks to an independent agent instance. Returns execution summary to user and full details to LLM."
}

func (t *SubagentTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task": map[string]interface{}{
				"type":        "string",
				"description": "The task for subagent to complete",
			},
			"label": map[string]interface{}{
				"type":        "string",
				"description": "Optional short label for the task (for display)",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Optional named agent. If set, loads identity and memory from workspace/agents/<name>/. Agent saves learnings to its memory after each task.",
			},
			"directory": map[string]interface{}{
				"type":        "string",
				"description": "Optional project directory. If AGENTS.md exists there, it will be included. Relative paths are resolved against workspace.",
			},
		},
		"required": []string{"task"},
	}
}

func (t *SubagentTool) SetContext(channel, chatID string) {
	t.originChannel = channel
	t.originChatID = chatID
}

func (t *SubagentTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	task, ok := args["task"].(string)
	if !ok {
		return ErrorResult("task is required").WithError(fmt.Errorf("task parameter is required"))
	}

	label, _ := args["label"].(string)
	name, _ := args["name"].(string)
	directory, _ := args["directory"].(string)
	name = sanitizeAgentName(name)

	if t.manager == nil {
		return ErrorResult("Subagent manager not configured").WithError(fmt.Errorf("manager is nil"))
	}

	systemPrompt := t.manager.buildSubagentSystemPrompt(label, name, directory)

	// Build messages for subagent
	messages := []providers.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: task,
		},
	}

	// Resolve config based on agent name
	// Priority: named_agent > subagents > defaults
	sm := t.manager
	resolvedCfg := sm.cfg.Agents.ResolveAgentConfig(name)
	maxIter := resolvedCfg.MaxIterations
	maxTok := resolvedCfg.MaxTokens
	msgThreshold := resolvedCfg.HistoryMessageThreshold
	temperature := resolvedCfg.Temperature

	sm.mu.RLock()
	tools := sm.tools
	sm.mu.RUnlock()

	scopedCtx := WithWriteScope(ctx, sm.resolveWriteRoots(name, directory))
	loopResult, err := RunToolLoop(scopedCtx, ToolLoopConfig{
		Provider:                sm.provider,
		Model:                   sm.defaultModel,
		Tools:                   tools,
		MaxIterations:           maxIter,
		RunID:                   fmt.Sprintf("sync-%s", name),
		ContextLimit:            sm.contextLimit,
		HistoryMessageThreshold: msgThreshold,
		LLMOptions: map[string]any{
			"max_tokens":  maxTok,
			"temperature": temperature,
		},
	}, messages, t.originChannel, t.originChatID)

	if err != nil {
		return ErrorResult(fmt.Sprintf("Subagent execution failed: %v", err)).WithError(err)
	}

	resultContent := strings.TrimSpace(loopResult.Content)
	if resultContent == "" {
		resultContent = emptySubagentResultMessage
	}

	// ForUser: Brief summary for user (truncated if too long)
	userContent := resultContent
	maxUserLen := 500
	if len(userContent) > maxUserLen {
		userContent = userContent[:maxUserLen] + "..."
	}

	// ForLLM: Full execution details
	labelStr := label
	if labelStr == "" {
		labelStr = "(unnamed)"
	}
	llmContent := fmt.Sprintf("Subagent task completed:\nLabel: %s\nIterations: %d\nResult: %s",
		labelStr, loopResult.Iterations, resultContent)

	return &ToolResult{
		ForLLM:  llmContent,
		ForUser: userContent,
		Silent:  false,
		IsError: false,
		Async:   false,
	}
}
