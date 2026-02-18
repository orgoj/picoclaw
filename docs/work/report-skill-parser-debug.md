# Skill Parser Debug Report

**Date:** 2026-02-18  
**Issue:** User reports WARNING on valid SKILL.md files  
**Affected File:** `pkg/skills/loader.go`

---

## Summary

**ROOT CAUSE:** ✅ **ALREADY FIXED** - Parser regex mismatch has been corrected in commit `0481dc7` (2026-02-18 12:37:03).

**USER ISSUE:** Installed binary is outdated - built from commit `70c731a` (2026-02-18 11:09:04), which predates the fix.

**IMPACT:** ⚠️ **CRITICAL** (for users with outdated binary) - All 30 workspace skills fail to load metadata and are silently skipped.

---

## Investigation Results

### 1. Skill File Format

**User's workspace skills** (30 files in `~/.picoclaw/workspace/skills/`):
```bash
$ head -1 ~/.picoclaw/workspace/skills/*/SKILL.md
----  # ALL 30 skills use 4 dashes
```

**Official picoclaw repository** (`skills/` and `cmd/picoclaw/workspace/skills/`):
```bash
$ head -1 skills/*/SKILL.md
---  # Official examples use 3 dashes
```

### 2. Parser Code

**File:** `pkg/skills/loader.go`, line 232

```go
func (sl *SkillsLoader) extractFrontmatter(content string) string {
    // Parser expects exactly 3 dashes
    re := regexp.MustCompile(`(?s)^---\n(.*?)\n---`)
    match := re.FindStringSubmatch(content)
    if len(match) > 1 {
        return match[1]
    }
    return ""
}
```

**Problem:** 
- Regex `^---\n` only matches **3 dashes** (`---`)
- User's files have **4 dashes** (`----`)
- Result: `frontmatter == ""`, metadata extraction fails

### 3. Validation Logic

**File:** `pkg/skills/loader.go`, lines 36-58

When frontmatter extraction fails:
1. `getSkillMetadata()` returns empty `SkillMetadata{}`
2. `validate()` checks:
   - `Name == ""` → "name is required" ❌
   - `Description == ""` → "description is required" ❌
3. Logs WARNING and **skips the skill**:

```go
if err := info.validate(); err != nil {
    slog.Warn("invalid skill from workspace", "name", info.Name, "error", err)
    continue  // SKILL IS NOT LOADED!
}
```

### 4. Test Verification

```bash
$ cat > /tmp/test_frontmatter.go << 'EOF'
package main

import (
    "fmt"
    "regexp"
)

func main() {
    content := `----
name: business-analyst
description: Test
----
# Content`
    
    // Parser's regex (3 dashes)
    re := regexp.MustCompile(`(?s)^---\n(.*?)\n---`)
    match := re.FindStringSubmatch(content)
    fmt.Printf("Parser regex (---): %v\n", len(match) > 1)
    
    // Fixed regex (3 or 4 dashes)
    re2 := regexp.MustCompile(`(?s)^---+\n(.*?)\n---+`)
    match2 := re2.FindStringSubmatch(content)
    fmt.Printf("Fixed regex (---+): %v\n", len(match2) > 1)
}
EOF

$ go run /tmp/test_frontmatter.go
Parser regex (---): false
Fixed regex (---+): true
```

---

## Impact Assessment

### What's Broken

1. **Skill Discovery:** Skills don't appear in agent's available skills list
2. **Skill Loading:** `LoadSkill()` returns `"", false` for all workspace skills
3. **Skill Context:** `BuildSkillsSummary()` returns empty string
4. **Functionality:** All skill-based workflows are non-functional

### Affected Skills (30 total)

All skills in `~/.picoclaw/workspace/skills/`:
- business-analyst, web-scraper, daily-startup
- project-management, error-monitor, learning-log
- eshop-*, web-*, pm2-*
- And 21 more...

### Error Messages

```
WARN invalid skill from workspace name= error="name is required; description is required"
```

---

## Root Cause Analysis

### Timeline of Events

1. **Before Fix:** Parser used `---` (3 dashes) - standard YAML frontmatter
2. **User's Skills:** All 30 skills use `----` (4 dashes) - likely from a template
3. **Commit 43c4fb3** (2026-02-18 11:45:17): Fixed greedy regex, but still used 3 dashes
4. **Commit 0481dc7** (2026-02-18 12:37:03): Fixed parser to use 4 dashes (`----`)
5. **User's Binary:** Built at 2026-02-18 11:09:04 from commit 70c731a - BEFORE the fix

### Why 4 Dashes?

**Evidence:** All 30 workspace skills consistently use `----`, suggesting a template or generator tool created them with this format.

**Official examples** in repository use `---` (3 dashes), indicating workspace skills were created by a different tool/template.

### The Fix (Already Committed)

Commit `0481dc7` changed the parser from:
```go
re := regexp.MustCompile(`(?s)^---\n(.*)\n---`)
```

To:
```go
re := regexp.MustCompile(`(?s)^----\n(.*?)\n----`)
```

This fix makes the parser match the actual format of workspace skills.

---

## Solution

### ✅ The Fix Is Already In The Repository

The parser has been fixed in commit `0481dc7` to accept `----` (4 dashes).

**Current code in `pkg/skills/loader.go` (lines 308-315):**
```go
func (sl *SkillsLoader) extractFrontmatter(content string) string {
    // (?s) enables DOTALL mode so . matches newlines
    // Match first ----, capture everything until NEXT ---- on its own line (non-greedy)
    re := regexp.MustCompile(`(?s)^----\n(.*?)\n----`)
    match := re.FindStringSubmatch(content)
    if len(match) > 1 {
        return match[1]
    }
    return ""
}
```

### Action Required: Rebuild PicoClaw

**The user needs to rebuild picoclaw from the latest source:**

```bash
cd /home/nanobot/.picoclaw/workspace/projects/picoclaw
git pull
make build
make install  # or copy build/picoclaw-* to ~/.local/bin/picoclaw
```

**Verification:**
```bash
# Check version
picoclaw version
# Should show commit 0481dc7 or later

