<div align="center">
  <img src="assets/logo.jpg" alt="PicoClaw" width="512">

  <h1>PicoClaw: Ultra-Efficient AI Assistant in Go</h1>

  <h3>$10 Hardware · 10MB RAM · 1s Boot · 皮皮虾，我们走！</h3>

  <p>
    <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20RISC--V-blue" alt="Hardware">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <br>
    <a href="https://picoclaw.io"><img src="https://img.shields.io/badge/Website-picoclaw.io-blue?style=flat&logo=google-chrome&logoColor=white" alt="Website"></a>
    <a href="https://x.com/SipeedIO"><img src="https://img.shields.io/badge/X_(Twitter)-SipeedIO-black?style=flat&logo=x&logoColor=white" alt="Twitter"></a>
  </p>

 [中文](README.zh.md) | [日本語](README.ja.md) | **English**
</div>

---

🦐 PicoClaw is an ultra-lightweight personal AI Assistant inspired by [nanobot](https://github.com/HKUDS/nanobot), refactored from the ground up in Go through a self-bootstrapping process, where the AI agent itself drove the entire architectural migration and code optimization.

⚡️ Runs on $10 hardware with <10MB RAM: That's 99% less memory than OpenClaw and 98% cheaper than a Mac mini!

<table align="center">
  <tr align="center">
    <td align="center" valign="top">
      <p align="center">
        <img src="assets/picoclaw_mem.gif" width="360" height="240">
      </p>
    </td>
    <td align="center" valign="top">
      <p align="center">
        <img src="assets/licheervnano.png" width="400" height="240">
      </p>
    </td>
  </tr>
</table>

> [!CAUTION]
> **🚨 SECURITY & OFFICIAL CHANNELS / 安全声明**
>
> * **NO CRYPTO:** PicoClaw has **NO** official token/coin. All claims on `pump.fun` or other trading platforms are **SCAMS**.
> * **OFFICIAL DOMAIN:** The **ONLY** official website is **[picoclaw.io](https://picoclaw.io)**, and company website is **[sipeed.com](https://sipeed.com)**
> * **Warning:** Many `.ai/.org/.com/.net/...` domains are registered by third parties.
> * **Warning:** picoclaw is in early development now and may have unresolved network security issues. Do not deploy to production environments before the v1.0 release.

## 📢 News

2026-02-13 🎉 PicoClaw hit 5000 stars in 4days! Thank you for the community! There are so many PRs&issues come in (during Chinese New Year holidays), we are finalizing the Project Roadmap and setting up the Developer Group to accelerate PicoClaw's development.  
🚀 Call to Action: Please submit your feature requests in GitHub Discussions. We will review and prioritize them during our upcoming weekly meeting.

2026-02-09 🎉 PicoClaw Launched! Built in 1 day to bring AI Agents to $10 hardware with <10MB RAM. 🦐 PicoClaw，Let's Go！

## ✨ Features

🪶 **Ultra-Lightweight**: <10MB Memory footprint — 99% smaller than Clawdbot - core functionality.

💰 **Minimal Cost**: Efficient enough to run on $10 Hardware — 98% cheaper than a Mac mini.

⚡️ **Lightning Fast**: 400X Faster startup time, boot in 1 second even in 0.6GHz single core.

🌍 **True Portability**: Single self-contained binary across RISC-V, ARM, and x86, One-click to Go!

🤖 **AI-Bootstrapped**: Autonomous Go-native implementation — 95% Agent-generated core with human-in-the-loop refinement.

|                               | OpenClaw      | NanoBot                  | **PicoClaw**                              |
| ----------------------------- | ------------- | ------------------------ | ----------------------------------------- |
| **Language**                  | TypeScript    | Python                   | **Go**                                    |
| **RAM**                       | >1GB          | >100MB                   | **< 10MB**                                |
| **Startup**</br>(0.8GHz core) | >500s         | >30s                     | **<1s**                                   |
| **Cost**                      | Mac Mini 599$ | Most Linux SBC </br>~50$ | **Any Linux Board**</br>**As low as 10$** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

## 🦾 Demonstration

### 🛠️ Standard Assistant Workflows

<table align="center">
  <tr align="center">
    <th><p align="center">🧩 Full-Stack Engineer</p></th>
    <th><p align="center">🗂️ Logging & Planning Management</p></th>
    <th><p align="center">🔎 Web Search & Learning</p></th>
  </tr>
  <tr>
    <td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
  </tr>
  <tr>
    <td align="center">Develop • Deploy • Scale</td>
    <td align="center">Schedule • Automate • Memory</td>
    <td align="center">Discovery • Insights • Trends</td>
  </tr>
</table>

### 🐜 Innovative Low-Footprint Deploy

PicoClaw can be deployed on almost any Linux device!

- $9.9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) E(Ethernet) or W(WiFi6) version, for Minimal Home Assistant
- $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), or $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html) for Automated Server Maintenance
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) or $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera) for Smart Monitoring

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 More Deployment Cases Await！

