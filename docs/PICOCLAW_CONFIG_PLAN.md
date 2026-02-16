# PicoClaw Configuration Enhancement Plan

## Executive Summary

This document proposes adding configurable parameters to PicoClaw for LLM execution limits. Currently, several critical parameters are hard-coded throughout the codebase, making it difficult for users to optimize performance for different use cases.

## Current State Analysis

### Existing Configuration

**File:** `config/config.example.json`

The current configuration includes:
```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "provider": "zhipu",
      "model": "glm-4.7",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  }
}
```

### Hard-Coded Parameters Identified

#### 1. **max_tool_iterations** (Partially Configurable)
- **Config:** `agents.defaults.max_tool_iterations` (default: 20)
- **Used in:** `pkg/agent/loop.go:143`
- **Issue:** Subagent hard-coded to 10 iterations in `pkg/tools/subagent.go:44`

#### 2. **max_tokens** (Partially Configurable)
- **Config:** `agents.defaults.max_tokens` (default: 8192)
- **Hard-coded locations:**
  - `pkg/agent/loop.go:435` - LLM request: `"max_tokens": 8192`
  - `pkg/agent/loop.go:450` - LLM request: `"max_tokens": 8192`
  - `pkg/agent/loop.go:727` - Summarization: `"max_tokens": 1024`
  - `pkg/agent/loop.go:762` - Summarization: `"max_tokens": 1024`
  - `pkg/tools/subagent.go:134` - Subagent: `"max_tokens": 4096`
  - `pkg/tools/subagent.go:292` - Subagent: `"max_tokens": 4096`
  - `pkg/tools/toolloop.go:59` - Tool loop default: `"max_tokens": 4096`

#### 3. **llm_timeout** (NOT Configurable)
- **Hard-coded locations:**
  - `pkg/providers/http_provider.go` - HTTP client timeout: `120 * time.Second`
  - `pkg/agent/loop.go:752` - Summarization timeout: `120 * time.Second`

## Proposed Configuration Structure

### New JSON Configuration

Add a new `execution` section to group LLM execution parameters:

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "provider": "zhipu",
      "model": "glm-4.7",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "execution": {
    "max_iterations": 50,
    "max_tokens": 8192,
    "llm_timeout": 120,
    "subagent": {
      "max_iterations": 10,
      "max_tokens": 4096
    },
    "summarization": {
      "max_tokens": 1024,
      "timeout": 120
    }
  }
}
```

### Configuration Parameters

#### Main Execution Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `execution.max_iterations` | int | 50 | Maximum tool iterations for main agent |
| `execution.max_tokens` | int | 8192 | Maximum tokens for LLM responses |
| `execution.llm_timeout` | int | 120 | Timeout in seconds for LLM API calls |

#### Subagent Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `execution.subagent.max_iterations` | int | 10 | Maximum tool iterations for subagents |
| `execution.subagent.max_tokens` | int | 4096 | Maximum tokens for subagent LLM responses |

#### Summarization Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `execution.summarization.max_tokens` | int | 1024 | Maximum tokens for summarization responses |
| `execution.summarization.timeout` | int | 120 | Timeout in seconds for summarization |

## Implementation Plan

### Phase 1: Configuration Schema Updates

**File:** `pkg/config/config.go`

1. Add new `ExecutionConfig` struct:
```go
type ExecutionConfig struct {
    MaxIterations  int                   `json:"max_iterations" env:"PICOCLAW_EXECUTION_MAX_ITERATIONS"`
    MaxTokens      int                   `json:"max_tokens" env:"PICOCLAW_EXECUTION_MAX_TOKENS"`
    LLMTimeout     int                   `json:"llm_timeout" env:"PICOCLAW_EXECUTION_LLM_TIMEOUT"`
    Subagent       SubagentConfig        `json:"subagent"`
    Summarization  SummarizationConfig   `json:"summarization"`
}

type SubagentConfig struct {
    MaxIterations int `json:"max_iterations" env:"PICOCLAW_EXECUTION_SUBAGENT_MAX_ITERATIONS"`
    MaxTokens     int `json:"max_tokens" env:"PICOCLAW_EXECUTION_SUBAGENT_MAX_TOKENS"`
}

