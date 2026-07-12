#!/usr/bin/env bash
# ci-audit.sh — Role-specific PR audit via opencode-go API.
# Reads agent personas from .pi/agents/ and applies each lens to the PR diff.
# Uses opencode-go (OpenAI-compatible API). Auth via OPENCODE_API_KEY.
# Posts collated findings as a GitHub PR comment.
#
# Prerequisites: gh CLI, curl, jq
# Env: OPENCODE_API_KEY, PR_NUMBER, GITHUB_REPOSITORY

set -euo pipefail

OPENCODE_API="https://opencode.ai/zen/go/v1/chat/completions"
OPENCODE_MODEL="${OPENCODE_MODEL:-deepseek-v4-flash}"

# ── Helpers ──────────────────────────────────────────────────────────────────

extract_prompt() {
  # Extract system prompt from agent file (everything after second ---).
  local file="$1"
  awk 'BEGIN { count=0 }
       /^---$/ { count++; next }
       count >= 2 { print }' "$file"
}

call_opencode() {
  # Call opencode-go API with system prompt + user message.
  local system_prompt="$1"
  local user_message="$2"

  local payload
  payload=$(jq -n \
    --arg model "$OPENCODE_MODEL" \
    --arg system "$system_prompt" \
    --arg user "$user_message" \
    '{
      model: $model,
      messages: [
        {role: "system", content: $system},
        {role: "user", content: $user}
      ],
      temperature: 0.1,
      max_tokens: 8000
    }')

  local response
  response=$(curl -s -S --fail-with-body "$OPENCODE_API" \
    -H "Authorization: Bearer $OPENCODE_API_KEY" \
    -H "Content-Type: application/json" \
    -d "$payload" 2>&1) || {
    echo "::warning::opencode API call failed: $response"
    return 1
  }

  # DeepSeek models put reasoning in reasoning_content, final answer in content.
  # Content may be empty if max_tokens consumed by reasoning; fall back gracefully.
  local content
  content=$(echo "$response" | jq -r '.choices[0].message.content // ""')
  if [ -z "$content" ] || [ "$content" = "null" ]; then
    echo "::warning::Empty content from API — may need higher max_tokens or non-reasoning model"
    echo "$response" | jq -r '.choices[0].message.reasoning_content // "No content returned"'
    return 1
  fi
  echo "$content"
}

# ── Agent lookup (bash 3.2+ compatible) ─────────────────────────────────────

agent_file() {
  case "$1" in
  cto) echo ".pi/agents/cto.md" ;;
  boutique) echo ".pi/agents/boutique-director.md" ;;
  compliancy) echo ".pi/agents/compliancy.md" ;;
  ux) echo ".pi/agents/ux-expert.md" ;;
  ai-engineer) echo ".pi/agents/ai-engineer.md" ;;
  *) echo "" ;;
  esac
}

agent_label() {
  case "$1" in
  cto) echo "**CTO** (scale, multi-tenancy, reliability)" ;;
  boutique) echo "**Boutique Director** (small team, personalization, reputation)" ;;
  compliancy) echo "**Compliancy Advisor** (GDPR, PII, data protection)" ;;
  ux) echo "**UX Expert** (power-user speed, keyboard, accessibility)" ;;
  ai-engineer) echo "**AI/ML Engineer** (MCP servers, LangGraph, AI architecture)" ;;
  *) echo "" ;;
  esac
}

# ── Main ─────────────────────────────────────────────────────────────────────

main() {
  if [ -z "${PR_NUMBER:-}" ]; then
    echo "::error::PR_NUMBER not set"
    exit 1
  fi

  if [ -z "${OPENCODE_API_KEY:-}" ]; then
    echo "::error::OPENCODE_API_KEY not set"
    exit 1
  fi

  echo "::group::Fetching PR diff"
  DIFF=$(gh pr diff "$PR_NUMBER" --repo "$GITHUB_REPOSITORY" 2>&1) || {
    echo "::error::Failed to get PR diff: $DIFF"
    exit 1
  }

  if [ -z "$DIFF" ]; then
    echo "::warning::Empty diff — nothing to audit"
    exit 0
  fi

  # Truncate diff to ~15000 chars to stay within token limits
  DIFF_TRUNCATED=$(echo "$DIFF" | head -c 15000)
  echo "Diff size: $(echo "$DIFF" | wc -c) bytes (truncated to 15000 for audit)"
  echo "::endgroup::"

  # Agent list: env var AUDIT_AGENTS overrides default (cto,boutique,compliancy,ux).
  IFS=',' read -ra AUDIT_AGENTS_ARR <<<"${AUDIT_AGENTS:-cto,boutique,compliancy,ux}"

  RESULTS=""
  AUDITED=""
  for agent in "${AUDIT_AGENTS_ARR[@]}"; do
    agent="${agent// /}" # trim whitespace
    local agent_file
    agent_file=$(agent_file "$agent")
    local agent_label
    agent_label=$(agent_label "$agent")

    if [ -z "$agent_file" ] || [ ! -f "$agent_file" ]; then
      echo "::warning::Agent '$agent' not found — skipping"
      continue
    fi

    echo "::group::Auditing with $agent_label"

    SYSTEM_PROMPT=$(extract_prompt "$agent_file")
    if [ -z "$SYSTEM_PROMPT" ]; then
      echo "::warning::Empty system prompt for $agent — skipping"
      echo "::endgroup::"
      continue
    fi

    USER_MSG="Audit this PR diff from the yop-pms hotel PMS. Output ONLY a findings table — no narration, no thinking out loud, no 'let me analyze', no internal monologue. Format:

**Finding N: TITLE** — SEVERITY
*File:* path, line N
*Issue:* One sentence.
*Fix:* One sentence.

Keep under 500 words. Be direct.

PR diff:
\`\`\`diff
$DIFF_TRUNCATED
\`\`\`"

    FINDINGS=$(call_opencode "$SYSTEM_PROMPT" "$USER_MSG") || {
      echo "::endgroup::"
      continue
    }

    RESULTS="$RESULTS

---

### $agent_label

$FINDINGS"
    AUDITED="$AUDITED, $agent_label"
    echo "::endgroup::"
  done

  # Strip leading comma + space
  AUDITED="${AUDITED#, }"
  AUDITED="${AUDITED#,}"

  if [ -z "$AUDITED" ]; then
    echo "::warning::No agents audited successfully"
    exit 0
  fi

  # ── Compile comment ────────────────────────────────────────────────────

  COMMENT="## 🔍 Role Audit — PR #${PR_NUMBER}

Automated multi-perspective audit against this PR diff.

$RESULTS

---

> 🤖 Generated by [ci-audit.sh](scripts/ci-audit.sh) via opencode-go.
> Agents run:$AUDITED.
> Audit personas live in \`.pi/agents/\`. To run locally: \`/run-audit full \"context\"\`"

  # ── Post comment ───────────────────────────────────────────────────────

  echo "::group::Posting PR comment"
  echo "$COMMENT" | gh pr comment "$PR_NUMBER" --repo "$GITHUB_REPOSITORY" --body-file - 2>&1 || {
    echo "::error::Failed to post PR comment"
    exit 1
  }
  echo "::endgroup::"

  echo "✅ Audit complete for PR #${PR_NUMBER}"
}

main "$@"
