---
description: Implement one ticket in a fresh context
---

# Speckit: Implement

One ticket = one fresh context window.

## Input

Ticket ID: $ARGUMENTS

## Process

1. Read AGENTS.md.
2. Read `.specify/specs/NNN-feature-name/tasks.md` and take only the specified ticket.
3. Write the test first, then the code.
4. Run the tests:
   - Go code changed: `make test`
   - Workflow files changed (scripts, commands, prompts, templates, spec text):
     `bash .specify/scripts/test/run.sh`
   - If both changed, run both.
5. Show the diff and the test results.
6. When the ticket closes, update `.specify/specs/NNN-feature-name/metrics.md`
   for this ticket: wall-clock minutes, tokens used, whether 100K was crossed.
   Never leave a field blank; use `0` or `not tracked`.

## Verify before you claim

After any scripted edit (sed, perl -i, python replace, or similar), confirm the
edit landed before writing any claim about it in notes, the checklist, the
ledger, or a commit message. Use `.specify/scripts/verify-edit.sh` where it
applies.

A claim recorded as done points to something observed -- a test result, a file
state, a command output -- not to intent.

If the check fails, do not write the claim. Show the failure to the user
immediately.


## Writing files

Write files through `.specify/scripts/writefile.sh` or the agent's file tool.
Never use a multi-line heredoc.

When the user must run something in a terminal, give them a single line or a
saved script file. Never a multi-line block that depends on paste behaving.

## Rules

- One ticket per fresh context.
- Test first, then code.
- No batching.
- No new deps without an ADR.
- Follow AGENTS.md ticket-type rules, including the workflow-change rules when
  the ticket is that type.

## Output

- Test file (new or changed)
- Implementation
- Test results
- Diff summary
- Updated metrics row for this ticket
