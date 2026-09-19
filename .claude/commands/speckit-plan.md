---
description: Write plan (HOW) as DAG, not phases
---

# Speckit: Plan

Write HOW. As a DAG (Kanban), not sequential phases.

## Input
Technical constraints: $ARGUMENTS

## Process

1. Read spec: .specify/specs/NNN-feature-name/spec.md
2. Read AGENTS.md
3. Read .specify/memory/constitution.md
4. Write plan.md using .ai-workflow/templates/plan.md
5. Generate tickets in tasks.md using .ai-workflow/templates/tasks.md

## Rules
- Each ticket = vertical slice (thin cross-layer cut)
- Explicit blocked_by relationships
- Parallelizable tickets marked
- Order: DB -> repository -> service -> handler -> swagger
- Follow AGENTS.md ticket-type rules

## Output
- .specify/specs/NNN-feature-name/plan.md
- .specify/specs/NNN-feature-name/tasks.md