type SummarizationConfig struct {
    MaxTokens int `json:"max_tokens" env:"PICOCLAW_EXECUTION_SUMMARIZATION_MAX_TOKENS"`
    Timeout   int `json:"timeout" env:"PICOCLAW_EXECUTION_SUMMARIZATION_TIMEOUT"`
}
```

2. Update main `Config` struct:
```go
type Config struct {
    Agents     AgentsConfig     `json:"agents"`
    Channels   ChannelsConfig   `json:"channels"`
    Providers  ProvidersConfig  `json:"providers"`
    Gateway    GatewayConfig    `json:"gateway"`
    Tools      ToolsConfig      `json:"tools"`
    Heartbeat  HeartbeatConfig  `json:"heartbeat"`
    Devices    DevicesConfig    `json:"devices"`
    Execution  ExecutionConfig  `json:"execution"`  // NEW
    mu         sync.RWMutex
}
```

3. Add defaults in `DefaultConfig()`:
```go
Execution: ExecutionConfig{
    MaxIterations: 50,
    MaxTokens:     8192,
    LLMTimeout:    120,
    Subagent: SubagentConfig{
        MaxIterations: 10,
        MaxTokens:     4096,
    },
    Summarization: SummarizationConfig{
        MaxTokens: 1024,
        Timeout:   120,
    },
},
```

4. Add `Validate()` method to `Config`:
```go
func (c *Config) Validate() error {
    // Validate provider field consistency
    if c.Agents.Defaults.Model != "" && c.Agents.Defaults.Provider == "" {
        provider := inferProviderFromModel(c.Agents.Defaults.Model)
        if provider == "" {
            return fmt.Errorf("cannot infer provider for model '%s', please set 'provider' field explicitly", c.Agents.Defaults.Model)
        }
        c.Agents.Defaults.Provider = provider
    }
    
    // Validate provider credentials
    if c.Agents.Defaults.Provider != "" {
        if !c.hasProviderCredentials(c.Agents.Defaults.Provider) {
            return fmt.Errorf("no API key configured for provider '%s'", c.Agents.Defaults.Provider)
        }
    }
    
    // Validate temperature range
    if c.Agents.Defaults.Temperature < 0.0 || c.Agents.Defaults.Temperature > 2.0 {
        return fmt.Errorf("temperature must be between 0.0 and 2.0, got %f", c.Agents.Defaults.Temperature)
    }
    
    // Validate max_tool_iterations
    if c.Agents.Defaults.MaxToolIterations < 1 {
        return fmt.Errorf("max_tool_iterations must be at least 1, got %d", c.Agents.Defaults.MaxToolIterations)
    }
    
    // Validate workspace path
    if c.Agents.Defaults.Workspace == "" {
        return fmt.Errorf("workspace path cannot be empty")
    }
    
    // Validate heartbeat interval
    if c.Heartbeat.Enabled && c.Heartbeat.Interval < 5 {
        return fmt.Errorf("heartbeat interval must be at least 5 minutes, got %d", c.Heartbeat.Interval)
    }
    
    // Check for port conflicts
    if err := c.validatePortConflicts(); err != nil {
        return err
    }
    
    // Validate proxy URLs
    if err := c.validateProxyURLs(); err != nil {
        return err
    }
    
    return nil
}
```

### Phase 2: Update Agent Loop

**File:** `pkg/agent/loop.go`

1. Update `AgentLoop` struct to use configurable values:
```go
type AgentLoop struct {
    bus              *bus.MessageBus
    provider         providers.LLMProvider
    workspace        string
    model            string
    contextWindow    int
    maxIterations    int
    maxTokens        int           // NEW
    llmTimeout       time.Duration // NEW
    summarizationCfg SummarizationConfig // NEW
    sessions         *session.SessionManager
    state            *state.Manager
    contextBuilder   *ContextBuilder
    tools            *tools.ToolRegistry
    running          atomic.Bool
    summarizing      sync.Map
}
```

2. Update `NewAgentLoop()` to read from config:
```go
return &AgentLoop{
    bus:            msgBus,
    provider:       provider,
    workspace:      workspace,
    model:          cfg.Agents.Defaults.Model,
    contextWindow:  cfg.Agents.Defaults.MaxTokens,
    maxIterations:  cfg.Execution.MaxIterations,
    maxTokens:      cfg.Execution.MaxTokens,           // NEW
    llmTimeout:     time.Duration(cfg.Execution.LLMTimeout) * time.Second, // NEW
    summarizationCfg: cfg.Execution.Summarization,     // NEW
    sessions:       sessionsManager,
    state:          stateManager,
    contextBuilder: contextBuilder,
    tools:          toolsRegistry,
    summarizing:    sync.Map{},
}
```

3. Replace hard-coded `"max_tokens": 8192` with `al.maxTokens`:
```go
// Before (line 435, 450):
response, err := al.provider.Chat(ctx, messages, providerToolDefs, al.model, map[string]interface{}{
    "max_tokens":  8192,
    "temperature": 0.7,
})

