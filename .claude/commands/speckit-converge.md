---
description: Multi-agent review until convergence
---

# Speckit: Converge

Never let the author review their own code.

## Input
Feature name: $ARGUMENTS

## Process

1. Read spec: .specify/specs/NNN-feature-name/spec.md
2. Read AGENTS.md
3. Get diff: git diff main
4. Spawn fresh agent to review:
   - Compare code vs spec
   - Check against AGENTS.md rules
   - Check against constitution
5. Tag findings: BLOCKER / SHOULD-FIX / NIT
6. Author fixes blockers
7. Repeat until 2 consecutive clean reviews

## Rules
- Fresh agent (not author)
- Multi-agent when it matters
- No refactors outside diff scope
- No repeat findings from previous reviews

## Output
- Findings report
- Convergence status: Converged | Not Converged