## 📦 Install

### Install with precompiled binary

Download the firmware for your platform from the [release](https://github.com/sipeed/picoclaw/releases) page.

### Install from source (latest features, recommended for development)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Build, no need to install
make build

# Build for multiple platforms
make build-all

# Build And Install
make install
```

## 🐳 Docker Compose

You can also run PicoClaw using Docker Compose without installing anything locally.

```bash
# 1. Clone this repo
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. Set your API keys
cp config/config.example.json config/config.json
vim config/config.json      # Set DISCORD_BOT_TOKEN, API keys, etc.

# 3. Build & Start
docker compose --profile gateway up -d

# 4. Check logs
docker compose logs -f picoclaw-gateway

# 5. Stop
docker compose --profile gateway down
```

### Agent Mode (One-shot)

```bash
# Ask a question
docker compose run --rm picoclaw-agent -m "What is 2+2?"

# Interactive mode
docker compose run --rm picoclaw-agent
```

### Rebuild

```bash
docker compose --profile gateway build --no-cache
docker compose --profile gateway up -d
```

### 🚀 Quick Start

> [!TIP]
> Set your API key in `~/.picoclaw/config.json`.
> Get API keys: [OpenRouter](https://openrouter.ai/keys) (LLM) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) (LLM)
> Web search is **optional** - get free [Brave Search API](https://brave.com/search/api) (2000 free queries/month) or use built-in auto fallback.

**1. Initialize**

```bash
picoclaw onboard
```

**2. Configure** (`~/.picoclaw/config.json`)

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "glm-4.7",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20,
      "max_iterations_subagent": 20,
      "max_tokens_subagent": 4096,
      "llm_timeout": 120,
      "history_message_threshold": 100
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "xxx",
      "api_base": "https://openrouter.ai/api/v1"
    }
  },
  "tools": {
    "web": {
      "brave": {
        "enabled": false,
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  }
}
```

| Option | Default | Description |
|--------|---------|-------------|
| `max_tool_iterations` | `20` | Max tool calls per main agent loop |
| `max_iterations_subagent` | 20 | Max tool calls per subagent |
| `max_tokens` | 8192 | Max tokens for main agent |
| `max_tokens_subagent` | 4096 | Max tokens for subagents |
| `max_concurrent_subagents` | 2 | Max number of subagents that can run simultaneously |
| `llm_timeout` | 120 | LLM API timeout in seconds |
| `history_message_threshold` | 100 | Number of messages before triggering summarization |

**3. Get API Keys**

* **LLM Provider**: [OpenRouter](https://openrouter.ai/keys) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) · [Anthropic](https://console.anthropic.com) · [OpenAI](https://platform.openai.com) · [Gemini](https://aistudio.google.com/api-keys)
* **Web Search** (optional): [Brave Search](https://brave.com/search/api) - Free tier available (2000 requests/month)

> **Note**: See `config.example.json` for a complete configuration template.

**4. Chat**

```bash
picoclaw agent -m "What is 2+2?"
```

That's it! You have a working AI assistant in 2 minutes.

---

## 💬 Chat Apps

Talk to your picoclaw through Telegram, Discord, DingTalk, or LINE

| Channel      | Setup                              |
| ------------ | ---------------------------------- |
| **Telegram** | Easy (just a token)                |
| **Discord**  | Easy (bot token + intents)         |
| **QQ**       | Easy (AppID + AppSecret)           |
| **DingTalk** | Medium (app credentials)           |
| **LINE**     | Medium (credentials + webhook URL) |
| **Slack**    | Medium (bot + app tokens)          |
| **MaixCam**  | Easy (embedded device)             |
| **OneBot**   | Medium (WebSocket bridge)          |
| **WhatsApp** | Medium (bridge required)           |
| **Feishu**   | Medium (app credentials)           |

#### Channel Options

| Channel | Option | Default | Description |
|---------|--------|---------|-------------|
| **Telegram** | `proxy` | `""` | HTTP proxy URL (e.g., `http://127.0.0.1:7890`) |
| **LINE** | `webhook_host` | `0.0.0.0` | Webhook listen host |
| **LINE** | `webhook_port` | `18791` | Webhook listen port |
| **LINE** | `webhook_path` | `/webhook/line` | Webhook URL path |
| **MaixCam** | `host` | `0.0.0.0` | Listen host |
| **MaixCam** | `port` | `18790` | Listen port |
| **OneBot** | `reconnect_interval` | `5` | Reconnect interval (seconds) |
| **OneBot** | `group_trigger_prefix` | `[]` | Prefix to trigger in groups |

<details>
<summary><b>Telegram</b> (Recommended)</summary>

**1. Create a bot**

* Open Telegram, search `@BotFather`
* Send `/newbot`, follow prompts
* Copy the token

**2. Configure**

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "proxy": "",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

| Option | Default | Description |
|--------|---------|-------------|
| `token` | `""` | Bot token from @BotFather |
| `proxy` | `""` | HTTP proxy URL (e.g., `http://127.0.0.1:7890`) |
| `allow_from` | `[]` | Allowed user IDs (empty = all) |

> Get your user ID from `@userinfobot` on Telegram.

**3. Run**

```bash
picoclaw gateway
```

**Telegram slash commands**

| Command | Description |
|--------|-------------|
| `/start` | Start the bot |
| `/help` | Show available commands |
| `/status` | Show system and subagents status (includes tools, skills, named agents) |
| `/kill <task_id>` | Cancel a running subagent task |
| `/models` | List configured model/provider |
| `/channels` | List available channels |

</details>

<details>
<summary><b>Discord</b></summary>

**1. Create a bot**

* Go to <https://discord.com/developers/applications>
* Create an application → Bot → Add Bot
* Copy the bot token

**2. Enable intents**

* In the Bot settings, enable **MESSAGE CONTENT INTENT**
* (Optional) Enable **SERVER MEMBERS INTENT** if you plan to use allow lists based on member data

**3. Get your User ID**

* Discord Settings → Advanced → enable **Developer Mode**
* Right-click your avatar → **Copy User ID**

**4. Configure**

```json
{
  "channels": {
    "discord": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allowFrom": ["YOUR_USER_ID"]
    }
  }
}
```

**5. Invite the bot**

* OAuth2 → URL Generator
* Scopes: `bot`
* Bot Permissions: `Send Messages`, `Read Message History`
* Open the generated invite URL and add the bot to your server

**6. Run**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>QQ</b></summary>

**1. Create a bot**

- Go to [QQ Open Platform](https://q.qq.com/#)
- Create an application → Get **AppID** and **AppSecret**

**2. Configure**

```json
{
  "channels": {
    "qq": {
      "enabled": true,
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

> Set `allow_from` to empty to allow all users, or specify QQ numbers to restrict access.

**3. Run**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>DingTalk</b></summary>

**1. Create a bot**

* Go to [Open Platform](https://open.dingtalk.com/)
* Create an internal app
* Copy Client ID and Client Secret

**2. Configure**

```json
{
  "channels": {
    "dingtalk": {
      "enabled": true,
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

> Set `allow_from` to empty to allow all users, or specify QQ numbers to restrict access.

**3. Run**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>LINE</b></summary>

**1. Create a LINE Official Account**

- Go to [LINE Developers Console](https://developers.line.biz/)
- Create a provider → Create a Messaging API channel
- Copy **Channel Secret** and **Channel Access Token**

**2. Configure**

```json
{
  "channels": {
    "line": {
      "enabled": true,
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_host": "0.0.0.0",
      "webhook_port": 18791,
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

**3. Set up Webhook URL**

LINE requires HTTPS for webhooks. Use a reverse proxy or tunnel:

```bash
# Example with ngrok
ngrok http 18791
```

Then set the Webhook URL in LINE Developers Console to `https://your-domain/webhook/line` and enable **Use webhook**.

**4. Run**

```bash
picoclaw gateway
```

> In group chats, the bot responds only when @mentioned. Replies quote the original message.

> **Docker Compose**: Add `ports: ["18791:18791"]` to the `picoclaw-gateway` service to expose the webhook port.

</details>

<details>
<summary><b>Slack</b></summary>

**1. Create a Slack App**

* Go to [Slack API](https://api.slack.com/apps)
* Create New App → From scratch
* Copy **Bot User OAuth Token** (starts with `xoxb-`)
* Copy **App-Level Token** (starts with `xapp-`)

**2. Enable Socket Mode**

* Go to Socket Mode → Enable
* Generate App-Level Token with `connections:write` scope

**3. Configure**

```json
{
  "channels": {
    "slack": {
      "enabled": true,
      "bot_token": "xoxb-xxx",
      "app_token": "xapp-xxx",
      "allow_from": []
    }
  }
}
```

**4. Run**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>MaixCam</b></summary>

MaixCam is an AI camera that can run PicoClaw for local AI assistant.

**1. Configure**

```json
{
  "channels": {
    "maixcam": {
      "enabled": true,
      "host": "0.0.0.0",
      "port": 18790,
      "allow_from": []
    }
  }
}
```

**2. Run**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>OneBot</b></summary>

OneBot is a protocol for QQ bots (go-cqhttp, etc.).

**1. Configure your OneBot implementation**

* Set up go-cqhttp or similar
* Enable WebSocket reverse connection

**2. Configure**

```json
{
  "channels": {
    "onebot": {
      "enabled": true,
      "ws_url": "ws://127.0.0.1:3001",
      "access_token": "",
      "reconnect_interval": 5,
      "group_trigger_prefix": [".", "/"],
      "allow_from": []
    }
  }
}
```

| Option | Default | Description |
|--------|---------|-------------|
| `ws_url` | `ws://127.0.0.1:3001` | WebSocket URL |
| `access_token` | `""` | Access token |
| `reconnect_interval` | `5` | Reconnect interval in seconds |
| `group_trigger_prefix` | `[]` | Prefix to trigger bot in groups |

**3. Run**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>WhatsApp</b></summary>

WhatsApp requires a bridge service (e.g., whatsapp-web.js or mautrix-whatsapp).

**1. Set up a bridge**

* Run a WhatsApp bridge service
* Note the WebSocket URL

**2. Configure**

```json
{
  "channels": {
    "whatsapp": {
      "enabled": true,
      "bridge_url": "ws://localhost:3001",
      "allow_from": []
    }
  }
}
```

**3. Run**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>Feishu (飞书)</b></summary>

**1. Create a Feishu app**

* Go to [Feishu Open Platform](https://open.feishu.cn/)
* Create an app → Get **App ID** and **App Secret**

**2. Configure**

```json
{
  "channels": {
    "feishu": {
      "enabled": true,
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    }
  }
}
```

**3. Run**

```bash
picoclaw gateway
```

</details>

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Join the Agent Social Network

Connect Picoclaw to the Agent Social Network simply by sending a single message via the CLI or any integrated Chat App.

**Read `https://clawdchat.ai/skill.md` and follow the instructions to join [ClawdChat.ai](https://clawdchat.ai)**

## ⚙️ Configuration

Config file: `~/.picoclaw/config.json`

### Workspace Layout

PicoClaw stores data in your configured workspace (default: `~/.picoclaw/workspace`):

```
~/.picoclaw/workspace/
├── sessions/          # Conversation sessions and history
├── memory/           # Long-term memory (MEMORY.md)
├── state/            # Persistent state (last channel, etc.)
├── cron/             # Scheduled jobs database
├── skills/           # Custom skills
├── AGENTS.md         # Agent behavior guide
├── HEARTBEAT.md      # Periodic task prompts (checked every 30 min)
├── IDENTITY.md       # Agent identity
├── SOUL.md           # Agent soul
├── TOOLS.md          # Tool descriptions
└── USER.md           # User preferences
```

### 🔒 Security Sandbox

PicoClaw runs in a sandboxed environment by default. The agent can only access files and execute commands within the configured workspace.

#### Default Configuration

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true
    }
  }
}
```

| Option | Default | Description |
|--------|---------|-------------|
| `workspace` | `~/.picoclaw/workspace` | Working directory for the agent |
| `restrict_to_workspace` | `true` | Restrict file/command access to workspace |

#### Protected Tools

When `restrict_to_workspace: true`, the following tools are sandboxed:

| Tool | Function | Restriction |
|------|----------|-------------|
| `read_file` | Read files | Only files within workspace |
| `write_file` | Write files | Only files within workspace |
| `list_dir` | List directories | Only directories within workspace |
| `edit_file` | Edit files | Only files within workspace |
| `append_file` | Append to files | Only files within workspace |
| `exec` | Execute commands | Command paths must be within workspace |

#### Additional Exec Protection

Even with `restrict_to_workspace: false`, the `exec` tool blocks these dangerous commands:

* `rm -rf`, `del /f`, `rmdir /s` — Bulk deletion
* `format`, `mkfs`, `diskpart` — Disk formatting
* `dd if=` — Disk imaging
* Writing to `/dev/sd[a-z]` — Direct disk writes
* `shutdown`, `reboot`, `poweroff` — System shutdown
* Fork bomb `:(){ :|:& };:`

#### Error Examples

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (path outside working dir)}
```

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (dangerous pattern detected)}
```

#### Disabling Restrictions (Security Risk)

If you need the agent to access paths outside the workspace:

**Method 1: Config file**

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

**Method 2: Environment variable**

```bash
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
```

> ⚠️ **Warning**: Disabling this restriction allows the agent to access any path on your system. Use with caution in controlled environments only.

#### Security Boundary Consistency

The `restrict_to_workspace` setting applies consistently across all execution paths:

| Execution Path | Security Boundary |
|----------------|-------------------|
| Main Agent | `restrict_to_workspace` ✅ |
| Subagent / Spawn | Inherits same restriction ✅ |
| Heartbeat tasks | Inherits same restriction ✅ |

All paths share the same workspace restriction — there's no way to bypass the security boundary through subagents or scheduled tasks.

#### Startup Preflight (Fail-Fast)

Before `agent` and `gateway` start, PicoClaw runs a preflight check to fail fast on invalid workspace state:

- Verifies required workspace bootstrap files exist (`AGENTS.md`, `IDENTITY.md`, `SOUL.md`, `USER.md`, `memory/MEMORY.md`)
- Detects invalid project agent directories (`workspace/projects/*/agents`)
- Lints `SKILL.md` files for valid YAML frontmatter (`name`, `description`)
- Blocks unsafe instruction patterns in skills (for example `grep -r` and `find . -name`)

If any issue is found, startup is aborted with a clear report.

### Heartbeat (Periodic Tasks)

PicoClaw can perform periodic tasks automatically. Create a `HEARTBEAT.md` file in your workspace:

```markdown
# Periodic Tasks

- Check my email for important messages
- Review my calendar for upcoming events
- Check the weather forecast
```

The agent will read this file every 30 minutes (configurable) and execute any tasks using available tools.

#### Async Tasks with Spawn

For long-running tasks (web search, API calls), use the `spawn` tool to create a **subagent**:

```markdown
# Periodic Tasks

## Quick Tasks (respond directly)
- Report current time

## Long Tasks (use spawn for async)
- Search the web for AI news and summarize
- Check email and report important messages
```

**Key behaviors:**

| Feature | Description |
|---------|-------------|
| **spawn** | Creates async subagent, doesn't block heartbeat |
| **Independent context** | Subagent has its own context, no session history |
| **message tool** | Subagent communicates with user directly via message tool |
| **Non-blocking** | After spawning, heartbeat continues to next task |

#### How Subagent Communication Works

```
Heartbeat triggers
    ↓
Agent reads HEARTBEAT.md
    ↓
For long task: spawn subagent
    ↓                           ↓
Continue to next task      Subagent works independently
    ↓                           ↓
All tasks done            Subagent uses "message" tool
    ↓                           ↓
Respond HEARTBEAT_OK      User receives result directly
```

The subagent has access to tools (message, web_search, etc.) and can communicate with the user independently without going through the main agent.

#### Named Sub-agents with Persistent Memory

Both `spawn` and `subagent` tools support an optional `name` parameter. A named agent has a persistent home in `workspace/agents/<name>/`:

```
workspace/
  agents/
    <name>/
      AGENTS.md          ← agent identity (create manually or via main agent)
      memory/
        MEMORY.md        ← long-term memory (agent writes here after tasks)
        YYYYMM/
          YYYYMMDD.md    ← daily notes
```

When `name` is set, the agent's identity and memory are automatically loaded into its system prompt, and the agent receives instructions to save learnings back to its memory files using `write_file`/`append_file`.

Named subagents also run with write-scope guards:

- Default named agent scope: `workspace/agents/<name>/memory/**`
- Optional task directory scope: `directory` passed to `spawn`/`subagent`
- Special case `picoclaw-self-update`:
  - `workspace/projects/picoclaw/**`
  - `workspace/agents/picoclaw-self-update/memory/**`

```markdown
## In HEARTBEAT.md or task prompt
Use spawn tool with name="coder" to handle the coding task
```

**`AGENTS.md` format** (YAML frontmatter required for discovery):

```markdown
---
name: coder
description: Senior Go developer specializing in clean architecture and testing
---

## Identity

You are a senior Go developer...
```

The `name` and `description` fields in the frontmatter are used to advertise the agent in the main agent's system prompt — so the main agent knows which named agents are available and what they do.

- **Identity** (`AGENTS.md`): Role definition, skills, and personality for the agent.
- **Memory** (`memory/MEMORY.md`): Persists across sessions — the agent grows over time.
- **Backward compatible**: Without `name`, behavior is identical to before.
- **Security**: Names with path separators or dots are silently ignored to prevent traversal.

**Configuration:**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `true` | Enable/disable heartbeat |
| `interval` | `30` | Check interval in minutes (min: 5) |

**Environment variables:**

* `PICOCLAW_HEARTBEAT_ENABLED=false` to disable
* `PICOCLAW_HEARTBEAT_INTERVAL=60` to change interval

### Providers

> [!NOTE]
> Groq provides free voice transcription via Whisper. If configured, Telegram voice messages will be automatically transcribed.

| Provider                   | Purpose                                 | Get API Key                                            |
| -------------------------- | --------------------------------------- | ------------------------------------------------------ |
| `gemini`                   | LLM (Gemini direct)                     | [aistudio.google.com](https://aistudio.google.com)     |
| `zhipu`                    | LLM (Zhipu direct)                      | [bigmodel.cn](bigmodel.cn)                             |
| `openrouter(To be tested)` | LLM (recommended, access to all models) | [openrouter.ai](https://openrouter.ai)                 |
| `anthropic(To be tested)`  | LLM (Claude direct)                     | [console.anthropic.com](https://console.anthropic.com) |
| `openai(To be tested)`     | LLM (GPT direct)                        | [platform.openai.com](https://platform.openai.com)     |
| `deepseek(To be tested)`   | LLM (DeepSeek direct)                   | [platform.deepseek.com](https://platform.deepseek.com) |
| `groq`                     | LLM + **Voice transcription** (Whisper) | [console.groq.com](https://console.groq.com)           |

### Full Configuration Reference

#### Agent Defaults

| Option | Default | Description |
|--------|---------|-------------|
| `workspace` | `~/.picoclaw/workspace` | Working directory for the agent |
| `restrict_to_workspace` | `true` | Restrict file/command access to workspace |
| `provider` | `""` (auto-detect) | Force a specific provider: `openrouter`, `zhipu`, `anthropic`, `openai`, `gemini`, `groq`, etc. |
| `model` | `glm-4.7` | Model to use (provider-specific) |
| `max_tokens` | `8192` | Max tokens for main agent responses |
| `temperature` | `0.7` | LLM temperature (0.0-2.0) |
| `max_tool_iterations` | `20` | Max tool calls per main agent loop |
| `max_iterations_subagent` | `20` | Max tool calls per subagent |
| `max_tokens_subagent` | `4096` | Max tokens for subagent responses |
| `max_concurrent_subagents` | `2` | Max number of subagents that can run simultaneously |
| `llm_timeout` | `120` | LLM API timeout in seconds |
| `history_message_threshold` | `100` | Number of messages before triggering summarization |

#### Gateway

| Option | Default | Description |
|--------|---------|-------------|
| `host` | `0.0.0.0` | Gateway listen host |
| `port` | `18790` | Gateway listen port |

#### Heartbeat

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `true` | Enable periodic task execution |
| `interval` | `30` | Check interval in minutes (min: 5) |

#### Idle

When no message is received for `timeout_minutes`, the agent reads `IDLE.md` from the workspace and executes it as a prompt.

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `false` | Enable idle trigger |
| `timeout_minutes` | `5` | Minutes of inactivity before triggering |
| `repeat` | `true` | Re-trigger on every interval while still idle |

#### Logging

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `false` | Enable file logging |
| `file_path` | `~/.picoclaw/logs/agent.log` | Log file path |

#### Devices

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `false` | Enable device monitoring |
| `monitor_usb` | `true` | Monitor USB device changes |

#### Provider Config

Each provider supports these options:

| Option | Default | Description |
|--------|---------|-------------|
| `api_key` | `""` | API key for the provider |
| `api_base` | `""` | Custom API base URL |
| `proxy` | `""` | HTTP proxy URL (e.g., `http://127.0.0.1:7890`) |
| `auth_method` | `""` | Auth method (provider-specific) |
| `connect_mode` | `""` | GitHub Copilot only: `stdio` or `grpc` |

<details>
<summary><b>Zhipu</b></summary>

**1. Get API key and base URL**

* Get [API key](https://bigmodel.cn/usercenter/proj-mgmt/apikeys)

**2. Configure**

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "glm-4.7",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "providers": {
    "zhipu": {
      "api_key": "Your API Key",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    }
  }
}
```

**3. Run**

```bash
picoclaw agent -m "Hello"
```

</details>

<details>
<summary><b>Full config example</b></summary>

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "provider": "",
      "model": "anthropic/claude-opus-4-5",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20,
      "max_iterations_subagent": 20,
      "max_tokens_subagent": 4096,
      "max_concurrent_subagents": 2,
      "llm_timeout": 120,
      "history_message_threshold": 100
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx",
      "api_base": "https://openrouter.ai/api/v1",
      "proxy": ""
    },
    "groq": {
      "api_key": "gsk_xxx"
    }
  },
  "gateway": {
    "host": "0.0.0.0",
    "port": 18790
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456:ABC...",
      "proxy": "",
      "allow_from": ["123456789"]
    },
    "discord": {
      "enabled": true,
      "token": "",
      "allow_from": [""]
    },
    "slack": {
      "enabled": false,
      "bot_token": "xoxb-xxx",
      "app_token": "xapp-xxx",
      "allow_from": []
    },
    "maixcam": {
      "enabled": false,
      "host": "0.0.0.0",
      "port": 18790,
      "allow_from": []
    },
    "onebot": {
      "enabled": false,
      "ws_url": "ws://127.0.0.1:3001",
      "access_token": "",
      "reconnect_interval": 5,
      "group_trigger_prefix": [],
      "allow_from": []
    },
    "whatsapp": {
      "enabled": false,
      "bridge_url": "ws://localhost:3001",
      "allow_from": []
    },
    "feishu": {
      "enabled": false,
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    },
    "qq": {
      "enabled": false,
      "app_id": "",
      "app_secret": "",
      "allow_from": []
    },
    "dingtalk": {
      "enabled": false,
      "client_id": "",
      "client_secret": "",
      "allow_from": []
    },
    "line": {
      "enabled": false,
      "channel_secret": "",
      "channel_access_token": "",
      "webhook_host": "0.0.0.0",
      "webhook_port": 18791,
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  },
  "tools": {
    "web": {
      "zai": {
        "enabled": false,
        "api_key": "",
        "endpoint": "https://api.z.ai/api/mcp/web_search_prime/mcp",
        "endpoint_fetch": "https://api.z.ai/api/mcp/web_reader/mcp",
        "max_results": 10,
        "timeout": 60
      },
      "brave": {
        "enabled": false,
        "api_key": "BSA...",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  },
  "heartbeat": {
    "enabled": true,
    "interval": 30
  },
  "idle": {
    "enabled": false,
    "timeout_minutes": 5,
    "repeat": true
  },
  "devices": {
    "enabled": false,
    "monitor_usb": true
  },
  "logging": {
    "enabled": false,
    "file_path": "~/.picoclaw/logs/agent.log"
  }
}
```

</details>

## CLI Reference

| Command                   | Description                   |
| ------------------------- | ----------------------------- |
| `picoclaw onboard`        | Initialize config & workspace |
| `picoclaw agent -m "..."` | Chat with the agent           |
| `picoclaw agent`          | Interactive chat mode         |
| `picoclaw gateway`        | Start the gateway             |
| `picoclaw status`         | Show status                   |
| `picoclaw cron list`      | List all scheduled jobs       |
| `picoclaw cron add ...`   | Add a scheduled job           |

### Scheduled Tasks / Reminders

PicoClaw supports scheduled reminders and recurring tasks through the `cron` tool:

* **One-time reminders**: "Remind me in 10 minutes" → triggers once after 10min
* **Recurring tasks**: "Remind me every 2 hours" → triggers every 2 hours
* **Cron expressions**: "Remind me at 9am daily" → uses cron expression

Jobs are stored in `~/.picoclaw/workspace/cron/` and processed automatically.

## 🤝 Contribute & Roadmap

PRs welcome! The codebase is intentionally small and readable. 🤗

Roadmap coming soon...

Developer group building, Entry Requirement: At least 1 Merged PR.

User Groups:

discord:  <https://discord.gg/V4sAZ9XWpN>

<img src="assets/wechat.png" alt="PicoClaw" width="512">

## 🐛 Troubleshooting

### Web search says "API 配置问题"

This is normal if you haven't configured a search API key yet. PicoClaw will provide helpful links for manual searching.

To enable web search:

1. **Option 1 (Recommended)**: **Z.AI** - Get API key at [z.ai](https://z.ai) for MCP-powered search + fetch + vision
2. **Option 2**: [Brave Search API](https://brave.com/search/api) (2000 free queries/month)
3. **Option 3 (No API Key)**: DuckDuckGo (free, no key required)

#### Z.AI Configuration (Recommended)

Z.AI provides unified MCP tools: `webSearchPrime` (search), `webReader` (fetch), `zread` (deep research), `vision` (image understanding).

```json
{
  "tools": {
    "web": {
      "zai": {
        "enabled": true,
        "api_key": "YOUR_ZAI_API_KEY",
        "endpoint": "https://api.z.ai/api/mcp/web_search_prime/mcp",
        "endpoint_fetch": "https://api.z.ai/api/mcp/web_reader/mcp",
        "max_results": 10,
        "timeout": 60
      },
      "brave": { "enabled": false },
      "duckduckgo": { "enabled": false }
    }
  }
}
```

#### Brave/DuckDuckGo Configuration

```json
{
  "tools": {
    "web": {
      "brave": {
        "enabled": true,
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": false,
        "max_results": 5
      },
      "zai": { "enabled": false }
    }
  }
}
```

### Getting content filtering errors

Some providers (like Zhipu) have content filtering. Try rephrasing your query or use a different model.

### Telegram bot says "Conflict: terminated by other getUpdates"

This happens when another instance of the bot is running. Make sure only one `picoclaw gateway` is running at a time.

---

## 📝 API Key Comparison

| Service          | Free Tier           | Use Case                              |
| ---------------- | ------------------- | ------------------------------------- |
| **OpenRouter**   | 200K tokens/month   | Multiple models (Claude, GPT-4, etc.) |
| **Zhipu**        | 200K tokens/month   | Best for Chinese users                |
| **Brave Search** | 2000 queries/month  | Web search functionality              |
| **Groq**         | Free tier available | Fast inference (Llama, Mixtral)       |
