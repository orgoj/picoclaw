package preflight

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Issue struct {
	Check   string
	Path    string
	Message string
	Hint    string
}

type Report struct {
	Issues []Issue
}

func (r *Report) HasIssues() bool {
	return len(r.Issues) > 0
}

func (r *Report) Error() string {
	if len(r.Issues) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("preflight failed with %d issue(s):\n", len(r.Issues)))
	for i, issue := range r.Issues {
		sb.WriteString(fmt.Sprintf("%d) [%s] %s\n", i+1, issue.Check, issue.Message))
		if issue.Path != "" {
			sb.WriteString(fmt.Sprintf("   path: %s\n", issue.Path))
		}
		if issue.Hint != "" {
			sb.WriteString(fmt.Sprintf("   hint: %s\n", issue.Hint))
		}
	}
	return sb.String()
}

func Run(workspace string) error {
	report := &Report{}

	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return fmt.Errorf("failed to resolve workspace path: %w", err)
	}

	checkWorkspaceBootstrap(absWorkspace, report)
	checkBadProjectAgentDirs(absWorkspace, report)
	lintSkills(absWorkspace, report)

	if report.HasIssues() {
		return report
	}
	return nil
}

func checkWorkspaceBootstrap(workspace string, report *Report) {
	if fi, err := os.Stat(workspace); err != nil {
		report.Issues = append(report.Issues, Issue{
			Check:   "workspace",
			Path:    workspace,
			Message: "workspace directory does not exist",
			Hint:    "run `picoclaw onboard` or fix agents.defaults.workspace in config",
		})
		return
	} else if !fi.IsDir() {
		report.Issues = append(report.Issues, Issue{
			Check:   "workspace",
			Path:    workspace,
			Message: "workspace path is not a directory",
			Hint:    "set agents.defaults.workspace to a valid directory",
		})
		return
	}

	requiredFiles := []string{
		"AGENT.md",
		"IDENTITY.md",
		"SOUL.md",
		"USER.md",
		filepath.Join("memory", "MEMORY.md"),
	}

	for _, rel := range requiredFiles {
		full := filepath.Join(workspace, rel)
		if fi, err := os.Stat(full); err != nil || fi.IsDir() {
			report.Issues = append(report.Issues, Issue{
				Check:   "bootstrap",
				Path:    full,
				Message: "required workspace bootstrap file is missing",
				Hint:    "restore workspace templates or rerun `picoclaw onboard`",
			})
		}
	}
}

func checkBadProjectAgentDirs(workspace string, report *Report) {
	projectsDir := filepath.Join(workspace, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		report.Issues = append(report.Issues, Issue{
			Check:   "projects",
			Path:    projectsDir,
			Message: fmt.Sprintf("failed to read projects directory: %v", err),
			Hint:    "fix workspace/projects permissions",
		})
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		bad := filepath.Join(projectsDir, entry.Name(), "agents")
		if fi, err := os.Stat(bad); err == nil && fi.IsDir() {
			report.Issues = append(report.Issues, Issue{
				Check:   "project-agents-dir",
				Path:    bad,
				Message: "invalid agent directory under workspace/projects/*/agents",
				Hint:    "move named agents to workspace/agents/<agent>",
			})
		}
	}
}

func lintSkills(workspace string, report *Report) {
	skillFiles := collectSkillFiles(workspace)
	for _, skillFile := range skillFiles {
		data, err := os.ReadFile(skillFile)
		if err != nil {
			report.Issues = append(report.Issues, Issue{
				Check:   "skill-read",
				Path:    skillFile,
				Message: fmt.Sprintf("failed to read SKILL.md: %v", err),
				Hint:    "fix file permissions or remove broken skill",
			})
			continue
		}

		content := string(data)
		if err := validateSkillFrontmatter(content); err != nil {
			report.Issues = append(report.Issues, Issue{
				Check:   "skill-frontmatter",
				Path:    skillFile,
				Message: err.Error(),
				Hint:    "add valid YAML frontmatter with `name` and `description`",
			})
		}

		for _, violation := range findInstructionViolations(content) {
			report.Issues = append(report.Issues, Issue{
				Check:   "skill-instruction",
				Path:    skillFile,
				Message: violation,
				Hint:    "replace blocked command/path with safe instruction",
			})
		}
	}
}

func collectSkillFiles(workspace string) []string {
	roots := []string{
		filepath.Join(workspace, "skills"),
		filepath.Join(workspace, "projects"),
	}

	var files []string
	for _, root := range roots {
		if _, err := os.Stat(root); err != nil {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				if name == ".git" || name == "node_modules" || name == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Base(path) == "SKILL.md" {
				files = append(files, path)
			}
			return nil
		})
	}

	sort.Strings(files)
	return files
}

func validateSkillFrontmatter(content string) error {
	if !strings.HasPrefix(content, "---\n") {
		return fmt.Errorf("missing YAML frontmatter opening (`---`)")
	}

	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return fmt.Errorf("missing YAML frontmatter closing (`---`)")
	}

	frontmatter := content[4 : 4+end]
	meta := parseSimpleYAML(frontmatter)
	if strings.TrimSpace(meta["name"]) == "" {
		return fmt.Errorf("frontmatter missing `name`")
	}
	if strings.TrimSpace(meta["description"]) == "" {
		return fmt.Errorf("frontmatter missing `description`")
	}
	return nil
}

func parseSimpleYAML(content string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"'")
		result[key] = value
	}
	return result
}

var violationChecks = []struct {
	re      *regexp.Regexp
	message string
}{
	{regexp.MustCompile(`(?mi)\bgrep\s+-[A-Za-z]*r[A-Za-z]*\b`), "blocked command `grep -r` in instructions"},
	{regexp.MustCompile(`(?mi)\bfind\s+\.\s+-name\b`), "blocked command `find . -name` in instructions"},
	{regexp.MustCompile(`(?mi)workspace/projects/[^\s/]+/agents/`), "invalid path `workspace/projects/*/agents/` in instructions"},
}

func findInstructionViolations(content string) []string {
	violations := make([]string, 0)
	for _, check := range violationChecks {
		if check.re.FindStringIndex(content) != nil {
			violations = append(violations, check.message)
		}
	}
	return violations
}
