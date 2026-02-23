# Changelog

## v0.1.45 - 2026-02-23

### Added
- New user-visible config fields under `tools.spawn`:
  - `allow_llm_status` (default `true`)
  - `allow_llm_history` (default `true`)
  - `allow_llm_message` (default `true`)
  - `allow_llm_cancel` (default `false`)
- Subagent API payloads now expose cancellation provenance metadata:
  - `cancel_by` in `/api/v1/subagents` and `/api/v1/subagents/{id}` responses.

### Changed
- Main-agent tool registration now honors `tools.spawn.allow_llm_*` gates for subagent management tools (`subagent_status`, `subagent_history`, `subagent_message`, `subagent_cancel`).
- Dashboard subagent panel now shows `Cancel by` and keeps state fresh with periodic fallback refresh in addition to SSE patch updates.
- Subagent list ordering now prioritizes currently running tasks before finished tasks for clearer live operations visibility.

### Fixed
- Cancellation origin is now explicitly logged and persisted (`admin_api:dashboard_kill`, `llm_tool:subagent_cancel`, etc.), removing ambiguity when diagnosing unexpected subagent stops.

## v0.1.44 - 2026-02-23

### Added
- New user-visible config field: `agents.defaults.llm_stream_mode` (`auto`/`on`/`off`), including override support in:
  - `agents.subagents.llm_stream_mode`
  - `agents.<name>.llm_stream_mode`

### Changed
- Z.AI-compatible HTTP calls now auto-enable streaming (`stream=true`) for no-tool chat completions on `api.z.ai` GLM models, with safe fallback to non-stream JSON parsing when SSE is not returned.

### Fixed
- Session summarization LLM calls now run under a retry-aware timeout budget (`llm_timeout * attempts + llm_retry_max_elapsed_seconds + buffer`) so summary retries are not prematurely terminated by a single short outer context deadline.

## v0.1.43 - 2026-02-23

### Changed
- `picoclaw memopt --llm` now skips short `MEMORY.md` files by default before making LLM calls.
- New user-visible CLI flag: `--llm-min-chars N` (default `8000`, `0` disables the guard), with explicit skip logs (`LLM: skipped short ...`).

### Fixed
- `picoclaw memopt --llm` no longer wraps the whole LLM-retry flow in a short outer timeout, preventing retry backoff from failing early with `context deadline exceeded`.

## v0.1.42 - 2026-02-22

### Fixed
- Dashboard composer now supports explicit routing fields (`channel`, `chat_id`, `sender_id`) alongside `session_key`, enabling direct non-Telegram operation from web UI without hidden defaults.
- Session-key selection now synchronizes route fields (`channel/chat`) automatically for faster queue edit/replay workflows.

## v0.1.41 - 2026-02-22

### Fixed
- Timeline now includes outbound channel traffic as first-class `source="outbound"` events derived from message bus history, so dashboard feed can mirror what was actually sent to channels.
- Message bus now keeps bounded outbound history (`ListOutboundHistory`) with IDs/timestamps for deterministic timeline correlation.

## v0.1.40 - 2026-02-22

### Fixed
- Dashboard queue/sessions/subagents/channels/catalog tables now patch rows incrementally without full table header/body redraw on refresh, reducing jitter and preserving UI state during live updates.
- HTTP provider request shaping now recognizes GPT-5 and modern OpenAI `o*` model families for `max_completion_tokens` usage instead of `max_tokens`.
- Channel base ingress now attaches routing metadata (`peer_kind`, `peer_id`) on inbound messages by default, with caller-provided values preserved.

## v0.1.39 - 2026-02-22

### Changed
- Gateway/agent automatic prefix semantics were split into two independent config fields:
  - `gateway.sys_message_prefix` (default `[SYS]`) for program/system notices (startup/shutdown/runtime/panic paths),
  - `gateway.auto_final_prefix` (default `[AUTO]`) for final main-loop outbound replies.
- Main-loop outbound replies are now automatically prefixed with `gateway.auto_final_prefix` on non-system channels.
- Subagent task IDs are now persisted across restarts via workspace state (`workspace/state/subagent-next-id.txt`), so new tasks continue incrementing instead of restarting from `subagent-1`.
- Runtime control menu now includes `+stop` / `+continue`:
  - `+stop` pauses normal queueing for the session and routes normal messages through inject flow,
  - `+continue` restores normal queueing.

