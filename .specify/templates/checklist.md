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
