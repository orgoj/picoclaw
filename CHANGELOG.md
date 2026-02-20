# Changelog

## v0.1.17 - 2026-02-20

### Changed
- Restored full LLM request observability in JSON audit stream via new `llm_request_full` events containing complete `messages_json` and `tools_json`.
- Kept runtime console/file logs concise (no reintroduction of verbose full payload dump lines there).

## v0.1.16 - 2026-02-20

### Fixed
- Subagent completion handling from `system` channel now delivers completion metadata to the origin session context instead of returning a response on outbound `system` channel.
- New completion-path audit metadata:
  - `subagent_completion_injected_active_run` (completion injected into active origin run)
  - `subagent_completion_buffered_session` (completion buffered in origin session history when no active run)

## v0.1.15 - 2026-02-20

### Changed
- Urgent inject behavior for active sessions now preempts the current run cycle (context cancel) instead of waiting for a later iteration.
- Added explicit audit metadata for preemption path: `agent_run_preempted_inject`.
- Updated inject confirmation wording on channel controls and Telegram `/inject` to indicate preemption of active cycle.
- Reduced debug log verbosity by removing full request payload dumps (`messages_json`, `tools_json`) and system-prompt preview spam from normal debug flow.

## v0.1.14 - 2026-02-20

### Added
- New sessions API endpoint: `/api/v1/sessions` (recent sessions with message count and timestamps).

### Changed
- Dashboard layout redesigned for ops flow:
  - left full-height history pane
  - right stacked panels: agents, sessions, queue, message
  - draggable splitters to resize left/right columns and right-column row heights
- History panel now supports click filters:
  - click session to switch stream/history source
  - click subagent to filter history messages by that subagent context
- History rendering now shows last 1000 messages (scrollable) with visible "showing X / Y" metadata.
- Subagent rows now include visual status cues (running vs. stopped/finished background state).
- Dashboard splitter positions are persisted in browser local storage.
- Gateway startup endpoint banner and README include `/api/v1/sessions`.
- Production binary size check (linux/amd64): `28,523,335` bytes.
- Upstream baseline (`9d5728e`) size: `26,375,085` bytes.
- Delta vs upstream baseline: `+2,148,250` bytes (`+8.15%`).

## v0.1.13 - 2026-02-20

### Added
- New runtime inspection API endpoint: `/api/v1/runtime`.
- Dashboard "Runtime info" popup with operator summary and sanitized config view.

### Changed
- Gateway startup endpoint banner now includes `/api/v1/runtime`.
- Dashboard now fetches runtime metadata continuously from API (version, tools/skills/agents counts, channel status, control commands list/count, queue/subagent counters).
- Config shown in dashboard runtime inspector is secret-sanitized (tokens/API keys/secrets/password fields masked).

## v0.1.12 - 2026-02-20

### Added
- Configurable runtime log level via `logging.level` (`debug|info|warn|error`).

### Changed
- CLI `--debug`/`-d` remains supported and now explicitly overrides `logging.level`.
- Updated docs/examples to include `logging.level`.

## v0.1.11 - 2026-02-20

### Added
- Debug-level queue observability in `bus` component for inbound/outbound operations (`enqueue/dequeue/merge/delete/wait`).
- Console flow marker for subagent logs: `[SUBAGENT:<id>]`.
- Subagent terminal/system messages now include explicit `subagent_id` metadata and `Subagent ID` header in content.

## v0.1.10 - 2026-02-20

### Fixed
- Hardened gateway signal handling to prevent stuck shutdowns:
  - First `Ctrl+C` starts graceful shutdown.
  - Second `Ctrl+C` forces immediate process exit.
  - Graceful shutdown now has a bounded timeout before forced exit.

## v0.1.9 - 2026-02-20

### Fixed
- Completed channel-agnostic control interception so control commands are handled by ingress instead of leaking to main agent text flow.
- Restored rich runtime detail for `<prefix>status` including running subagents, recent subagents, queue context, and session context.

### Added
- New channel-agnostic controls:
  - `<prefix>help`
  - `<prefix>models`
  - `<prefix>channels`
  - `<prefix>delete` (deletes last queued message in same session/sender)