### Fixed
- Dashboard channel-visible system filtering now recognizes both `[SYS]` and `[AUTO]` system messages.
- SSE events endpoint (`/api/v1/events`) now supports read-only/no-session connections (no required `session_key`) so the dashboard can stay live even before a specific session is selected.
- SSE stream now emits incremental `patch` events between snapshots instead of forcing full snapshot payload processing on each tick.
- Added agent-scoped log streams under `logging.dir/agents`:
  - `main.jsonl` for main flow,
  - `<subagent-id>.jsonl` for subagent-specific flow.
- New Admin API endpoint: `GET /api/v1/agents/{id}/log?tail=N` for direct per-agent log retrieval.
- New Admin API endpoint: `GET /api/v1/timeline` with filter surface (`session`, `agent`, `kind`, `level`, `q`, `from`, `limit`) for unified dashboard feed consumption.
- Crash/panic and operator subagent-cancel incidents are now enqueued back to queue head as urgent incident context for immediate main-loop visibility.
- Incident queue-head context now carries explicit structured metadata fields (`cause`, `channel`, `chat_id`, `sender_id`, `session_key`, `task_id` when applicable) for deterministic downstream filtering.
- Dashboard history rendering now consumes unified timeline API data and refreshes via SSE-triggered timeline reload (single feed model for session/agent/kind/level views).
- Timeline log ingestion now enforces JSONL-only runtime records (non-JSON log lines are ignored), with formal `source/kind/level/payload` mapping from structured log fields.
- Dashboard right-panel tables (`channels`, `+ commands`, `defined agents`, `skills`) now use keyed row patching instead of full-table string rebuilds.
- Timeline API now enforces always-present event schema fields (`event_id`, `timestamp_ms`, `source`, `session_key`, `agent_id`, `kind`, `level`, `payload`) and computes stable content-based `event_id` values for runtime-log rows.
- Main-loop outbound policy now suppresses fallback final channel send when an explicit successful `message` tool send already happened in the same run (`[AUTO]` remains fallback-only).
- Successful `message` tool deliveries are now persisted into target session history (`assistant` role) so dashboard timeline mirrors channel-visible output.
- Timeline now ingests live inbound queue rows as first-class `source="queue"` events, preserving structured metadata (`cause`, `channel`, `chat_id`, `sender_id`, `task_id`, `kind`, `level`) in `payload.metadata`.
- Dashboard table rendering no longer resets table `innerHTML` on refresh for queue/sessions/subagents/channels/catalog lists; all operator panes now use keyed row patching with stale-row pruning.

## v0.1.38 - 2026-02-22

### Changed
- Runtime file logging switched to JSONL-first filenames:
  - main runtime log: `agent.jsonl` (previously `agent.log`)
  - debug runtime log: `debug.jsonl` (previously `debug.log`)
- Dashboard/admin API log-history path now reads from `agent.jsonl`.
- Session persistence switched from per-session `.json` snapshots to `.jsonl` files (`workspace/sessions/*.jsonl`), with one JSON object per line for write/read flow.

### Fixed
- Debug file output now writes structured JSONL entries (same schema family as runtime log entries) instead of plain text lines.

## v0.1.37 - 2026-02-22

### Fixed
- Dashboard JS runtime syntax error in timestamp parsing was fixed.
- Restored dashboard boot sequence for live SSE updates and drag-resize handlers (script no longer crashes during initialization).

## v0.1.36 - 2026-02-22

### Fixed
- Dashboard history rendering no longer rewrites the full list on each refresh tick.
- History pane now applies keyed incremental DOM patching by `historyKey`, updating only changed rows and preserving existing row nodes where possible.
- This reduces UI flicker/reflow and improves text-selection/copy stability during live SSE updates.

## v0.1.35 - 2026-02-22

### Added
- Admin API now exposes direct subagent cancellation via `DELETE /api/v1/subagents/{id}` (dashboard `KILL` no longer depends on control-message parsing).
- Runtime inspector payload (`/api/v1/runtime`) now includes catalog metadata:
  - `runtime.catalog.defined_agents` (all discovered/configured named agents),
  - `runtime.catalog.skills` (all discovered skills with source/path metadata).
- Session message model now stores per-message `timestamp_ms` when messages are persisted.

### Changed
- Dashboard layout is now operator-first: left flow `Messages -> Queue -> Input`, right stack `Agents -> Sessions -> Channels`.
- Dashboard Channels panel now includes:
  - enabled channel runtime status,
  - full `+` control command menu list,
  - defined named-agent catalog,
  - discovered skills catalog.
- Dashboard composer now has quick `+` command selector and ArrowUp empty-input queue recall/edit helper.

