# Codebase Map: PicoClaw
> Ultra-efficient AI Assistant in Go | Tech: Go, LLM, MCP | Generated: 2026-02-17

## Architecture
PicoClaw is an ultra-lightweight personal AI assistant designed to run on low-resource hardware ($10 SBCs, <10MB RAM). It follows a modular architecture where a central **AgentLoop** manages conversation state, interacts with various **LLM Providers** (Claude, Gemini, Zhipu, etc.), and executes **Tools** (filesystem, shell, hardware, web search). The system supports multi-channel communication (Telegram, Discord, etc.) via a **Message Bus** and features a background **Heartbeat** for periodic tasks.

## Module Index
| Directory | Purpose | Key Files | Depends On |
|-----------|---------|-----------|------------|
| `cmd/picoclaw/` | CLI Entry Point | `main.go` | `pkg/` |
| `pkg/agent/` | Core Agent Logic | `loop.go`, `context.go`, `memory.go` | `pkg/providers/`, `pkg/tools/`, `pkg/session/` |
| `pkg/channels/` | Communication Channels | `manager.go`, `telegram.go`, `discord.go` | `pkg/bus/` |
| `pkg/tools/` | Tool Implementations | `registry.go`, `filesystem.go`, `shell.go`, `spawn.go` | `pkg/mcp/` |
| `pkg/providers/` | LLM Provider Integration | `claude_provider.go`, `gemini_provider.go` | - |
| `pkg/bus/` | Internal Message Routing | `bus.go`, `types.go` | - |
| `pkg/session/` | Session Persistence | `manager.go` | - |
| `pkg/config/` | Configuration Management | `config.go` | - |
| `pkg/heartbeat/` | Periodic Task Service | `service.go` | `pkg/agent/` |
| `pkg/migrate/` | Migration from OpenClaw | `migrate.go`, `workspace.go` | `pkg/config/` |
| `pkg/skills/` | Dynamic Skill Loading | `loader.go`, `installer.go` | - |

## Entry Points & Config
- `cmd/picoclaw/main.go` - Main CLI application
- `config/config.example.json` - Template for user configuration
- `~/.picoclaw/config.json` - Default user configuration path

## Patterns & Conventions
- **Concurrency**: Extensive use of Go channels and `MessageBus` for decoupled communication.
- **Error Handling**: Standard Go `error` patterns; errors in tools are returned to the agent as text.
- **Safety**: Optional workspace restriction (`restrict_to_workspace`) for filesystem and shell tools.
- **Extensibility**: Tools and skills are registered dynamically; supports MCP (Model Context Protocol).

## File Index
| File | Role |
|------|------|
| `cmd/picoclaw/main.go` | Application bootstrap and CLI command routing. |
| `pkg/agent/loop.go` | Main agent loop handling LLM calls and tool iterations. |
| `pkg/agent/context.go` | Builds system prompts and message context for the LLM. |
| `pkg/agent/memory.go` | Handles long-term memory and session summarization. |
| `pkg/bus/bus.go` | Pub-sub message bus for internal communication. |
| `pkg/channels/manager.go` | Manages lifecycle and message routing for all active channels. |
| `pkg/channels/telegram.go` | Integration with Telegram Bot API. |
| `pkg/config/config.go` | Configuration structures and I/O logic. |
| `pkg/cron/service.go` | Scheduler for one-time and recurring tasks. |
| `pkg/heartbeat/service.go` | Triggers periodic agent tasks based on `HEARTBEAT.md`. |
| `pkg/mcp/client.go` | Client for Model Context Protocol (MCP) servers. |
| `pkg/migrate/migrate.go` | Logic for migrating workspace and config from OpenClaw. |
| `pkg/providers/types.go` | Common interfaces and types for LLM providers. |
| `pkg/session/manager.go` | Manages conversation history and state persistence on disk. |
| `pkg/skills/loader.go` | Discovers and loads dynamic skills from the workspace. |
| `pkg/tools/registry.go` | Central registry for available agent tools. |
| `pkg/tools/filesystem.go` | Tools for reading, writing, and listing files. |
| `pkg/tools/shell.go` | Tool for executing shell commands. |
| `pkg/tools/spawn.go` | Spawns asynchronous subagents for long-running tasks. |
| `pkg/tools/subagent.go` | Synchronous subagent execution tool. |
| `pkg/tools/web.go` | Web search tool supporting multiple backends (Brave, DDG, ZAI). |
| `pkg/tools/zai.go` | Specialized tools for Z.AI MCP services. |
| `pkg/state/state.go` | Manages persistent system state (e.g., last active channel). |
| `pkg/voice/transcriber.go` | Integration with voice-to-text services (e.g., Groq Whisper). |
