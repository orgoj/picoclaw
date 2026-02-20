# Changelog

## v0.1.3 - 2026-02-20

### Added
- Telegram command `/inject` for immediate high-priority processing (active-run inject, otherwise queue bypass).
- Telegram command `/first` to enqueue a message at the head of inbound queue.
- Ingress controls in config:
  - `ingress.merge_window_seconds`
  - `ingress.gap_notice_seconds`
  - `ingress.concat_prefix`
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
- Agent loop injects timing context note into user input when relevant metadata is present.

### Compatibility Notes
- `logging.file_path` is removed; use `logging.dir`.
- Telegram `/urgent` is retained as a backward-compatible alias of `/inject`.
