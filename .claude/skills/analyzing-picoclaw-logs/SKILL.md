---
name: analyzing-picoclaw-logs
description: "Use this skill to analyze logs and memory files from the running picoclaw instance (downloaded to nanobotnb/.picoclaw/). It helps identify errors, communication issues, and agent performance to suggest code or prompt improvements."
---

# Analyzing Picoclaw Logs

This skill provides a map and workflow for investigating the state of a remote `picoclaw` instance by analyzing its synchronized data in `/home/michael/projects/picoclaw/picoclaw/nanobotnb/.picoclaw/`.

## Data Map

All paths are relative to: `/home/michael/projects/picoclaw/picoclaw/nanobotnb/.picoclaw/workspace/`

### 1. Logs & Heartbeat
- **`heartbeat.log`**: Main log for the heartbeat mechanism. Check for `[ERROR]` tags and LLM failures.
- **`pm2-logs/`**: (Symlink) Contains PM2 process logs (out and err) for the running nanobot.
- **`sessions/`**: JSON files for active/recent sessions (e.g., `telegram_XXX.json`, `heartbeat.json`). Useful for seeing the raw message flow.

### 2. Memory & State
- **`memory/MEMORY.md`**: The agent's long-term memory.
- **`memory/HISTORY.md`**: Recent activity history.
- **`memory/FAILURES.md`**: Specifically tracked failures and learnings.
- **`memory/YYYY-MM-DD.md`**: Daily logs of agent activities.
- **`state/state.json`**: Current runtime state (variables, flags).

### 3. Agent Identity & Soul
- **`IDENTITY.md`**: Who the agent is.
- **`SOUL.md`**: Core directives and personality.
- **`USER.md`**: Information about the user.
- **`AGENTS.md`**: Definitions of available sub-agents.

## Workflow

### Step 1: Error Discovery
Search for errors in logs to find root causes:
```bash
grep "\[ERROR\]" /home/michael/projects/picoclaw/picoclaw/nanobotnb/.picoclaw/workspace/heartbeat.log
```

### Step 2: Context Analysis
Read the memory and failures to see what the agent *thinks* happened:
- Check `memory/FAILURES.md` for recorded issues.
- Check `memory/MEMORY.md` for current context.

### Step 2.5: Running Version Check (Required Before Regression Claims)
Before concluding "new regression", verify what build/version is actually running on the live instance and compare it against the code/commit under review.
- Record the observed runtime version/commit in your notes.
- If versions differ, treat the mismatch as first-order explanation until disproven.

### Step 3: Session Deep-Dive
If an error occurred during interaction, check the relevant session file in `sessions/` to see the exact prompts and responses.

### Step 4: Improvement Proposals
Based on findings:
1. **Code Fix**: If it's a bug in the Go/Python code, suggest changes to the local files in `/home/michael/projects/picoclaw/`.
2. **Prompt Tuning**: If it's a behavioral issue, propose updates to `SOUL.md`, `IDENTITY.md`, or specific skill prompts. **Note**: You cannot edit files in `.picoclaw/` directly as they are overwritten by the sync process. Provide the improved text to the user.

## Interaction Pacing Guardrail
During active live-debug conversation, do not launch long-running/background jobs unless the user explicitly requests that run at that moment.

## Important Note
The files in `.picoclaw/` are a **one-way sync** from the running instance. Any edits you make there will be lost. Always suggest improvements to the user so they can apply them to the running system or the source code.