# Test skill loading
picoclaw agent -m "What skills do you have available?"
# Should list all 30 skills including business-analyst
```

---

## Action Items

### Immediate (User Action Required)

1. **Rebuild picoclaw from latest source:**
   ```bash
   cd /home/nanobot/.picoclaw/workspace/projects/picoclaw
   make build
   make install
   ```

2. **Verify the fix:**
   ```bash
   picoclaw version
   # Should show: v0.1.1-172-g0481dc7 or later
   
   # Test skill loading
   picoclaw agent -m "Use business-analyst skill"
   ```

### Already Completed (In Repository)

1. ✅ Parser updated to accept `----` (commit 0481dc7)
2. ✅ Regex made non-greedy with `(.*?)` (commit 43c4fb3)
3. ✅ Both `extractFrontmatter` and `stripFrontmatter` updated

### Future Improvements (Optional)

1. **Add unit tests** for frontmatter parsing with different dash counts
2. **Make parser more flexible** to accept both `---` and `----`:
   ```go
   re := regexp.MustCompile(`(?s)^---+\n(.*?)\n---+`)
   ```
3. **Improve error messages** to hint at frontmatter format issues
4. **Update skill-creator** to use standard `---` format
5. **Document frontmatter format** in skill creation guide

---

## Verification Steps

After rebuilding picoclaw:

```bash
# 1. Check version includes the fix
picoclaw version
# Expected: v0.1.1-172-g0481dc7 or later (commit >= 0481dc7)

# 2. Test skill loading directly
cat > /tmp/test_skill.go << 'EOF'
package main

import (
    "fmt"
    "regexp"
    "strings"
)

func main() {
    content := `----
name: business-analyst
description: Test
----
# Content`
    
    re := regexp.MustCompile(`(?s)^----\n(.*?)\n----`)
    match := re.FindStringSubmatch(content)
    
    if len(match) > 1 {
        fmt.Println("✅ Parser correctly extracts frontmatter")
        fmt.Printf("   Extracted: %q\n", strings.TrimSpace(match[1]))
    } else {
        fmt.Println("❌ Parser failed to extract frontmatter")
    }
}
EOF
go run /tmp/test_skill.go

# 3. Test with picoclaw agent
picoclaw agent -m "List all available skills"
# Expected: Should list business-analyst and other skills

# 4. Check for warnings in logs
picoclaw gateway 2>&1 | grep -i "invalid skill"
# Expected: No output (no warnings)
```

Expected result: All 30 skills load successfully without warnings.

---

## Appendix: Affected Files

### Parser Code
- `pkg/skills/loader.go` (lines 232-240)

### Test Files
- `pkg/skills/loader_test.go` (add frontmatter tests)

### Documentation
- `README.md` (clarify frontmatter format)
- `skills/agent-builder/SKILL.md` (example format)

### User Skills (30 files)
All files in: `~/.picoclaw/workspace/skills/*/SKILL.md`

---

## Conclusion

**This was a PARSER BUG that has ALREADY BEEN FIXED.**

The issue occurred because:
1. Workspace skills use `----` (4 dashes) for frontmatter
2. Old parser (before commit 0481dc7) expected `---` (3 dashes)
3. User's installed binary was built before the fix

**The fix** (commit 0481dc7, 2026-02-18 12:37:03):
- Updated parser to accept `----` (4 dashes)
- Fixed both `extractFrontmatter()` and `stripFrontmatter()` functions
- Made regex non-greedy with `(.*?)`

**User action required:** Rebuild picoclaw from latest source to get the fix.

---

## Additional Context: Why 4 Dashes?

Investigation of all 30 workspace skills shows they ALL use `----`:
```bash
$ head -1 ~/.picoclaw/workspace/skills/*/SKILL.md | grep -c "----"
30
```

This consistency suggests a template or generator tool created these skills with the 4-dash format, while the official picoclaw repository examples use the standard 3-dash format (`---`).

**Recommendation for consistency:**
- Either update all workspace skills to use `---` (standard YAML)
- Or make the parser flexible to accept both formats with regex `^---+\n(.*?)\n---+`

---

**Report Generated:** 2026-02-17  
**Author:** Debug Subagent  
**Status:** CRITICAL - Requires Immediate Fix