// After:
response, err := al.provider.Chat(ctx, messages, providerToolDefs, al.model, map[string]interface{}{
    "max_tokens":  al.maxTokens,
    "temperature": 0.7,
})
```

4. Update summarization timeout (line 752):
```go
// Before:
ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)

// After:
ctx, cancel := context.WithTimeout(context.Background(), time.Duration(al.summarizationCfg.Timeout)*time.Second)
```

5. Update summarization max_tokens (line 727, 762):
```go
// Use al.summarizationCfg.MaxTokens instead of hard-coded 1024
```

### Phase 3: Update HTTP Provider

**File:** `pkg/providers/http_provider.go`

1. Make timeout configurable:
```go
type HTTPProvider struct {
    client    *http.Client
    apiKeys   map[string]string
    apiBases  map[string]string
    timeout   time.Duration  // NEW
}

func NewHTTPProvider(apiKeys, apiBases map[string]string, timeout time.Duration) *HTTPProvider {
    if timeout == 0 {
        timeout = 120 * time.Second
    }
    return &HTTPProvider{
        client: &http.Client{
            Timeout: timeout,
        },
        apiKeys:  apiKeys,
        apiBases: apiBases,
        timeout:  timeout,
    }
}
```

### Phase 4: Update Subagent

**File:** `pkg/tools/subagent.go`

1. Update `SubagentManager` to use configurable values:
```go
type SubagentManager struct {
    tasks          map[string]*SubagentTask
    mu             sync.RWMutex
    provider       providers.LLMProvider
    defaultModel   string
    bus            *bus.MessageBus
    workspace      string
    tools          *ToolRegistry
    maxIterations  int
    maxTokens      int           // NEW
    nextID         int
}

