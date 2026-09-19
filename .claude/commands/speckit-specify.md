---
description: Write spec (WHAT and WHY) for a feature
---

# Speckit: Specify

Write WHAT and WHY. Not HOW.

## Input
Feature description: $ARGUMENTS

## Process

1. Read .specify/memory/constitution.md
2. Read AGENTS.md
3. Create spec directory: .specify/specs/NNN-feature-name/
4. Write spec.md using .ai-workflow/templates/spec.md
5. Include:
   - Problem (1-2 sentences)
   - Solution (1-2 sentences)
   - Behaviors (numbered)
   - Edge cases (with expected behavior)
   - Non-goals (at least 2)
   - Success criteria (checkboxes)
   - Open questions

## Rules
- No code in spec
- No "how" -- only "what" and "why"
- Non-goals are mandatory
- Edge cases must have expected behavior

## Output
.specify/specs/NNN-feature-name/spec.md
