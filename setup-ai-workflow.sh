#!/usr/bin/env bash
# setup-ai-workflow.sh
# Run from template-go-echo-squirrel root

set -euo pipefail

REPO_ROOT="$(pwd)"
AI_DIR="$REPO_ROOT/.ai-workflow"
ADR_DIR="$REPO_ROOT/docs/adr"
GLOBAL_DIR="$HOME/.ai-workflow/global"

echo "=========================================="
echo " AI Workflow Setup"
echo " Repo: $REPO_ROOT"
echo "=========================================="
echo ""

echo "[1/6] Creating directories..."
mkdir -p "$AI_DIR/templates"
mkdir -p "$AI_DIR/prompts"
mkdir -p "$AI_DIR/features"
mkdir -p "$ADR_DIR"
mkdir -p "$GLOBAL_DIR"
touch "$AI_DIR/features/.gitkeep"
echo "      OK"
echo ""

echo "[2/6] Writing templates..."

cat > "$AI_DIR/templates/spec.md" << 'EOF'
# Spec: [Feature Name]

**Date:** YYYY-MM-DD
**Author:** [name]
**Status:** Draft | Approved

---

## Problem

[Anong problema ang sinasolve? 1-2 sentences.]

## Solution

[Anong approach? 1-2 sentences.]

## Behaviors

1. [Behavior 1]
2. [Behavior 2]
3. [Behavior 3]

## Edge Cases

- [Edge case 1] -> [expected behavior]
- [Edge case 2] -> [expected behavior]

## Non-Goals

- [Hindi kasama 1]
- [Hindi kasama 2]

## Success Criteria

- [ ] [Criterion 1]
- [ ] [Criterion 2]

## Open Questions

- [ ] [Question 1]
- [ ] [Question 2]
EOF

cat > "$AI_DIR/templates/plan.md" << 'EOF'
# Plan: [Feature Name]

**Spec:** ./spec.md
**Date:** YYYY-MM-DD

---

## Files to Touch

| File | Change | Risk |
|---|---|---|
| internal/service/xxx.go | Add method | Low |
| internal/repository/xxx.go | Add query | Medium |
| internal/handler/xxx.go | Add endpoint | Low |
| internal/database/migrations/NNNN.sql | New table | High |

## Order

1. Migration (DB first)
2. Repository (data access)
3. Service (business logic)
4. Handler (HTTP)
5. Swagger regen

## Risks

- [Risk 1] -> [mitigation]
- [Risk 2] -> [mitigation]

## Test Strategy

- Unit: [what]
- Integration: [what]
- Manual: [what]

## Rollback

[Paano i-rollback kung may problema?]
EOF

cat > "$AI_DIR/templates/tasks.md" << 'EOF'
# Tasks: [Feature Name]

**Plan:** ./plan.md
**Date:** YYYY-MM-DD

---

## T1: [name]

**Files:** [list]
**Test:** [what]
**Commit:** `type(scope): msg`
**Done when:** [criteria]

## T2: [name]

**Files:** [list]
**Test:** [what]
**Commit:** `type(scope): msg`
**Done when:** [criteria]

## T3: [name]

**Files:** [list]
**Test:** [what]
**Commit:** `type(scope): msg`
**Done when:** [criteria]
EOF

cat > "$AI_DIR/templates/checklist.md" << 'EOF'
# Feature Checklist

## Feature: [Name]
**Started:** YYYY-MM-DD
**Branch:** feature/xxx
**ADR:** docs/adr/NNNN-xxx.md (kung may decision)

---

## Phase 1: Understand

- [ ] Grilled -- Q&A log in notes.md, no "I assume" left
- [ ] Spec written (spec.md)
- [ ] Plan written (plan.md)
- [ ] Tickets generated (tasks.md)

## Phase 2: Build

Per ticket: clear context -> load AGENTS + ticket -> test-first -> commit

- [ ] T1: [name] -- context fresh, tests pass
- [ ] T2: [name] -- context fresh, tests pass
- [ ] T3: [name] -- context fresh, tests pass
- [ ] ... (add as needed)

## Phase 3: Verify