func NewSubagentManager(provider providers.LLMProvider, defaultModel, workspace string, bus *bus.MessageBus, maxIter, maxTokens int) *SubagentManager {
    if maxIter == 0 {
        maxIter = 10
    }
    if maxTokens == 0 {
        maxTokens = 4096
    }
    return &SubagentManager{
        tasks:         make(map[string]*SubagentTask),
        provider:      provider,
        defaultModel:  defaultModel,
        bus:           bus,
        workspace:     workspace,
        tools:         NewToolRegistry(),
        maxIterations: maxIter,
        maxTokens:     maxTokens,
        nextID:        1,
    }
}
```

2. Replace hard-coded `max_tokens: 4096` with `sm.maxTokens` (lines 134, 292)

3. Update `agent/loop.go` to pass config values when creating subagent manager:
```go
subagentManager := tools.NewSubagentManager(
    provider, 
    cfg.Agents.Defaults.Model, 
    workspace, 
    msgBus,
    cfg.Execution.Subagent.MaxIterations,
    cfg.Execution.Subagent.MaxTokens,
)
```

### Phase 5: Update Tool Loop

**File:** `pkg/tools/toolloop.go`

1. Update `ToolLoopConfig` to accept max_tokens:
```go
type ToolLoopConfig struct {
    Provider      providers.LLMProvider
    Model         string
    Tools         *ToolRegistry
    MaxIterations int
    MaxTokens     int            // NEW
    LLMOptions    map[string]any
}
```

2. Use `config.MaxTokens` in default options:
```go
llmOpts := config.LLMOptions
if llmOpts == nil {
    maxTokens := config.MaxTokens
    if maxTokens == 0 {
        maxTokens = 4096
    }
    llmOpts = map[string]any{
        "max_tokens":  maxTokens,
        "temperature": 0.7,
    }
}
```

## Backward Compatibility

### Migration Strategy

1. **Keep existing `agents.defaults.max_tool_iterations`** for backward compatibility
2. **New `execution.max_iterations`** takes precedence if set
3. **Fallback chain:**
   - `execution.max_iterations` → `agents.defaults.max_tool_iterations` → default (50)

### Config Migration

Update `pkg/migrate/config.go` to handle migration:
```go
// If execution section doesn't exist, create it from legacy values
if _, ok := migrated["execution"]; !ok {
    migrated["execution"] = map[string]interface{}{
        "max_iterations": defaults["max_tool_iterations"],
        "max_tokens":     defaults["max_tokens"],
        "llm_timeout":    120,
    }
}
```

## Environment Variables

All new parameters support environment variable overrides:

| Parameter | Environment Variable |
|-----------|---------------------|
| `execution.max_iterations` | `PICOCLAW_EXECUTION_MAX_ITERATIONS` |
| `execution.max_tokens` | `PICOCLAW_EXECUTION_MAX_TOKENS` |
| `execution.llm_timeout` | `PICOCLAW_EXECUTION_LLM_TIMEOUT` |
| `execution.subagent.max_iterations` | `PICOCLAW_EXECUTION_SUBAGENT_MAX_ITERATIONS` |
| `execution.subagent.max_tokens` | `PICOCLAW_EXECUTION_SUBAGENT_MAX_TOKENS` |
| `execution.summarization.max_tokens` | `PICOCLAW_EXECUTION_SUMMARIZATION_MAX_TOKENS` |
| `execution.summarization.timeout` | `PICOCLAW_EXECUTION_SUMMARIZATION_TIMEOUT` |

## Updated Example Configuration

**File:** `config/config.example.json`

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "provider": "zhipu",
      "model": "glm-4.7",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "execution": {
    "max_iterations": 50,
    "max_tokens": 8192,
    "llm_timeout": 120,
    "subagent": {
      "max_iterations": 10,
      "max_tokens": 4096
    },
    "summarization": {
      "max_tokens": 1024,
      "timeout": 120
    }
  },
  "channels": {
    "telegram": {
      "enabled": false,
      "token": "YOUR_TELEGRAM_BOT_TOKEN",
      "proxy": "",
      "allow_from": ["YOUR_USER_ID"]
    },
    "qq": {
      "enabled": false,
      "uin": 0,
      "password": "",
      "sign_server": "",
      "allow_from": []
    }
  },
  "providers": {
    "zhipu": {
      "api_key": "YOUR_ZHIPU_API_KEY",
      "api_base": ""
    },
    "openai": {
      "api_key": "YOUR_OPENAI_API_KEY",
      "api_base": ""
    },
    "shengsuanyun": {
      "api_key": "YOUR_SHENGSUANYUN_API_KEY",
      "api_base": ""
    },
    "deepseek": {
      "api_key": "YOUR_DEEPSEEK_API_KEY",
      "api_base": ""
    },
    "github_copilot": {
      "api_key": "",
      "api_base": ""
    }
  },
  "tools": {
    "web": {
      "brave": {
        "api_key": "YOUR_BRAVE_API_KEY",
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
  "devices": {
    "enabled": false,
    "monitor_usb": true
  },
  "maixcam": {
    "enabled": false,
    "host": "192.168.1.100",
    "port": 18791
  },
  "gateway": {
    "host": "0.0.0.0",
    "port": 18790
  }
}
```

> **Note:** The MaixCam port (18791) is intentionally different from the Gateway port (18790) to avoid port conflicts.

## Configuration Validation

### Critical Validation Rules

#### 1. Provider Field Consistency

**Issue:** When `model` is set but `provider` is empty, the system relies on model-name detection which may fail silently.

