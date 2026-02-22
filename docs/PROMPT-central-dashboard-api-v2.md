# Prompt for Main Agent: Implement API v2 for Central Dashboard

Implement a backward-compatible `API v2` for PicoClaw central dashboard support.

## Primary Objective

Build `/api/v2/*` endpoints as a stable contract for:
- local dashboard use
- future central dashboard aggregating multiple PicoClaw instances

Keep `/api/v1/*` fully working.

## Mandatory Constraints

- Do not break existing `/api/v1/*` handlers or current dashboard behavior.
- Reuse existing runtime/queue/subagent logic; avoid business-logic duplication.
- Keep token/network usage low via server-side filtering and compact payloads.
- Keep external channel safety: no raw tool `exec` output auto-forwarding.

## Source of Truth

Use this contract:
- `docs/API-v2-central-dashboard.md`

If implementation detail conflicts with current runtime reality, adapt carefully and document the delta in changelog.

## Deliverables

1. New handlers and routes under `/api/v2/*`:
   - `GET /api/v2/runtime`
   - `GET /api/v2/messages`
   - `GET /api/v2/queue`
   - `PATCH /api/v2/queue/{id}`
   - `DELETE /api/v2/queue/{id}`
   - `POST /api/v2/queue/{id}/move`
   - `POST /api/v2/messages`
   - `GET /api/v2/agents`
   - `DELETE /api/v2/subagents/{id}`
   - `GET /api/v2/sessions`
   - `GET /api/v2/logs`
   - `GET /api/v2/stream` (SSE)

2. Shared envelope DTO for feed/event rows:
   - `id`, `seq`, `ts`, `instance_id`, `session_key`, `channel`, `agent_id`, `agent_kind`, `level`, `kind`, `meta`, `payload`

3. Server-side filters for list/stream endpoints:
   - `scope`, `agent_id`, `session_key`, `channel`, `level`, `kind`, `q`, `since_ts`, `until_ts`, `limit`, `cursor`, and `from_seq` for SSE

4. Direct subagent kill API:
   - `DELETE /api/v2/subagents/{id}` must call manager cancel directly (not queue message parsing)

5. Instance identity:
   - add `gateway.instance_id` (with sensible fallback if missing)
   - include it in runtime + feed envelopes

6. Documentation and release hygiene:
   - update `README.md` (new API section)
   - update `CHANGELOG.md` with user-visible fields/endpoints
   - bump version
   - keep `config/config.example.json` aligned with any new config fields

7. Tests:
   - add/extend `pkg/adminapi` tests for new endpoints and filters
   - cover SSE basic behavior and direct subagent cancel endpoint

## UX Integration Follow-up (same branch if feasible)

- Switch dashboard JS to consume `/api/v2/*`.
- Preserve current layout direction:
  - left column: messages -> queue -> input composer
  - right column: agents/sessions/channels
  - logs in separate tab

## Acceptance Checklist

- `make vet` passes
- `make test` passes
- `curl` smoke checks for all `/api/v2/*` routes
- old `/api/v1/*` routes still respond
- dashboard loads and remains operational