- [ ] Fresh-agent self-review
- [ ] Code review: DeepSeek -- findings: [list]
- [ ] Code review: Claude -- findings: [list]
- [ ] Code review: Gemini -- findings: [list]
- [ ] All blockers resolved
- [ ] Converged: 2 consecutive clean reviews

## Phase 4: Ship

- [ ] `make lint && make test` pass
- [ ] Integration tests pass (if repo changed)
- [ ] `make swagger` no diff (if API changed)
- [ ] Commit message follows convention
- [ ] PR opened
- [ ] CI passing
- [ ] Rollback plan documented
- [ ] Merged

---

## Notes

- Context clears: [count] (red flag if > 5)
- Blockers: [list]
- Decisions: [list]
- ADRs created: [list]
EOF

cat > "$AI_DIR/templates/notes.md" << 'EOF'
# Notes: [Feature Name]

**Date:** YYYY-MM-DD

---

## Q&A Log (Grill)

**Q:** [question]
**A:** [answer]

**Q:** [question]
**A:** [answer]

## Decisions

- [Decision 1] -- [rationale]
- [Decision 2] -- [rationale]

## Blockers

- [Blocker 1] -- [status]
- [Blocker 2] -- [status]

## Context Clears

| # | Ticket | Reason | Notes |
|---|---|---|---|
| 1 | T1 | Start | -- |
| 2 | T2 | After T1 commit | -- |

## ADRs Created

- docs/adr/NNNN-xxx.md
EOF

cat > "$AI_DIR/templates/adr.md" << 'EOF'
# ADR NNNN: [Title]

**Status:** Proposed | Accepted | Deprecated | Superseded
**Date:** YYYY-MM-DD
**Deciders:** [names]

---

## Context

[Anong situation ang nag-require ng decision?]

## Decision

[Anong decision? 1-2 sentences.]

## Consequences

### Positive
- [Benefit 1]
- [Benefit 2]

### Negative
- [Drawback 1]
- [Drawback 2]

### Neutral
- [Trade-off 1]

## Alternatives Considered

### [Alternative 1]
- **Why rejected:** [reason]

### [Alternative 2]
- **Why rejected:** [reason]

## References

- [Link 1]
- [Link 2]
EOF

echo "      OK"
echo ""

echo "[3/6] Writing prompts..."

cat > "$AI_DIR/prompts/01-grill.md" << 'EOF'
# Prompt: Grill (Phase 1)

## System
You are a senior Go backend engineer working on template-go-echo-squirrel.
Your job is to ask clarifying questions BEFORE writing any spec.

## Rules
- Read AGENTS.md first
- Ask questions, don't assume
- Cover: edge cases, non-goals, security, performance, backwards compat
- One question at a time, wait for answer
- Don't write code or spec yet

## User Prompt Template
Read AGENTS.md.

I want to add [FEATURE] to template-go-echo-squirrel.

Before writing any spec, ask me questions to clarify:
- What problem does this solve?
- Who are the users?
- What are the edge cases?
- What are the non-goals?
- Security implications?
- Backwards compatibility?

Ask one question at a time. Wait for my answer.
EOF

cat > "$AI_DIR/prompts/02-spec.md" << 'EOF'
# Prompt: Spec (Phase 1)

## System
You are a senior Go backend engineer.
Write a spec using the template. Include non-goals. No code.

## User Prompt Template
Based on our Q&A in notes.md, write spec.md using
.ai-workflow/templates/spec.md.

Requirements:
- Problem: 1-2 sentences
- Solution: 1-2 sentences
- Behaviors: numbered list
- Edge cases: with expected behavior
- Non-goals: at least 2
- Success criteria: checkboxes
- Open questions: kung may natira

Output: complete spec.md content.
EOF

cat > "$AI_DIR/prompts/03-plan.md" << 'EOF'
# Prompt: Plan (Phase 1)

## System
You are a senior Go backend engineer.
Write a plan using the template. List files, order, risks.

## User Prompt Template
Based on spec.md, write plan.md using
.ai-workflow/templates/plan.md.

Requirements:
- Files to touch: table with risk level
- Order: DB -> repository -> service -> handler -> swagger
- Risks: with mitigation
- Test strategy: unit + integration + manual
- Rollback: concrete steps

