---
name: agents-evolution
description: "High-level learning loop for improving agents. Analyzes logs, memory (FAILURES.md), and sessions to identify root causes and harden the system through code or prompt improvements."
---

# Agents Evolution

This skill manages the iterative process of improving `picoclaw` agents by analyzing their performance and failures in the remote environment.

## Workflow

### 0. Log Rotation & Versioning (CRITICAL)
Before starting analysis, ensure you are looking at the correct log:
- **Automatic Rotation**: PicoClaw now rotates `agent.log` upon restart, renaming the old log to `agent.log.YYYYMMDD-HHMMSS.old`.
- **Version Markers**: Each new log session starts with a `PicoClaw started` INFO message containing the version and git commit.
- **Git Context**: Cross-reference the git commit in the log with `git log` to see which fixes were already applied.

### 1. Identify Issues
Run the analysis script on the **current** log:
```bash
./scripts/analyze-agent-failures.sh
```

### 2. Root Cause Analysis
- **Exclude Fixed Issues**: Ignore errors that occurred before the latest "PicoClaw started" marker with the current version.
- **Timeouts**: Check if the agent is trying to do too much in one `exec` call or if network latency is high.
- **Safety Guards**: If `exec` is blocked, evaluate if the command was actually dangerous or if the regex in `pkg/tools/shell.go` is too strict.
- **ZAI Failures**: "Resource not found" usually means the URL or prompt for the web tool was malformed.

### 3. Implement Improvements
- **Code Fixes**: Modify Go files in `pkg/` (e.g., `pkg/tools/shell.go` to adjust safety rules).
- **Prompt Hardening**: Update `SOUL.md` or `IDENTITY.md` with instructions to avoid known pitfalls (e.g., "Always sort files before using `comm`").
- **Skill Evolution**: Create or update skills in the agent's workspace to provide better deterministic tools.

## Resources
- **`scripts/analyze-agent-failures.sh`**: Summarizes errors from logs and memory.
- **`../analyzing-picoclaw-logs/SKILL.md`**: Map of all synchronized data.

## Note
You cannot edit files in `.picoclaw/` directly. All behavioral improvements (prompts) must be presented to the user to be applied to the running agent.
