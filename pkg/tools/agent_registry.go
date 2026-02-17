package tools

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AgentInfo holds the discoverable metadata of a named agent.
type AgentInfo struct {
	Name        string
	Description string
}

var agentNamePattern = regexp.MustCompile(`^[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*$`)

// LoadAvailableAgents scans workspace/agents/*/AGENTS.md for YAML frontmatter
// and returns a list of named agents with name and description.
// Agents without valid YAML frontmatter (name + description) are silently skipped.
func LoadAvailableAgents(workspace string) []AgentInfo {
	agentsDir := filepath.Join(workspace, "agents")
	dirs, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil
	}

	var agents []AgentInfo
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		agentsFile := filepath.Join(agentsDir, dir.Name(), "AGENTS.md")
		data, err := os.ReadFile(agentsFile)
		if err != nil {
			continue
		}
		meta := parseAgentFrontmatter(string(data))
		if meta == nil {
			continue
		}
		agents = append(agents, *meta)
	}
	return agents
}

// parseAgentFrontmatter extracts name and description from YAML frontmatter.
// Returns nil if frontmatter is missing or invalid.
func parseAgentFrontmatter(content string) *AgentInfo {
	fm := extractFrontmatter(content)
	if fm == "" {
		return nil
	}
	kv := parseSimpleYAML(fm)
	name := strings.TrimSpace(kv["name"])
	description := strings.TrimSpace(kv["description"])
	if name == "" || description == "" {
		return nil
	}
	// Validate name — same security rule as buildSubagentSystemPrompt
	if !agentNamePattern.MatchString(name) {
		return nil
	}
	return &AgentInfo{Name: name, Description: description}
}

func extractFrontmatter(content string) string {
	re := regexp.MustCompile(`(?s)^---\n(.*?)\n---`)
	match := re.FindStringSubmatch(content)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func parseSimpleYAML(content string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			value = strings.Trim(value, "\"'")
			result[key] = value
		}
	}
	return result
}
