# PicoClaw API v2 for Central Multi-Agent Dashboard

Status: proposal (backward-compatible with existing `/api/v1/*`)

## Goals

- Keep `/api/v1/*` unchanged for current local dashboard.
- Add `/api/v2/*` as stable, multi-node, central-dashboard API.
- Standardize message/event schema across channels and agents.
- Make filtering server-side (fast UI, lower token and network cost).

## Design Principles

- API-first, channel-agnostic, agent-agnostic.
- Event stream with monotonic `seq` for replay/resume.
- Explicit scope separation:
  - Operator feed (`messages/events`)
  - Queue controls
  - Runtime/health metadata
  - Logs (separate from chat feed)
- Do not expose tool `exec` raw output to external channels by default.

## Base

- Prefix: `/api/v2`
- Content-Type: `application/json`
- Time format: Unix milliseconds in `ts`
- Pagination:
  - Pull endpoints: `limit`, `cursor` (opaque)
  - Stream endpoints: `from_seq`

## Common Envelope

All feed rows and stream events should use the same envelope:

```json
{
  "id": "evt_01J...",
  "seq": 102944,
  "ts": 1766403123456,
  "instance_id": "node-prg-01",
  "session_key": "telegram:12345",
  "channel": "telegram",
  "agent_id": "main",
  "agent_kind": "main",
  "level": "info",
  "kind": "message",
  "meta": {
    "source": "main_loop"
  },
  "payload": {
    "role": "assistant",
    "content": "..."
  }
}
```

## Enumerations

- `agent_kind`: `main | subagent | system`
- `level`: `debug | info | warn | error`
- `kind`:
  - `message`
  - `queue_item`
  - `queue_update`
  - `agent_status`
  - `runtime_notice`
  - `system_notice`
  - `log_line`

## Filtering Contract (shared query params)

Use on list endpoints (`/messages`, `/queue`, `/logs` where applicable):

- `instance_id` (optional for local node, required in central aggregator)
- `scope`: `main | subagent | all` (default `all`)
- `agent_id`
- `session_key`
- `channel`
- `level`
- `kind`
- `q` (case-insensitive substring)
- `since_ts`
- `until_ts`
- `limit` (default 200, max 2000)
- `cursor` (opaque)

## Endpoints

### 1) `GET /api/v2/runtime`

Purpose: compact runtime card for operators and central collectors.

Response:

```json
{
  "runtime": {
    "instance_id": "node-prg-01",
    "version": { "app": "v0.1.34", "go": "go1.24.0" },
    "agent": {
      "running": true,
      "uptime_sec": 812,
      "main_context_est_tokens": 22400,
      "main_messages_total": 514,
      "main_tool_calls_total": 129
    },
    "subagents": {
      "running_count": 2,
      "recent_count": 11,
      "queue_count": 1
    },
    "inbound_queue": { "count": 4 },
    "channels": {
      "enabled_count": 2,
      "enabled": ["telegram", "web"],
      "status": {}
    },
    "controls": {
      "prefix": "+",
      "commands": ["+status", "+inject MESSAGE", "+kill TASK_ID"]
    }
  }
}
```

### 2) `GET /api/v2/messages`

Purpose: unified operator feed (no raw debug dump by default).

Response:

```json
{
  "items": [],
  "next_cursor": "opaque",
  "total_hint": 1203
}
```

### 3) `GET /api/v2/queue`

Purpose: queue inspection for UI (`messages -> queue -> input` flow).

Response:

```json
{
  "items": [
    {
      "id": "in_123",
      "seq": 102955,
      "ts": 1766403125000,
      "session_key": "telegram:12345",
      "channel": "telegram",
      "sender_id": "user",
      "content": "hello",
      "meta": {}
    }
  ]
}
```

### 4) `PATCH /api/v2/queue/{id}`

Purpose: edit queued item content.

Body:

```json
{ "content": "new content" }
```

### 5) `DELETE /api/v2/queue/{id}`

Purpose: delete queued item.

### 6) `POST /api/v2/queue/{id}/move`

Purpose: reorder queued item.

Body:

```json
{ "index": 0 }
```

### 7) `POST /api/v2/messages`

Purpose: send or inject message to main loop.

Body:

```json
{
  "session_key": "web:dashboard",
  "channel": "web",
  "chat_id": "dashboard",
  "sender_id": "web",
  "mode": "queue",
  "content": "run check"
}
```

`mode` enum:
- `queue` -> normal enqueue
- `inject` -> urgent inject (active run)
- `first` -> enqueue at queue head
- `append` -> append to previous queued message
- `delete_last` -> delete last queued message in session

### 8) `GET /api/v2/agents`

Purpose: main/subagent status list for right panel.

Response:

```json
{
  "items": [
    {
      "id": "main",
      "kind": "main",
      "status": "running",
      "started_ts": 1766403000000,
      "ended_ts": 0,
      "message_count": 514,
      "tool_call_count": 129,
      "context_est_tokens": 22400,
      "pending": 0
    },
    {
      "id": "sub_abc",
      "name": "ui-designer",
      "kind": "subagent",
      "status": "running",
      "started_ts": 1766403100000,
      "ended_ts": 0,
      "message_count": 17,
      "tool_call_count": 12,
      "context_est_tokens": 4100,
      "pending": 0
    }
  ]
}
```

### 9) `DELETE /api/v2/subagents/{id}`

Purpose: direct kill endpoint (do not route through queue message).

Response:

```json
{ "ok": true, "id": "sub_abc", "status": "canceled" }
```

### 10) `GET /api/v2/sessions`

Purpose: session list with latest activity metadata.

### 11) `GET /api/v2/logs`

Purpose: operator logs separate from user-visible message feed.

Query params:
- `source`: `agent|debug|audit`
- `tail`: default 500, max 5000
- plus shared filters where relevant

### 12) `GET /api/v2/stream`

Purpose: SSE stream with the same envelope as `/messages`.

Query params:
- `from_seq`
- shared filters

Events:
- `event: snapshot` (initial state)
- `event: upsert` (new or updated row)
- `event: delete` (queue/message removal)
- `event: heartbeat` (keepalive)

## Error Model

Use consistent error body:

```json
{
  "error": {
    "code": "invalid_argument",
    "message": "content is required",
    "details": {}
  }
}
```

Recommended `code` values:
- `invalid_argument`
- `not_found`
- `conflict`
- `unavailable`
- `internal`

## Compatibility Plan

1. Keep existing `/api/v1/*` unchanged.
2. Add `/api/v2/*` in parallel (shared internal services, no duplicate business logic).
3. Move Web UI to consume `/api/v2/*`.
4. Add central-collector service that reads `/api/v2/stream` from multiple instances.
5. After stable migration window, mark `/api/v1/*` as deprecated (still available).

## Implementation Notes (for this repo)

- Reuse current handlers in `pkg/adminapi/inbound.go` through adapter layer.
- Introduce shared DTO mappers to avoid copy-paste response logic.
- Assign per-instance stable `instance_id`:
  - config key proposal: `gateway.instance_id`
  - fallback: hostname + port
- Persist `seq` in-process monotonic counter (for single node); central collector remaps into global sequence if needed.
- Keep default log safety:
  - no automatic forwarding of tool `exec` output to external channels
  - dashboard log tab can show debug, but operator can hide by default

