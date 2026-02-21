package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type ContextBuilder struct {
	workspace    string
	skillsLoader *skills.SkillsLoader
	memory       *MemoryStore
	tools        *tools.ToolRegistry // Direct reference to tool registry
	agentsMu     sync.RWMutex
	knownAgents  []tools.AgentInfo
}

type systemPromptBuildStats struct {
	IdentityChars       int
	ToolsSectionChars   int
	BootstrapTotalChars int
	BootstrapByFile     map[string]int
	SkillsChars         int
	NamedAgentsChars    int
	MemoryChars         int
	MemoryLongTermChars int
	MemoryRecentChars   int
}

func getGlobalConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".picoclaw")
}

func NewContextBuilder(workspace string) *ContextBuilder {
	// builtin skills: skills directory in current project
	// Use the skills/ directory under the current working directory
	wd, _ := os.Getwd()
	builtinSkillsDir := filepath.Join(wd, "skills")
	globalSkillsDir := filepath.Join(getGlobalConfigDir(), "skills")

	return &ContextBuilder{
		workspace:    workspace,
		skillsLoader: skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir),
		memory:       NewMemoryStore(workspace),
	}
}

// SetToolsRegistry sets the tools registry for dynamic tool summary generation.
func (cb *ContextBuilder) SetToolsRegistry(registry *tools.ToolRegistry) {
	cb.tools = registry
}

func (cb *ContextBuilder) getIdentity() string {
	now := time.Now().Format("2006-01-02 15:04 (Monday)")
	workspacePath, _ := filepath.Abs(filepath.Join(cb.workspace))
	runtime := fmt.Sprintf("%s %s, Go %s", runtime.GOOS, runtime.GOARCH, runtime.Version())

	// Build tools section dynamically
	toolsSection := cb.buildToolsSection()

	return fmt.Sprintf(`# picoclaw 🦞

You are picoclaw, a helpful AI assistant.

## Current Time
%s

## Runtime
%s

## Workspace
Your workspace is at: %s
- Memory: %s/memory/MEMORY.md
- Daily Notes: %s/memory/YYYYMM/YYYYMMDD.md
- Skills: %s/skills/{skill-name}/SKILL.md

%s

## Important Rules

1. **ALWAYS use tools** - When you need to perform an action (schedule reminders, send messages, execute commands, etc.), you MUST call the appropriate tool. Do NOT just say you'll do it or pretend to do it.

2. **Be helpful and accurate** - When using tools, briefly explain what you're doing.

3. **Memory** - When remembering something, write to %s/memory/MEMORY.md`,
		now, runtime, workspacePath, workspacePath, workspacePath, workspacePath, toolsSection, workspacePath)
}

func (cb *ContextBuilder) getIdentityWithStats() (string, int) {
	now := time.Now().Format("2006-01-02 15:04 (Monday)")
	workspacePath, _ := filepath.Abs(filepath.Join(cb.workspace))
	runtime := fmt.Sprintf("%s %s, Go %s", runtime.GOOS, runtime.GOARCH, runtime.Version())

	// Build tools section dynamically
	toolsSection := cb.buildToolsSection()
	prompt := fmt.Sprintf(`# picoclaw 🦞

You are picoclaw, a helpful AI assistant.

## Current Time
%s

## Runtime
%s

## Workspace
Your workspace is at: %s
- Memory: %s/memory/MEMORY.md
- Daily Notes: %s/memory/YYYYMM/YYYYMMDD.md
- Skills: %s/skills/{skill-name}/SKILL.md

%s

## Important Rules

1. **ALWAYS use tools** - When you need to perform an action (schedule reminders, send messages, execute commands, etc.), you MUST call the appropriate tool. Do NOT just say you'll do it or pretend to do it.

2. **Be helpful and accurate** - When using tools, briefly explain what you're doing.

3. **Memory** - When remembering something, write to %s/memory/MEMORY.md`,
		now, runtime, workspacePath, workspacePath, workspacePath, workspacePath, toolsSection, workspacePath)

	return prompt, len(toolsSection)
}

func (cb *ContextBuilder) refreshNamedAgents() []tools.AgentInfo {
	agents := tools.LoadAvailableAgents(cb.workspace)
	cb.agentsMu.Lock()
	cb.knownAgents = append([]tools.AgentInfo(nil), agents...)
	cb.agentsMu.Unlock()
	return agents
}