- Updated Telegram `/help` and README command list to include all prefix controls.

## v0.1.8 - 2026-02-20

### Added
- New agent defaults field `deny_path_patterns` for hard-blocking file and directory access with glob patterns.

### Changed
- File tools now enforce deny patterns across `read_file`, `write_file`, `list_dir`, `edit_file`, and `append_file`.
- Deny patterns are enforced even when `restrict_to_workspace` is set to `false` (unsafe mode).
- Documented user-visible deny-pattern metadata:
  - Config field: `agents.defaults.deny_path_patterns`
  - Glob semantics: `*`, `**`, `?`

## v0.1.7 - 2026-02-20

### Fixed
- Added channel-agnostic `<prefix>status` control command with immediate runtime status reply.
- Updated append control semantics to support `<prefix><prefix>MESSAGE` for appending to the previous queued message in the same session, with explicit confirmation reply.
- Prevented append escape from silently enqueuing unintended standalone messages.

## v0.1.6 - 2026-02-20

### Changed
- Priority controls moved from Telegram-specific slash flow to channel-agnostic control parsing in core channel ingress:
  - `<prefix>inject` for immediate urgent handling
  - `<prefix>first` for inbound queue head insertion
  - `<prefix>kill` for subagent cancellation
- Control prefix is configurable via `ingress.concat_prefix` (first rune is used).
- Escape sequence `<prefix>+` now appends a literal `+` to the previous queued message in the same session (no control parsing).
- Telegram slash command menu for control operations is cleared; `/help` now points to universal prefix controls.

## v0.1.5 - 2026-02-20

### Fixed
- Gateway shutdown (`Ctrl+C`) now sends shutdown notice directly to the active channel before cancellation, instead of relying on outbound queue dispatch.
- Channel manager no longer reports expected `context canceled`/timeout sends during shutdown as hard errors.
- Telegram outbound send path now exits cleanly on cancellation (no misleading HTML-parse fallback log during shutdown), and Telegram handler is explicitly stopped in channel stop flow.

## v0.1.4 - 2026-02-20

### Added
- Reflection hardening rules in `AGENTS.md` from diary consolidation:
  - confirm implementation direction before edits when requirements are ambiguous
  - use native `apply_patch` tool directly (avoid shell-wrapped patch flows)
  - explicitly list user-visible metadata/fields in changelog behavior notes
  - run `make vet` early after structural/API edits
  - keep documentation updates only in explicitly requested locations

## v0.1.3 - 2026-02-20

### Added
- Telegram command `/inject` for immediate high-priority processing (active-run inject, otherwise queue bypass).
- Telegram command `/first` to enqueue a message at the head of inbound queue.
- Ingress controls in config:
  - `ingress.merge_window_seconds`
  - `ingress.gap_notice_seconds`
  - `ingress.concat_prefix`
- Time-aware inbound metadata on queued messages:
  - `received_at`
  - `enqueued_at`
  - `delta_since_prev_ms`
  - `queue_len_at_enqueue`
  - `gap_notice` (when silence gap exceeds configured threshold)
- Audit event log file (`audit.jsonl`) under configured logging directory.
- `VERSION` file as canonical build version source for `make`.

### Changed
- Logging config switched from `logging.file_path` to `logging.dir`.
- Runtime log outputs are unified under `logging.dir`:
  - `agent.log`
  - `heartbeat.log`
  - `audit.jsonl`
- Heartbeat logging path now follows configured logging directory.
- Inbound queue now enriches message metadata with timing/queue context.
- Inbound queue supports message merge behavior (same session/sender, time window, or forced prefix).
- Agent loop injects `<timing_context>` into the user input when relevant metadata is present.
- Idle processing now tracks and injects idle runtime context (`idle_streak_count`, `idle_since`, `last_user_message_at`, `seconds_since_user_message`).

### Compatibility Notes
- `logging.file_path` is removed; use `logging.dir`.
- Telegram `/urgent` is retained as a backward-compatible alias of `/inject`.
