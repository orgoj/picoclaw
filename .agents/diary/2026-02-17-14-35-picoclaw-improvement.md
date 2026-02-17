# Session Diary

**Date**: 2026-02-17 14:35
**Agent**: Pi-Agent (Antigravity)
**Project**: /home/michael/projects/picoclaw

## Task Summary
Improve PicoClaw stability, subagent management, and shell security for the `nanobotnb` instance.

## Work Done
- **Configurable Logging**: Added `Logging` section to `pkg/config/config.go` and implemented `setupLogging` helper in `cmd/picoclaw/main.go` to persist logs to `~/.picoclaw/logs/agent.log`.
- **Subagent Management**: Registered missing tools (`subagent_status`, `subagent_history`, `subagent_message`, `subagent_cancel`) in `pkg/agent/loop.go` to give supervisors visibility and control over spawned tasks.
- **Shell Security Enhancement**: Modified `pkg/tools/shell.go` to allow absolute paths within workspace, added an allowlist for common `/dev/` devices (`null`, `zero`, `random`, `stdin`, etc.), and added logic to skip URL patterns.
- **Error Handling**: Improved error checking for home directory expansion and directory creation in the main entry points.
- **Testing & Quality**: Fixed `pkg/tools/web_test.go` to match new tool signatures and ensured all tests pass with `make test`.

## Mistakes & Corrections (CRITICAL)
### Where I Made Errors:
- **Redundant Documentation**: I mistakenly modified `workspace/AGENT.md` which is a bootstrap template. The user corrected me that active agents won't see changes there and it should remain clean.
- **Missing Imports**: Initially forgot to import `github.com/sipeed/picoclaw/pkg/logger` in `shell.go` when adding log warnings, which broke the build.
- **Regex Edge Case**: The initial `rm` deny pattern was too broad, then too narrow (missing a space for the test case).

### What Caused the Mistakes:
- **Assumption**: Assumed `AGENT.md` was part of the active agent's immediate context rather than a one-time template.
- **Test-Driven Refinement**: Security regexes are notoriously brittle and required several iterations against the project's own test suite.

## Lessons Learned
### Technical:
- **Dynamic Context**: PicoClaw generates its tool descriptions dynamically from the code, making manual documentation in bootstrap files redundant for existing sessions.
- **Shell Filtering**: When restricting shell access to a workspace, it's safer to resolve absolute paths via `filepath.Abs` and check prefixes rather than using simple string or regex blocks.

### Process:
- **Verify Bootstrap vs. Runtime**: Always distinguish between project templates (`workspace/`) and the agent's actual runtime memory/history.
- **Run `make vet` earlier**: Build-breaking missing imports would have been caught sooner.

## Skills Used
| Skill | Issue/Observation | Action |
|-------|-------------------|--------|
| commit | Finalizing changes | Used structured commit messages for feat/docs/fix categories. |
| planning | Task complexity | Created PLAN-061b286f to track multi-file modifications. |
| agents-diary | Session end | Documenting session for cross-agent continuity. |