### Fixed
- Dashboard `KILL` action now calls direct API cancel endpoint (no queue/control-command roundtrip).
- Dashboard aggregated history now uses best-effort chronological ordering with priority on message `timestamp_ms` and parsed log timestamps, improving cross-source timeline coherence.

## v0.1.34 - 2026-02-22

### Fixed
- Main-agent loop no longer auto-forwards tool `ForUser` payloads to external channels during tool execution.
- Raw tool output (especially from `exec`, e.g. `--help`/grep/stdout-stderr dumps) is now kept internal for LLM/session context unless explicitly sent via the `message` tool.
- Main-agent final response remains delivered through the normal outbound path (no silent drop after tool activity).
- Automatic panic/runtime error notices sent from agent loop are now prefixed with `gateway.auto_message_prefix` (same style as startup/shutdown auto notices).

### Added
- New session rule in `AGENTS.md` enforcing strict outbound policy:
  - no automatic tool-output forwarding,
  - user-visible output is limited to explicit `message` sends and final main-agent response,
  - automatic panic/runtime errors must include configured auto prefix.

## v0.1.33 - 2026-02-22

### Fixed
- Main agent loop now applies pre-call context trimming against a conservative token estimate (safety multiplier + fixed padding) before each LLM request.
- This prevents provider-side prompt token overflow on OpenAI-compatible endpoints (including SiliconFlow) when local token approximation underestimates actual server tokenization.

### Added
- New runtime warning log when pre-call trimming is applied:
  - `Context pre-trimmed for provider safety` with before/after token estimate and message counts.

### Docs
- Added troubleshooting guidance for `max_prompt_tokens` provider errors to `README.md`.

## v0.1.32 - 2026-02-21

### Changed
- Subagent runtime now supports per-agent LLM provider selection with fallback chain:
  - `agents.<name>.provider` -> `agents.subagents.provider` -> `agents.defaults.provider`.
- Synchronous (`subagent`) and asynchronous (`spawn`) named subagent runs now resolve provider per task instead of always reusing only the main-agent provider instance.

### Added
- New user-visible config fields:
  - `agents.subagents.provider`
  - `agents.<name>.provider`

### Docs
- Updated `README.md` and `config/config.example.json` to document provider overrides for subagents and named agents.

## v0.1.31 - 2026-02-21

### Added
- New runtime CLI command: `picoclaw memopt` for whole-workspace memory optimization directly from the installed binary (no Makefile required).
- `picoclaw memopt` supports:
  - `--workspace` override (default from config workspace),
  - safe scoped processing limited to `workspace/memory/**` and `workspace/agents/*/memory/**`,
  - default `--dry-run` behavior,
  - `--apply` write mode,
  - optional `--llm` pass over `MEMORY.md` files.
- `--help` for `memopt` now works without requiring a valid config file.

## v0.1.30 - 2026-02-21

### Fixed
- Active-run urgent inject no longer cancels/preempts the current session run; urgent content is now appended into the active context and handled via a follow-up LLM iteration in the same run.
- Inject confirmation text now reflects in-context active-run injection (no preemption wording) across channel controls and Telegram `/inject`.
- Inject payload content is now neutral by default: runtime wraps with `<inject_message ...>` and no longer auto-appends urgency instructions.
- Subagent completion for sessions without an active run now triggers immediate same-session LLM processing (no wait for next user message), while still persisting completion into session history.
- Subagent completion immediate trigger payload now carries explicit identity metadata in `<inject_message ...>` attributes: `sender_id`, `agent_id`, and `label`, so console/history views can attribute which subagent finished.
- Subagent toolloop context window now honors `agents.defaults.context_window` instead of deriving window from `max_tokens`, so runtime context telemetry and trimming align with configured default window.
- Dashboard message textarea (`Message` panel) now has responsive max-height bounds so it no longer grows past mobile viewport edges.
- Dashboard runtime inspector now renders actual new lines in summary/config panes (no literal `\n` text artifacts).
- Dashboard all-history mode now includes tailed runtime console lines from `logging.dir/agent.log`, so history view reflects operator-visible console flow beyond per-session chat history.
- Dashboard default all-history aggregation now maximizes coverage with bounded caps: session list fetch (`1000`), `agent.log` tail (`5000`), and render limit (`3000`) for responsiveness.
- Dashboard filter/selection state is now visibly explicit: history header shows active scope pills (`ALL`, `SESSION`, `AGENT`) and selected rows in Sessions/Agents tables use stronger highlight cues.
- LLM transient retry detection now matches timeout errors case-insensitively and includes `context deadline exceeded` / `Client.Timeout exceeded` forms, so temporary provider stalls are retried instead of failing on first timeout.
- Subagent config resolution now consistently falls back `named > subagents > defaults` for loop/model knobs, so sparse subagent blocks inherit `agents.defaults.*` instead of zero-value behavior.
- Session summarization history retention is now configurable via `agents.defaults.summary_keep_last_messages` (replacing hardcoded keep-last-4 behavior).
- Debug `LLM iteration` logs now include compact absolute context state (`context_state`, `context_tokens`, `context_window`, `context_remaining`) for runtime tuning without extra per-iteration log spam.
- Subagent config now applies `agents.subagents.memory_threshold` and `agents.subagents.summary_keep_last_messages` (with named-agent overrides), eliminating previously ignored fields.
- Subagent toolloop debug logs now emit the same compact absolute context state fields as main loop.

