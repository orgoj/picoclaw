#!/bin/bash
# analyze-agent-failures.sh
# Analyzes agent.log and FAILURES.md to identify recurring issues.

BASE_DIR="/home/michael/projects/picoclaw/picoclaw/nanobotnb/.picoclaw"
LOG_FILE="$BASE_DIR/logs/agent.log"
FAILURES_FILE="$BASE_DIR/workspace/memory/FAILURES.md"

echo "### Analysis of Agent Failures ###"

if [ -f "$LOG_FILE" ]; then
    # Show version of the current log session
    VERSION_INFO=$(grep "\"message\":\"PicoClaw started\"" "$LOG_FILE" | tail -n 1)
    if [ -n "$VERSION_INFO" ]; then
        echo "#### Current Session Info ####"
        echo "$VERSION_INFO" | sed 's/.*"fields":{\([^}]*\)}.*/\1/' | tr ',' '\n'
        echo "----------------------------"
    fi

    echo "#### Top Errors from current agent.log ####"
    grep "\"level\":\"ERROR\"" "$LOG_FILE" | sed 's/.*"error":"\([^"]*\)".*/\1/' | sort | uniq -c | sort -nr | head -n 10
fi

if [ -f "$FAILURES_FILE" ]; then
    echo -e "\n#### Recent Entries in FAILURES.md ####"
    tail -n 20 "$FAILURES_FILE"
fi

echo -e "\n#### Active Sessions with Issues ####"
grep -l "ERROR" "$BASE_DIR/workspace/sessions/"*.json 2>/dev/null | xargs -n1 basename | head -n 5