Follow AGENTS.md ticket-type rules.

Output: complete plan.md content.
EOF

cat > "$AI_DIR/prompts/04-tickets.md" << 'EOF'
# Prompt: Tickets (Phase 1)

## System
You are a senior Go backend engineer.
Break plan into tickets. Each ticket = 1 commit.

## User Prompt Template
Based on plan.md, write tasks.md using
.ai-workflow/templates/tasks.md.

Requirements:
- Each ticket = 1 commit
- Each ticket must have: files, test, commit message, done criteria
- If a ticket can't be described in 1 sentence, split it
- Order: dependency order
- Follow AGENTS.md ticket-type rules

Output: complete tasks.md content.
EOF

cat > "$AI_DIR/prompts/05-build.md" << 'EOF'
# Prompt: Build (Phase 2, per ticket)

## System
You are a senior Go backend engineer.
Implement ONE ticket. Test first, then code.

## Rules
- Follow AGENTS.md strictly
- Test first, then code
- No new deps without ADR
- Run make test before claiming done
- Show diff + test results

## User Prompt Template
Context:
- AGENTS.md (attached)
- tasks.md (T[N] only)
- Files: [list]

Task: Implement T[N].

Rules:
- Test first, then code
- Follow AGENTS.md strictly
- No new deps
- Run make test
- Show diff + test results

Output:
1. Test file (new/changed)
2. Implementation
3. Test results
4. Diff summary
EOF

cat > "$AI_DIR/prompts/06-review.md" << 'EOF'
# Prompt: Review (Phase 3)

## System
You are a senior Go backend engineer reviewing a diff.
Be strict. Tag findings by severity. Don't suggest refactors
outside the diff scope.

## User Prompt Template
Review this diff against AGENTS.md.

Context:
- AGENTS.md (attached)
- Spec: spec.md (attached)
- Diff: [attached]

Review for:
1. Violations of AGENTS.md rules
2. Security issues (auth, SQL injection, CSRF, rate limit)
3. Missing tests
4. Architecture violations (handler doing SQL, etc.)
5. Edge cases not covered
6. Error handling (apperror usage)
7. Context usage (context.Context, cancellation)

Output format:
- BLOCKER: [issue] -- [why] -- [fix]
- SHOULD-FIX: [issue] -- [why] -- [fix]
- NIT: [issue] -- [why]

Do NOT suggest refactors outside the diff scope.
Do NOT repeat findings from previous reviews.
EOF

cat > "$AI_DIR/prompts/07-ship.md" << 'EOF'
# Prompt: Ship (Phase 4)

## System
You are a senior Go backend engineer.
Run pre-ship checklist. Report status.

## User Prompt Template
Pre-ship checklist for [FEATURE]:

1. `make lint` -- pass?
2. `make test` -- pass?
3. `go test -tags=integration ./internal/repository/` -- pass?
4. `make swagger` -- no diff?
5. Commit message follows convention?
6. Rollback plan documented?

Run each. Report:
- Command
- Result (pass/fail)
- Output (kung fail)

Kung lahat pass, suggest commit message.
Kung may fail, suggest fix.
EOF

echo "      OK"
echo ""

echo "[4/6] Writing ADRs..."

cat > "$ADR_DIR/0001-session-auth.md" << 'EOF'
# ADR 0001: Session-Based Auth (not JWT)

**Status:** Accepted
**Date:** 2026-09-17
**Deciders:** [your name]

---

## Context

Kailangan ng auth mechanism para sa API. Dalawang main options: session-based o JWT.

## Decision

Session-based auth gamit ang `alexedwards/scs` + Postgres store.

## Consequences

### Positive
- Revocation instant -- delete session
- CSRF protection natural (double-submit cookie)
- Server-side control -- pwede i-invalidate anytime
- Hindi kailangan ng token refresh logic

### Negative
- Kailangan ng session store (Postgres)
- Hindi stateless -- mas mahirap i-scale horizontally
- May DB round-trip per request

### Neutral
- Cookie-based -- kailangan ng SameSite + HttpOnly

