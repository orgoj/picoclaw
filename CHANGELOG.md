# Changelog

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