func (cb *ContextBuilder) getKnownAgents() []tools.AgentInfo {
	cb.agentsMu.RLock()
	defer cb.agentsMu.RUnlock()
	return append([]tools.AgentInfo(nil), cb.knownAgents...)
}

func (cb *ContextBuilder) GetNamedAgentsInfo(refresh bool) map[string]interface{} {
	var agents []tools.AgentInfo
	if refresh {
		agents = cb.refreshNamedAgents()
	} else {
		agents = cb.getKnownAgents()
		if len(agents) == 0 {
			agents = cb.refreshNamedAgents()
		}
	}

	names := make([]string, 0, len(agents))
	for _, a := range agents {
		names = append(names, a.Name)
	}
	return map[string]interface{}{
		"count": len(names),
		"names": names,
	}
}

func (cb *ContextBuilder) buildNamedAgentsSummary() string {
	agents := cb.refreshNamedAgents()
	if len(agents) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# Named Agents\n\n")
	sb.WriteString("The following named agents are available. Use the `name` parameter in `subagent` or `spawn` tools to delegate tasks to them. Each named agent carries its own identity and persistent memory.\n\n")
	for _, a := range agents {
		sb.WriteString(fmt.Sprintf("- **%s** — %s\n", a.Name, a.Description))
	}
	return sb.String()
}

func (cb *ContextBuilder) buildToolsSection() string {
	if cb.tools == nil {
		return ""
	}

	summaries := cb.tools.GetSummaries()
	if len(summaries) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Available Tools\n\n")
	sb.WriteString("**CRITICAL**: You MUST use tools to perform actions. Do NOT pretend to execute commands or schedule tasks.\n\n")
	sb.WriteString("You have access to the following tools:\n\n")
	for _, s := range summaries {
		sb.WriteString(s)
		sb.WriteString("\n")
	}

	return sb.String()
}

func (cb *ContextBuilder) BuildSystemPrompt() string {
	systemPrompt, _ := cb.buildSystemPromptWithStats()
	return systemPrompt
}

func (cb *ContextBuilder) buildSystemPromptWithStats() (string, systemPromptBuildStats) {
	parts := []string{}
	stats := systemPromptBuildStats{
		BootstrapByFile: make(map[string]int),
	}

	// Core identity section
	identity, toolsSectionChars := cb.getIdentityWithStats()
	parts = append(parts, identity)
	stats.IdentityChars = len(identity)
	stats.ToolsSectionChars = toolsSectionChars

	// Bootstrap files
	bootstrapContent, bootstrapSizes := cb.LoadBootstrapFilesDetailed()
	if bootstrapContent != "" {
		parts = append(parts, bootstrapContent)
		stats.BootstrapTotalChars = len(bootstrapContent)
		for k, v := range bootstrapSizes {
			stats.BootstrapByFile[k] = v
		}
	}

	// Skills - show summary, AI can read full content with read_file tool
	skillsSummary := cb.skillsLoader.BuildSkillsSummary()
	if skillsSummary != "" {
		skillsBlock := fmt.Sprintf(`# Skills

The following skills extend your capabilities. To use a skill, read its SKILL.md file using the read_file tool.

%s`, skillsSummary)
		parts = append(parts, skillsBlock)
		stats.SkillsChars = len(skillsBlock)
	}

	// Named agents available for delegation
	agentsSummary := cb.buildNamedAgentsSummary()
	if agentsSummary != "" {
		parts = append(parts, agentsSummary)
		stats.NamedAgentsChars = len(agentsSummary)
	}

	// Memory context
	memoryContext, memoryStats := cb.memory.GetMemoryContextWithStats()
	if memoryContext != "" {
		parts = append(parts, "# Memory\n\n"+memoryContext)
		stats.MemoryChars = len("# Memory\n\n" + memoryContext)
	}
	stats.MemoryLongTermChars = memoryStats["long_term_chars"]
	stats.MemoryRecentChars = memoryStats["recent_notes_chars"]

	// Join with "---" separator
	return strings.Join(parts, "\n\n---\n\n"), stats
}

func (cb *ContextBuilder) LoadBootstrapFiles() string {
	content, _ := cb.LoadBootstrapFilesDetailed()
	return content
}

