---
name: agent-builder
description: Create and manage named sub-agents in picoclaw. Use when creating a new named agent (workspace/agents/<name>/), writing its AGENTS.md identity, or setting up its memory structure. Named agents have persistent identity and memory across tasks.
---

# Agent Builder

## Directory Structure

```
workspace/agents/<name>/
  AGENTS.md          ← identity + YAML frontmatter (required for discovery)
  memory/
    MEMORY.md        ← long-term memory (agent writes here after tasks)
    YYYYMM/
      YYYYMMDD.md    ← daily notes
```

## AGENTS.md Format

```markdown
---
name: <name>
description: <one-line description for the main agent to know when to use this agent>
---

## Identity

You are <name>, a <role> specializing in <domain>.

## Core Skills

- <skill 1>
- <skill 2>

## Working Style

<How the agent approaches problems, what quality standards it holds>

## Memory

After each task, save key learnings:
- Long-term insights → agents/<name>/memory/MEMORY.md
- Daily notes → agents/<name>/memory/YYYYMM/YYYYMMDD.md
```

## Creating a Named Agent

1. Create directory: `workspace/agents/<name>/`
2. Write `AGENTS.md` with YAML frontmatter (`name` + `description` required)
3. Optionally seed `memory/MEMORY.md` with initial knowledge

Use `write_file` — it creates parent directories automatically.

## Naming Rules

- Lowercase letters, digits, hyphens only: `[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*`
- No dots, slashes, or spaces (security — path traversal prevention)
- Examples: `coder`, `reviewer`, `data-analyst`, `go-expert`

## Using a Named Agent

```
subagent(task="...", name="coder", label="refactor")
spawn(task="...", name="reviewer", label="code-review", directory="projects/myapp")
```

The agent's identity and memory are automatically injected into its system prompt.

## When to Create a New Named Agent

- Recurring specialized role (code reviewer, researcher, writer)
- Agent that benefits from accumulated experience (grows smarter over time)
- Domain specialist with specific working style

## When NOT to Use Named Agents

- One-off tasks with no recurring pattern
- Tasks where fresh context is better than accumulated memory
- Simple delegations — use anonymous `subagent`/`spawn` instead