## Alternatives Considered

### JWT
- **Why rejected:** Revocation mahirap, CSRF harder, kailangan ng refresh token rotation

### Cookie-only (no server-side store)
- **Why rejected:** Walang server-side revocation

## References

- internal/session/
- internal/middleware/auth.go
EOF

cat > "$ADR_DIR/0002-postgres-session-store.md" << 'EOF'
# ADR 0002: Postgres Session Store (not in-memory)

**Status:** Accepted
**Date:** 2026-09-17
**Deciders:** [your name]

---

## Context

Ang session store ay pwedeng in-memory (map) o Postgres.

## Decision

Postgres session store gamit ang `scs/postgresstore`.

## Consequences

### Positive
- Persistent across restarts
- Pwede i-share sa multiple instances
- Pwede i-query para sa audit
- May built-in cleanup

### Negative
- DB round-trip per request
- Kailangan ng index sa sessions table
- May cleanup job

### Neutral
- Same DB as main data

## Alternatives Considered

### In-memory (map)
- **Why rejected:** Hindi persistent, hindi scalable

### Redis
- **Why rejected:** Dagdag na dependency, may Postgres na

## References

- internal/session/
- internal/database/migrations/ (sessions table)
EOF

cat > "$ADR_DIR/0003-csrf-double-submit.md" << 'EOF'
# ADR 0003: CSRF Double-Submit Cookie Pattern

**Status:** Accepted
**Date:** 2026-09-17
**Deciders:** [your name]

---

## Context

Kailangan ng CSRF protection para sa session-based auth.

## Decision

Double-submit cookie pattern -- CSRF token sa cookie + header, i-compare.

## Consequences

### Positive
- Stateless -- walang server-side storage ng CSRF token
- Simple -- 1 cookie + 1 header
- Compatible sa session-based auth

### Negative
- Kailangan ng JavaScript para i-set ang header
- Vulnerable sa XSS (pero may HttpOnly session cookie naman)

### Neutral
- SameSite=Lax bilang additional layer

## Alternatives Considered

### Synchronizer token pattern
- **Why rejected:** Kailangan ng server-side storage per session

### SameSite=Strict only
- **Why rejected:** Hindi compatible sa lahat ng browser/flow

## References

- internal/middleware/csrf.go
EOF

cat > "$ADR_DIR/0004-bcrypt-cost-10.md" << 'EOF'
# ADR 0004: Bcrypt Cost 10

**Status:** Accepted
**Date:** 2026-09-17
**Deciders:** [your name]

---

## Context

Kailangan ng password hashing. Bcrypt cost ay 4-31, default 10.

## Decision

Bcrypt cost 10.

## Consequences

### Positive
- ~100ms per hash -- balance ng security at UX
- Standard sa industry
- Pwede i-increase later kung needed

### Negative
- Mas mababa kaysa sa cost 12 (mas secure)
- Mas mataas kaysa sa cost 8 (mas mabilis)

### Neutral
- May `golang.org/x/crypto/bcrypt` na

## Alternatives Considered

### Cost 12
- **Why rejected:** ~400ms -- masyadong mabagal para sa UX

### Argon2id
- **Why rejected:** Mas komplikado, bcrypt ay sapat na

## References

- internal/service/auth.go
EOF

cat > "$ADR_DIR/0005-account-lockout.md" << 'EOF'
# ADR 0005: Account Lockout (5 attempts, 15 min)

**Status:** Accepted
**Date:** 2026-09-17
**Deciders:** [your name]

---

## Context

Kailangan ng protection laban sa brute-force login attempts.

## Decision

5 failed logins -> 15 min lockout.

## Consequences

### Positive
- Simple -- madaling maintindihan
- Effective laban sa brute-force
- Standard sa industry

### Negative
- Pwede i-DOS ng attacker (i-lock ang legitimate user)
- Kailangan ng unlock mechanism

### Neutral
- May rate limiting din (5/min) bilang additional layer

## Alternatives Considered

### Exponential backoff
- **Why rejected:** Mas komplikado, mas mahirap i-explain

### CAPTCHA
- **Why rejected:** Dagdag na dependency, UX impact