### Added
- New Admin API endpoint: `/api/v1/history/agent-log?tail=N` returning recent `agent.log` lines for dashboard all-history aggregation.

## v0.1.29 - 2026-02-21

### Changed
- Dashboard history view is now one-line operator style with click-to-expand full message detail instead of a long wrapped text block.
- Dashboard message composer now uses dedicated mode buttons (`Queue`, `Inject`, `Force First`, `Append`, `Delete Last`) instead of a mode select dropdown.
- Dashboard filter semantics are now explicit:
  - top-bar `Clear all filters` resets both session and subagent filters
  - selecting a subagent clears any active session filter
  - clear state defaults to aggregated all-session history
- Dashboard mobile UX now uses panel tabs (`Agents`, `Sessions`, `Queue`, `Message`) so only one right-side panel is visible at once.

### Fixed
- Subagent ordering in dashboard/API responses is now stable and deterministic (newest started first with tie-breakers), preventing random row flicker/reordering between refreshes.
- Dashboard all-history scope now refreshes across known sessions, rather than silently sticking to a single previously selected session after filter clear.

### Added
- New dashboard user-visible controls/fields:
  - top-bar action: `Clear all filters`
  - history detail pane showing `Session`, `Index`, `Role`, and detected `Agent`
  - color cues for history rows by role/tone plus per-agent name color hint

### Fixed
- Subagent completion events are now guaranteed to reach main-agent session context in all cases: completion is always persisted as `system` history and also queued as urgent context for the next main-agent run when no run is active.
- Debug-level runtime diagnostics are now mirrored to `debug.log` (in `logging.dir`) in addition to console output, while structured JSON logs continue in `agent.log`.

## v0.1.28 - 2026-02-21

### Fixed
- `exec` safety guard now enforces `agents.defaults.deny_path_patterns` in addition to existing dangerous-command/path-traversal checks.
- Denied-path matching for `exec` now covers both absolute and relative path arguments (including `cd ... &&` command flows), so sensitive targets like `.beads/*.jsonl` and `.beads/*.db` are blocked even when `restrict_to_workspace` is `false`.
- `cron` scheduled commands now run through an `exec` instance configured with the same runtime safety settings (`restrict_to_workspace` and `deny_path_patterns`), removing policy drift versus interactive `exec`.

### Added
- New `exec` guard tests for deny-pattern blocking:
  - `.beads/issues.jsonl`
  - `.beads/beads.db`
- New `cron` regression test ensuring scheduled command execution respects deny-pattern blocking for `.beads/issues.jsonl`.

## v0.1.27 - 2026-02-20

### Changed
- Unified LLM retry/backoff behavior into a shared helper used by:
  - main agent run loop,
  - subagent tool loop,
  - conversation summarization/summary-merge calls.
- Subagent LLM requests now use the same transient error retry policy as the main loop instead of failing on first timeout/EOF.

### Added
- New internal module `pkg/llm/retry.go` for centralized retry logic with:
  - retryable network/5xx detection,
  - rate-limit backoff handling,
  - max elapsed retry window clamping,
  - consistent retry observability fields in logs.

## v0.1.26 - 2026-02-20

### Fixed
- Exec safety guard no longer misclassifies slash-containing free-text arguments as out-of-workspace paths (for `--description`, `--body`, `--message`, `--title`, and `-m`).
- Workspace path restriction for `exec` still blocks actual absolute filesystem path arguments outside the configured working directory.

### Added
- Explicit user-visible exec guard metadata for supported free-text flags:
  - `--description`
  - `--body`
  - `--message`
  - `--title`
  - `-m`

## v0.1.25 - 2026-02-20

