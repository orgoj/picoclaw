# Prompt: Subagent Channel Routing Guardrail

Implement these rules as hard behavior (code + AGENTS docs):

1. Subagent internal tool outputs (especially `exec`) must NEVER be sent directly to Telegram/any external channel.
2. Subagent errors/progress/completion must be delivered only to the main-agent session context (system inject/history), not to outbound channels.
3. Only the main agent may decide what to send externally.
4. External message can be sent only by:
- main-agent final response, or
- explicit `message` tool call.
5. Raw subagent tool output (`--help`, `grep`, stack traces, long stdout/stderr) must not be forwarded externally unless the main agent explicitly summarizes/quotes it.

Acceptance checks:
- Running a subagent that executes `br --help | grep ...` produces no direct Telegram outbound from that subagent.
- Main agent still receives full subagent context internally.
- User sees only main-agent-composed summaries unless explicit `message` tool is used.
