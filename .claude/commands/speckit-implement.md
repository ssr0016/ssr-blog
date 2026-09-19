---
description: Implement one ticket in a fresh context
---

# Speckit: Implement

One ticket = one fresh context window.

## Input
Ticket ID: $ARGUMENTS

## Process

1. Read AGENTS.md
2. Read .specify/specs/NNN-feature-name/tasks.md
3. Focus on the specified ticket only
4. Write test first, then code
5. Run make test
6. Show diff + test results

## Rules
- One ticket per fresh context
- Test first, then code
- No batching
- No new deps without ADR
- Follow AGENTS.md ticket-type rules

## Output
- Test file (new/changed)
- Implementation
- Test results
- Diff summary