### Changed
- Dashboard history auto-scroll now respects operator reading position: live updates only stick to bottom when user is already near the bottom.
- Dashboard subagent filter no longer silently falls back to full history when no match exists; the history pane now shows explicit empty-state text for selected-agent scope.
- Dashboard `session_key` input is no longer hardcoded to a fixed Telegram value.
- Dashboard outbound status feedback now uses concise operator-facing result text instead of raw JSON blobs.

### Added
- Destructive action confirmations in dashboard controls:
  - message mode `Delete Last`
  - queue row `del`
  - subagent `KILL`
- Request busy-state handling on dashboard action buttons to reduce accidental duplicate operations (send/save/move/delete/kill).

## v0.1.24 - 2026-02-20

### Changed
- Dashboard message composer now maps directly to channel-agnostic prefix controls instead of a generic urgent toggle.
- Dashboard subagent table restores per-row `KILL` action when runtime controls include `<prefix>kill TASK_ID`.
- Dashboard inbound queue table now exposes explicit reorder actions (`top`, `up`, `down`, `bottom`) alongside save/delete.
- Dashboard subagent selection and history filtering behavior was tightened:
  - active subagent row uses persistent selected-row highlight
  - history filter matches subagent markers more broadly (`subagent:*` patterns)
  - if no explicit marker exists, history falls back to full session stream so live updates remain visible
  - agents panel now includes explicit `Clear filter` action to reset subagent history filter in one click

### Added
- New user-visible dashboard message fields:
  - `Mode` selector: `Queue`, `Inject`, `Force First`, `Append`, `Delete Last`
  - `commandHint` helper text showing the exact resolved control command with configured prefix

## v0.1.23 - 2026-02-20

### Changed
- Runtime loop limits are now explicit and separated for both main agent and subagents:
  - `max_iterations` = max total LLM loop iterations
  - `max_tool_iterations` = max iterations that include tool calls
- Subagent resolver no longer aliases one limit to the other; both values are resolved independently from `agents.subagents` and optional `agents.<name>` overrides.

### Docs
- Updated `config/config.example.json` and README config references to show both loop limits for `agents.defaults`, `agents.subagents`, and named agents.

## v0.1.22 - 2026-02-20

### Changed
- Subagent runtime config branch is now strict and standalone: runtime no longer uses `agents.defaults.max_tokens_subagent` fallback.
- Subagent context sizing now derives from `agents.subagents.max_tokens` (or named override via `agents.<name>.max_tokens`).
- Added per-named-agent concurrency override: `agents.<name>.max_concurrent_subagents` on top of global `agents.defaults.max_concurrent_subagents`.

### Docs
- Updated `config/config.example.json` and README to remove deprecated `*_subagent` runtime keys and document per-named-agent concurrency control.

## v0.1.21 - 2026-02-20

### Changed
- Subagent iteration limit resolution is now deterministic: `agents.subagents.max_iterations` is the baseline, with optional `agents.<name>.max_iterations` override.
- Removed runtime use of `agents.defaults.max_iterations_subagent` for subagent loop limits.
- Subagent model selection is now configurable via `agents.subagents.model` with optional per-named-agent override `agents.<name>.model`.

### Docs
- Updated `config/config.example.json` and README config references to reflect current subagent schema and model/iteration controls.

## v0.1.20 - 2026-02-20

### Fixed
- Dashboard SSE stream (`/api/v1/events`) no longer gets cut by the gateway HTTP write timeout every few seconds.
- Gateway HTTP server timeout profile updated for long-lived streams:
  - `WriteTimeout` disabled for SSE compatibility.
  - `ReadHeaderTimeout` set to `5s`.
  - `ReadTimeout` set to `10s`.
  - `IdleTimeout` set to `120s`.

## v0.1.19 - 2026-02-20

### Changed
- Gateway automatic notices are now broadcast to all known external sessions (`channel:chat`) discovered from session history, not only the last active session.
- Automatic gateway notices now include startup online notice, shutdown notice, and runtime startup/health error notices.
- All automatic gateway notices now include a configurable visible prefix.

### Added
- New config field: `gateway.auto_message_prefix` (default: `[AUTO]`).
- New user-visible notice metadata/fields in templates and payload context:
  - `{{timestamp}}`
  - `{{channel}}`
  - `{{chat_id}}`
  - `{{signal}}` (shutdown notice)
  - `{{error}}` (runtime error notice)

## v0.1.18 - 2026-02-20

### Changed
- IDLE trigger now enters the same main session flow as a normal message (same `channel/chat/session`), instead of running in an isolated idle-only context.
- IDLE payload is XML-wrapped (`<idle_message>`, `<idle_context>`, `<idle_protocol>`, optional `<subagent_status>`) so the main agent has explicit idle-mode context in the conversation history.

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