func (cb *ContextBuilder) LoadBootstrapFilesDetailed() (string, map[string]int) {
	bootstrapFiles := []string{
		"AGENTS.md",
		"SOUL.md",
		"USER.md",
		"IDENTITY.md",
	}

	var result string
	sizes := make(map[string]int, len(bootstrapFiles))
	for _, filename := range bootstrapFiles {
		filePath := filepath.Join(cb.workspace, filename)
		if data, err := os.ReadFile(filePath); err == nil {
			result += fmt.Sprintf("## %s\n\n%s\n\n", filename, string(data))
			sizes[filename] = len(data)
		}
	}

	return result, sizes
}

func (cb *ContextBuilder) BuildMessages(history []providers.Message, summary string, currentMessage string, media []string, channel, chatID string) []providers.Message {
	messages := []providers.Message{}

	systemPrompt, stats := cb.buildSystemPromptWithStats()

	// Add Current Session info if provided
	if channel != "" && chatID != "" {
		systemPrompt += fmt.Sprintf("\n\n## Current Session\nChannel: %s\nChat ID: %s", channel, chatID)
	}

	// Log system prompt summary for debugging (debug mode only)
	logger.DebugCF("agent", "System prompt built",
		map[string]interface{}{
			"total_chars":              len(systemPrompt),
			"total_lines":              strings.Count(systemPrompt, "\n") + 1,
			"section_count":            strings.Count(systemPrompt, "\n\n---\n\n") + 1,
			"identity_chars":           stats.IdentityChars,
			"tools_section_chars":      stats.ToolsSectionChars,
			"bootstrap_total_chars":    stats.BootstrapTotalChars,
			"bootstrap_agents_chars":   stats.BootstrapByFile["AGENTS.md"],
			"bootstrap_soul_chars":     stats.BootstrapByFile["SOUL.md"],
			"bootstrap_user_chars":     stats.BootstrapByFile["USER.md"],
			"bootstrap_identity_chars": stats.BootstrapByFile["IDENTITY.md"],
			"skills_chars":             stats.SkillsChars,
			"named_agents_chars":       stats.NamedAgentsChars,
			"memory_total_chars":       stats.MemoryChars,
			"memory_long_term_chars":   stats.MemoryLongTermChars,
			"memory_recent_chars":      stats.MemoryRecentChars,
		})

	if summary != "" {
		systemPrompt += "\n\n## Summary of Previous Conversation\n\n" + summary
	}

	//This fix prevents the session memory from LLM failure due to elimination of toolu_IDs required from LLM
	// --- INICIO DEL FIX ---
	//Diegox-17
	for len(history) > 0 && (history[0].Role == "tool") {
		logger.DebugCF("agent", "Removing orphaned tool message from history to prevent LLM error",
			map[string]interface{}{"role": history[0].Role})
		history = history[1:]
	}
	//Diegox-17
	// --- FIN DEL FIX ---

	messages = append(messages, providers.Message{
		Role:    "system",
		Content: systemPrompt,
	})

	messages = append(messages, history...)

	messages = append(messages, providers.Message{
		Role:    "user",
		Content: currentMessage,
	})

	return messages
}

func (cb *ContextBuilder) AddToolResult(messages []providers.Message, toolCallID, toolName, result string) []providers.Message {
	messages = append(messages, providers.Message{
		Role:       "tool",
		Content:    result,
		ToolCallID: toolCallID,
	})
	return messages
}

func (cb *ContextBuilder) AddAssistantMessage(messages []providers.Message, content string, toolCalls []map[string]interface{}) []providers.Message {
	msg := providers.Message{
		Role:    "assistant",
		Content: content,
	}
	// Always add assistant message, whether or not it has tool calls
	messages = append(messages, msg)
	return messages
}

func (cb *ContextBuilder) loadSkills() string {
	allSkills := cb.skillsLoader.ListSkills()
	if len(allSkills) == 0 {
		return ""
	}

	var skillNames []string
	for _, s := range allSkills {
		skillNames = append(skillNames, s.Name)
	}

	content := cb.skillsLoader.LoadSkillsForContext(skillNames)
	if content == "" {
		return ""
	}

	return "# Skill Definitions\n\n" + content
}

// GetSkillsInfo returns information about loaded skills.
func (cb *ContextBuilder) GetSkillsInfo() map[string]interface{} {
	allSkills := cb.skillsLoader.ListSkills()
	skillNames := make([]string, 0, len(allSkills))
	for _, s := range allSkills {
		skillNames = append(skillNames, s.Name)
	}
	return map[string]interface{}{
		"total":     len(allSkills),
		"available": len(allSkills),
		"names":     skillNames,
	}
}
