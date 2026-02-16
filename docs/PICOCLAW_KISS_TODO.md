# PicoClaw KISS Refactoring TODO

## Status: TRANSFORMED TO PICOCLAW

**Date:** 2026-02-16T07:42

---

## Current Focus

**Main Task:** Transform nanobot to run as PicoClaw with default config (no ZAI).

**Config Location:** `/home/nanobot/.nanobot/workspace/.picoclaw/config.json`

**Provider Stack:**
- Default: OpenAI (gpt-4.1-mini)
- Fallback: Anthropic (optional)
- Web: DuckDuckGo (no API key required)

**Channels:**
- Telegram: disabled (for stability testing)

---

## KISS Plans (Ready for Implementation)

### CONFIG_KISS.md
- **Status:** ✅ APPROVED (Review V3)
- **Params:** 5 parameters in `agents.defaults`
  - `max_iterations_main` (50)
  - `max_iterations_subagent` (20)
  - `max_tokens_main` (8192)
  - `max_tokens_subagent` (4096)
  - `llm_timeout` (120)
- **Location:** `projects/picoclaw/docs/PICOCLAW_CONFIG_KISS.md`

### SUBAGENT_KISS.md
- **Status:** ✅ APPROVED (Review V3 - minor issues, fix pending)
- **Tools:** 4 tools in `subagent_tools.go`
  - `subagent_status` - list subagents
  - `subagent_history` - view conversation history
  - `subagent_message` - send guidance
  - `subagent_cancel` - cancel subagent
- **Location:** `projects/picoclaw/docs/PICOCLAW_SUBAGENT_KISS.md`

### MCP_KISS.md
- **Status:** ⚠️ NEEDS REVISION (Review V3 - SDK API issues, fix pending)
- **Provider:** ZAI MCP for web search/fetch
- **Issues:** SDK API calls must be fixed
- **Location:** `projects/picoclaw/docs/PICOCLAW_MCP_KISS.md`

---

## Reviews History

### Round 3 (Initial Reviews)
- **CONFIG_REVIEW_V3:** ✅ APPROVED
- **SUBAGENT_REVIEW_V3:** ⚠️ Minor issues (mutex, type assertions)
- **MCP_REVIEW_V3:** ❌ FAIL (SDK API mismatches)

### Round 4 (Convention-Fixes + Reviews V4)
- **config-fix-v4:** ✅ Fixed to use `agents.defaults.*`
- **subagent-fix-v4:** ✅ Fixed mutex + type assertions
- **mcp-fix-v4:** ✅ Fixed SDK API calls
- **Reviews V4:** ✅ COMPLETE (2026-02-16T08:09)
  - CONFIG_REVIEW_V4: ⚠️ 90% correct - needs env tag removal + implementation
  - SUBAGENT_REVIEW_V4: ✅ APPROVED - already implemented
  - MCP_REVIEW_V4: ✅ APPROVED - ready for implementation

### Round 5 (Implementation)
- **config-impl-v5:** ✅ COMPLETE - 3 config fields implemented, no env tags
- **mcp-impl-v5:** ✅ COMPLETE - ZAI MCP client and tools implemented

---

## Status: ✅ IMPLEMENTATION COMPLETE

**Date:** 2026-02-16T09:20

All KISS refactoring tasks completed:
- Config: 3 fields added, defaults set, used by SubagentManager
- Subagent tools: 4 tools in subagent_tools.go
- MCP: Client + ZAI providers implemented

**Next:** Test PicoClaw with default config

---

## Notes

- **Reviews V4 summary:** See `REVIEWS_V4_SUMMARY.md`
- **Next:** Complete implementations → Test PicoClaw with default config
