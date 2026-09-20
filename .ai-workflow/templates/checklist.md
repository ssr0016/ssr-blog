# Feature Checklist

## Feature: [Name]
**Started:** YYYY-MM-DD
**Branch:** feature/xxx
**ADR:** docs/adr/NNNN-xxx.md (kung may decision)

Evidence rule: a checked `[x]` item cites the command run and the result observed. An item
with no observed result stays `[ ]`.

---

## Phase 1: Understand

- [ ] Grilled -- Q&A log in notes.md, no "I assume" left
- [ ] Spec written (spec.md)
- [ ] Plan written (plan.md)
- [ ] Tickets generated (tasks.md)

## Phase 2: Build

Per ticket: clear context -> load AGENTS + ticket -> test-first -> commit

- [ ] T1: [name] -- context fresh, tests pass, evidence cited
- [ ] T2: [name] -- context fresh, tests pass, evidence cited
- [ ] T3: [name] -- context fresh, tests pass, evidence cited
- [ ] ... (add as needed)

## Phase 3: Verify

Converge runs as bounded rounds. See the converge command for the fixed rotation.

- [ ] Round 1: two parallel reviewers -- findings in findings.md
- [ ] Round 2: fix verification via verify-edit.sh -- findings in findings.md
- [ ] Round 3: adversarial -- findings in findings.md
- [ ] Round 4: no-context full review -- findings in findings.md
- [ ] All BLOCKER and SHOULD-FIX resolved or explicitly deferred
- [ ] Converged: 2 consecutive clean rounds
- [ ] Caps not reached (rounds / minutes / tokens)

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
