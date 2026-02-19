# Report: P0-1 Telegram Error Handling Enhancement

## Summary
Enhanced the `formatErrorMessage` function in `pkg/agent/loop.go` to provide more specific error messages and inject error details into session history.

## Changes Made

### 1. `pkg/agent/loop.go`

**Function Signature Update:**
```go
// Before
func (al *AgentLoop) formatErrorMessage(err error) string

// After  
func (al *AgentLoop) formatErrorMessage(err error, sessionKey string) string
```

**Error Message Mapping:**
| Error Type | Pattern Matched | User Message |
|------------|-----------------|--------------|
| Timeout | `timeout`, `context deadline exceeded` | ⏱️ API timeout (z.ai) - please try again |
| Rate Limit | `429`, `rate limit`, `too many requests` | 🚦 API rate limited - waiting... |
| Server Error | `status=5`, `500`, `502`, `503`, `504` | ⚠️ API error: status=XXX - temporary issue |
| Connection Refused | `connection refused` | 🔌 Network error: connection refused |
| Network Unreachable | `network is unreachable` | 🔌 Network error: network unreachable |
| Connection Interrupted | `unexpected EOF`, `connection reset` | 🔌 Network error: connection interrupted |
| Generic API Error | `API request failed` | ⚠️ API request failed - please try again |
| LLM Error | `LLM call failed` | ⚠️ Failed to communicate with AI service |
| Fallback | Any other error | ⚠️ An error occurred: [error] |

**Session History Injection:**
```go
// Inject error details into session history so agent knows what happened
if sessionKey != "" && errorDetails != "" {
    al.sessions.AddMessage(sessionKey, "system", errorDetails)
}
```

### 2. `pkg/agent/loop_error_test.go`
- Updated test cases to match new specific error messages
- Added test for session key parameter
- Added new test cases: rate limit (429), server errors (503), network unreachable

## Testing
All 12 tests pass:
```
=== RUN   TestFormatErrorMessage
    --- PASS: TestFormatErrorMessage/timeout_error
    --- PASS: TestFormatErrorMessage/unexpected_EOF
    --- PASS: TestFormatErrorMessage/connection_reset
    --- PASS: TestFormatErrorMessage/rate_limit_429
    --- PASS: TestFormatErrorMessage/server_error_500
    --- PASS: TestFormatErrorMessage/server_error_503
    --- PASS: TestFormatErrorMessage/connection_refused
    --- PASS: TestFormatErrorMessage/network_unreachable
    --- PASS: TestFormatErrorMessage/LLM_call_failed
    --- PASS: TestFormatErrorMessage/generic_error
```

## Commit
```
c363ced fix: improve API error messages with specific details
```

## Verification
- ✅ Code compiles (`go build ./...`)
- ✅ All tests pass (`go test ./pkg/agent/...`)
- ✅ No breaking changes (existing functionality preserved)
- ✅ Panic recovery still works (unchanged)