**Solution:**
```go
func (c *Config) Validate() error {
    // If model is set but provider is empty, try to infer provider
    if c.Agents.Defaults.Model != "" && c.Agents.Defaults.Provider == "" {
        provider := inferProviderFromModel(c.Agents.Defaults.Model)
        if provider == "" {
            return fmt.Errorf("cannot infer provider for model '%s', please set 'provider' field explicitly", c.Agents.Defaults.Model)
        }
        c.Agents.Defaults.Provider = provider
    }
    
    // Validate that provider credentials are configured
    if c.Agents.Defaults.Provider != "" {
        if !c.hasProviderCredentials(c.Agents.Defaults.Provider) {
            return fmt.Errorf("no API key configured for provider '%s'", c.Agents.Defaults.Provider)
        }
    }
    return nil
}
```

#### 2. Port Conflict Detection

**Issue:** MaixCam and Gateway both defaulted to port 18790, causing conflicts.

**Solution:**
- Changed MaixCam default port to `18791`
- Add startup validation:
```go
func (c *Config) Validate() error {
    // Check for port conflicts
    ports := make(map[int]string)
    
    if c.Gateway.Port > 0 {
        if existing, ok := ports[c.Gateway.Port]; ok {
            return fmt.Errorf("port conflict: gateway and %s both use port %d", existing, c.Gateway.Port)
        }
        ports[c.Gateway.Port] = "gateway"
    }
    
    if c.MaixCam.Enabled && c.MaixCam.Port > 0 {
        if existing, ok := ports[c.MaixCam.Port]; ok {
            return fmt.Errorf("port conflict: maixcam and %s both use port %d", existing, c.MaixCam.Port)
        }
        ports[c.MaixCam.Port] = "maixcam"
    }
    return nil
}
```

### Value Range Validation

#### Temperature Validation

```go
func validateTemperature(temp float64) error {
    if temp < 0.0 || temp > 2.0 {
        return fmt.Errorf("temperature must be between 0.0 and 2.0, got %f", temp)
    }
    return nil
}
```

#### MaxToolIterations Validation

```go
func validateMaxToolIterations(iter int) error {
    if iter < 1 {
        return fmt.Errorf("max_tool_iterations must be at least 1, got %d (value of 0 would cause agent to return empty response immediately)", iter)
    }
    if iter > 1000 {
        return fmt.Errorf("max_tool_iterations seems unreasonably high (%d), max recommended is 1000", iter)
    }
    return nil
}
```

#### Heartbeat Interval Validation

```go
func validateHeartbeatInterval(interval int) error {
    if interval < 5 {
        return fmt.Errorf("heartbeat interval must be at least 5 minutes, got %d", interval)
    }
    return nil
}
```

### Empty Value Handling

#### Empty AllowFrom Behavior

**Definition:** When `allow_from` is an empty array `[]`:
- For Telegram/QQ channels: **Allow nobody** (block all requests)
- Use `["*"]` or `["all"]` to explicitly allow everyone

```go
func (c *ChannelConfig) IsAllowed(userID string) bool {
    if len(c.AllowFrom) == 0 {
        // Empty array = allow nobody (secure default)
        return false
    }
    for _, allowed := range c.AllowFrom {
        if allowed == "*" || allowed == "all" || allowed == userID {
            return true
        }
    }
    return false
}
```

#### Workspace Path Validation

```go
func validateWorkspace(path string) error {
    if path == "" {
        return fmt.Errorf("workspace path cannot be empty")
    }
    
    // Expand environment variables
    path = os.ExpandEnv(path)
    
    // Validate after expansion
    if path == "" {
        return fmt.Errorf("workspace path cannot be empty after environment variable expansion")
    }
    
    return nil
}

// Enhanced expandHome handles:
// - ~username syntax (unsupported, returns error)
// - $HOME/workspace syntax (via os.ExpandEnv)
func expandHome(path string) (string, error) {
    if path == "" {
        return "", fmt.Errorf("path is empty")
    }
    
    // First expand environment variables
    path = os.ExpandEnv(path)
    
    if len(path) > 0 && path[0] == '~' {
        // Check for ~username syntax
        if len(path) > 1 && path[1] != '/' {
            return "", fmt.Errorf("~username syntax not supported, use full path or ~/ for current user")
        }
        home, err := os.UserHomeDir()
        if err != nil {
            return "", fmt.Errorf("cannot determine home directory: %w", err)
        }
        if len(path) == 1 {
            return home, nil
        }
        return home + path[1:], nil
    }
    return path, nil
}
```

### Proxy URL Validation