## References

- internal/service/auth.go
- internal/middleware/ratelimit.go
EOF

echo "      OK"
echo ""

echo "[5/6] Writing global AGENTS.md..."

if [ -f "$GLOBAL_DIR/AGENTS.md" ]; then
  echo "      SKIP (already exists)"
else
  cat > "$GLOBAL_DIR/AGENTS.md" << 'EOF'
# Global Agent Instructions

> Universal rules. Inherited by all projects.
> Project-specific rules override these.

---

## Tone

- Direct, no fluff
- No emojis unless asked
- No "I'd be happy to help" -- just do it

## Git

- Conventional commits: `type(scope): msg`
- Types: feat, fix, docs, test, refactor, chore, perf, ci
- Never commit without tests passing
- Never force-push to main

## Code

- Test before claiming done
- No new deps without justification
- Follow existing patterns
- Small, focused changes

## Context Hygiene

- 1 ticket = 1 context window
- Clear context after each ticket
- Load only relevant files
- Red flag: 5+ context clears per ticket

## Review

- Converge, don't just pass
- 2 consecutive clean reviews
- Triage: blocker / should-fix / nit
- Don't suggest refactors outside diff scope

## Security

- Never commit secrets
- Never log sensitive data
- Validate all input
- Use parameterized queries

## Inheritance

- Global -> stack -> project -> module
- Pinaka-specific ang panalo
- Kapag conflict, i-document sa project AGENTS.md

---

Last updated: 2026-09-20
EOF
  echo "      OK"
fi
echo ""

echo "[6/6] DeepSeek CLI wrapper..."

if [ -f "$HOME/bin/ds" ]; then
  echo "      SKIP (already exists)"
else
  mkdir -p "$HOME/bin"
  cat > "$HOME/bin/ds" << 'EOF'
#!/usr/bin/env bash
# DeepSeek CLI wrapper
# Usage: ds "prompt" file1 file2 ...
#    or: PROMPT="..." ds file1 file2 ...

set -euo pipefail

if [ -z "${DEEPSEEK_API_KEY:-}" ]; then
  echo "Error: DEEPSEEK_API_KEY not set" >&2
  echo "  export DEEPSEEK_API_KEY=\"sk-...\"" >&2
  exit 1
fi

PROMPT="${1:-${PROMPT:-}}"
if [ -n "${1:-}" ]; then shift; fi

CONTEXT=""
for file in "$@"; do
  if [ -f "$file" ]; then
    CONTEXT+="=== $file ===\n"
    CONTEXT+="$(cat "$file")\n\n"
  fi
done

PAYLOAD=$(jq -n \
  --arg sys "You are a senior Go backend engineer working on template-go-echo-squirrel. Follow AGENTS.md strictly. Be direct, no fluff." \
  --arg user "${CONTEXT}
${PROMPT}" \
  '{model: "deepseek-chat", messages: [{role: "system", content: $sys}, {role: "user", content: $user}]}')

curl -s https://api.deepseek.com/v1/chat/completions \
  -H "Authorization: Bearer $DEEPSEEK_API_KEY" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD" | jq -r '.choices[0].message.content'
EOF
  chmod +x "$HOME/bin/ds"
  echo "      OK (~/bin/ds)"
fi
echo ""

echo "=========================================="
echo " DONE"
echo "=========================================="
echo ""
echo "Structure:"
find "$AI_DIR" -type f | sort | sed "s|$REPO_ROOT/||"
echo ""
echo "ADRs:"
find "$ADR_DIR" -type f | sort | sed "s|$REPO_ROOT/||"
echo ""
echo "Global:"
echo "  $GLOBAL_DIR/AGENTS.md"
echo ""
echo "Next steps:"
echo "  1. echo 'export PATH=\"\$HOME/bin:\$PATH\"' >> ~/.bashrc"
echo "  2. echo 'export DEEPSEEK_API_KEY=\"sk-...\"' >> ~/.bashrc"
echo "  3. source ~/.bashrc"
echo "  4. bash replace-agents-md.sh"
echo "  5. git add -A && git commit -m 'chore: add AI workflow'"
echo ""
