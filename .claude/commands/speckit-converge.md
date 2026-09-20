---
description: Multi-agent review until convergence, bounded by caps
---

# Speckit: Converge

Never let the author review their own code. Converge is bounded: it stops on
two consecutive clean rounds, on any cap, or on your interrupt.

## Input

Feature name: $ARGUMENTS

## Caps

Caps come from `.specify/converge.yaml` plus `~/.spec-kit/converge.yaml`.
Defaults:

- max_rounds: 4
- max_minutes: 15
- max_tokens: 90000

Print the values in effect at the start of every run.

The token cap counts everything spent on converge for this feature, all
rounds and reviewers combined. Token usage is read from the agent session
report. If that is unavailable, the field says `not tracked` and the
token cap cannot fire on that run; the round and minute caps still can.

## Process

1. State the range reviewed: the exact commit range and the spec file.
   - Default range is `git diff main...HEAD` when the branch differs from main.
   - If the range is empty, say the range is empty, ask for an explicit range,
     and stop. Never report an empty review as clean.
2. Read the spec at `.specify/specs/NNN-feature-name/spec.md` and `AGENTS.md`.
3. Run rounds from the fixed rotation below. A fifth round is never needed;
   the round cap stops the run first.
4. After each round, update `findings.md` (report + ledger) and `metrics.md`.
5. Stop on the first of:
   - two consecutive clean rounds -> **Converged**
   - any cap reached -> **Aborted** (Not Converged)
   - user interrupt -> **Not Converged** (interrupted)

## Rotation

The rotation is fixed by the workflow, not chosen by the user. Round 1 counts
the two parallel reviewers as one round.

**Round 1.** Spawn two reviewers in parallel.
- Reviewer A: spec conformance. Compare the change to each behavior it claims
  to implement.
- Reviewer B: AGENTS.md rules and security (auth, injection, CSRF, rate
  limit, error handling, context usage).
The two count as one round. The round is clean only if both reviewers are
clean.

**Round 2.** First verify the round 1 fixes actually landed, using
`.specify/scripts/verify-edit.sh` on the files the author touched. If a fix
did not land, that is a new finding and the round is not clean. Then run a
General review of the change.

**Round 3.** Adversarial inputs and test quality. Look for inputs the tests
do not cover, tests that assert too little, and cases where the implementation
passes but the behavior is wrong.

**Round 4.** Independent full review with no prior context. Give the reviewer
the spec and the diff only. No earlier findings, no ledger.

## Severity

Every finding is BLOCKER, SHOULD-FIX, or NIT.

A round is **clean** only if it has zero BLOCKER and zero SHOULD-FIX findings.
NITs do not break a clean round but are still recorded.

The feature is **Converged** only after two consecutive clean rounds. Any
non-clean round resets the count to zero.

## Ledger

Findings are kept in `.specify/specs/NNN-feature-name/findings.md`: one report
per round, plus a ledger table. Each ledger entry records round, severity,
summary, disposition (fixed / deferred with reason / rejected with reason),
and, if a repeat, which earlier finding it repeats.

The orchestrating agent compares each reported finding against the ledger and
marks repeats. This is why round 4 can stay context-free.

A finding already in the ledger is not raised again as new.

Every non-fixed SHOULD-FIX and NIT ends up in the feature `notes.md` with
its disposition. Nothing is dropped silently.

## Reviewer is read-only

During a round the reviewer changes nothing except the findings report, the
ledger (`findings.md`), and `metrics.md`.

Fixes are made by the author between rounds, stay inside the scope of the
change under review, and out-of-scope problems are recorded in `notes.md` as
known issues.

## Empty range

If the current branch equals main and a plain diff is empty, say the range is
(empty and ask for an explicit range. Do not report Converged.

## Interrupted or capped mid-round

If the minute or token cap is hit in the middle of a round, finish the round as
far as it can be reported, mark it incomplete, and do not count it as clean.

If you interrupt converge, status is Not Converged (interrupted); the ledger
and metrics are saved as they stand.

## Output

End every run with:

- Status: Converged | Not Converged | Aborted
- Consecutive-clean count
- Stop condition that fired (two consecutive clean / rounds cap / minutes cap / tokens cap / interrupt / empty range)

- Open findings still unresolved, if any

An Aborted run hands the open findings to you for a decision and does not claim
the feature is ready to ship. You may ship after an Aborted run only if you
write an "Accepted open findings" section in the feature `notes.md`, listing
each open finding with a one-line reason. Without that written acceptance, a
new converge run is required.

## Re-running after Aborted

Converge may be re-run on a feature that ended Aborted. It starts a new run
with fresh caps only if you ask for it. `metrics.md` records both runs.

## Rules

- Reviewer is never the author of the change under review.
- Reviewers are fresh each round.
- No refactors outside the diff scope.
- No repeat findings from previous rounds; repeats are marked, not re-raised.
