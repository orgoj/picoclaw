# P0-1 COMPLETE: API Retry Logic Fix

## Summary
Fixed API retry logic in `pkg/agent/loop.go` to handle 5xx errors and rate limits properly.

## Changes Made

### 1. Extended Retryable Errors
Added 5xx server errors to the retryable condition:
```go
strings.Contains(errStr, "status=5") ||
strings.Contains(errStr, "500") ||
strings.Contains(errStr, "502") ||
strings.Contains(errStr, "503") ||
strings.Contains(errStr, "504")
```

### 2. Rate Limit (429) Special Handling
```go
isRateLimit := strings.Contains(errStr, "429") ||
    strings.Contains(errStr, "rate limit") ||
    strings.Contains(errStr, "too many requests")

if isRateLimit && retry < maxRetries {
    waitTime := time.Duration(10*(retry+1)) * time.Second // 10s, 20s, 30s
    // ... logging
    time.Sleep(waitTime)
    continue
}
```

### 3. Exponential Backoff for Other Errors
Changed from linear (`1s, 2s, 3s`) to exponential (`2s, 4s, 8s`):
```go
waitTime := time.Duration(2<<(retry)) * time.Second
```

## Test Results
- ✅ Build: SUCCESS
- ✅ Tests: All packages pass
- ✅ Commit: `f67fe85`

## File Modified
- `pkg/agent/loop.go` (lines 521-577)

## Commit
```
fix: improve API retry logic with 5xx support and exponential backoff
```