```go
func validateProxyURL(proxy string) error {
    if proxy == "" {
        return nil // Empty is valid (no proxy)
    }
    
    _, err := url.Parse(proxy)
    if err != nil {
        return fmt.Errorf("invalid proxy URL '%s': %w", proxy, err)
    }
    return nil
}
```

### User ID Precision Warning

**Issue:** Large user IDs (e.g., Telegram user IDs) may lose precision when passed as JSON numbers.

**Documentation:**
```markdown
### Important: User ID Format

When configuring `allow_from` in channel configs, always use **string format** for user IDs to avoid precision loss:

❌ Incorrect (may lose precision for large IDs):
"allow_from": [12345678901234567890]

✅ Correct:
"allow_from": ["12345678901234567890"]

JavaScript/JSON numbers are limited to 2^53-1 precision. User IDs should be strings.
```

## Testing Plan

### Unit Tests

1. **Config Loading Tests**
   - Test default values when execution section is missing
   - Test environment variable overrides
   - Test backward compatibility with old config format
   - Test validation errors for invalid configurations

2. **Agent Loop Tests**
   - Verify max_iterations is respected
   - Verify max_tokens is used in LLM calls
   - Verify timeout works correctly
   - Test max_tool_iterations = 0 returns error (not silent empty response)
   - Test max_tool_iterations = 1 works correctly

3. **Subagent Tests**
   - Verify subagent uses separate config values
   - Verify subagent max_iterations and max_tokens

4. **Validation Tests**
   - Test temperature validation (negative, > 2.0, boundary values)
   - Test workspace empty string rejection
   - Test port conflict detection
   - Test provider credential validation
   - Test heartbeat interval minimum enforcement
   - Test proxy URL validation

### Integration Tests

1. **End-to-End Tests**
   - Test agent execution with various config combinations
   - Test timeout behavior under load
   - Test subagent execution limits

## Documentation Updates

1. Update `README.md` with new configuration options
2. Add configuration examples for different use cases:
   - Fast responses (low max_iterations, low max_tokens)
   - Complex tasks (high max_iterations, high max_tokens)
   - Resource-constrained environments (low timeouts)

## Estimated Effort

- **Phase 1 (Config Schema):** 2 hours
- **Phase 2 (Agent Loop):** 3 hours
- **Phase 3 (HTTP Provider):** 1 hour
- **Phase 4 (Subagent):** 2 hours
- **Phase 5 (Tool Loop):** 1 hour
- **Testing:** 3 hours
- **Documentation:** 1 hour

**Total:** ~13 hours

## Benefits

1. **Flexibility:** Users can tune performance for their specific use cases
2. **Resource Management:** Better control over API costs and timeouts
3. **Debugging:** Easier to test with lower iteration counts
4. **Scalability:** Configure different limits for main agent vs subagents
5. **Backward Compatible:** Existing configs continue to work

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking existing configs | Comprehensive migration logic and fallbacks |
| Confusion between old and new parameters | Clear documentation and deprecation warnings |
| Performance regression | Thorough benchmarking before/after |
| Environment variable conflicts | Namespaced environment variables |
| Provider field left empty | Auto-infer provider from model name, validate credentials at startup |
| Port conflicts (MaixCam vs Gateway) | Different default ports (18791 vs 18790), startup validation |
| Empty allow_from security risk | Document behavior: empty = deny all, use ["*"] for allow all |
| max_tool_iterations = 0 silent failure | Add validation requiring minimum value of 1 |
| Temperature out of range | Add validation: 0.0 <= temperature <= 2.0 |
| Large user IDs losing precision | Document requirement to use string format for IDs |
| Invalid proxy URLs | Add validation with clear error messages |
| Empty workspace path | Add validation requiring non-empty path |

## Conclusion

This proposal adds essential configurability to PicoClaw while maintaining backward compatibility. The new `execution` configuration section provides clear, centralized control over LLM execution parameters, making it easier for users to optimize PicoClaw for their specific needs.

## Next Steps

1. Review and approve this proposal
2. Implement Phase 1 (Configuration Schema)
3. Implement remaining phases in order
4. Update example configuration
5. Add comprehensive tests
6. Update documentation
7. Create migration guide for existing users
