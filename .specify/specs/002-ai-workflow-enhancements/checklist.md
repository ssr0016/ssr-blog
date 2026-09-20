# Feature Checklist

## Feature: AI Workflow Enhancements (002)
**Started:** 2026-09-20
**Branch:** main
**ADR:** docs/adr/0006-bounded-converge.md

Evidence rule: a checked `[x]` item cites the command run and the result observed. An item
with no observed result stays `[ ]`.

---

## Phase 1: Understand

- [x] Spec, plan, tasks written and committed -- `git log --oneline -1 5b27930` -> `docs(spec): add 002 ai workflow enhancements spec, plan, tasks`
- [ ] Grill Q&A log in notes.md -- not recorded; resolutions live in spec.md "Open Questions"

## Phase 2: Build

Per ticket: clear context -> load AGENTS + ticket -> test-first -> commit

### T1: Governance -- ADR 0006 and constitution Article 7.4

Checks were written before the change and run as one script (scratchpad `t1_checks.sh`; each
command is also listed below in single-line form).

- [x] Baseline before any change failed as expected -- script output `failures: 25` (ADR missing, 7.4 missing, version 1.0). C2b and the constitution hygiene checks passed only because nothing had changed yet.

After the change (script output `failures: 0`, 29 PASS lines):
- [x] C1: 7.4 is after 7.3 and before Amendments -- `grep -n -E '^### 7\.[34]|^## Amendments'` -> 7.3 at line 143, 7.4 at 148, Amendments at 155
- [x] C2a: only additions plus version line -- `git diff -U0 .specify/memory/constitution.md` shows one removed line, `-Version: 1.0`, and added lines are the 7.4 block plus `+Version: 1.1`
- [x] C2b: 7.1 to 7.3 unchanged -- `diff <(git show HEAD:.specify/memory/constitution.md | sed -n '/^### 7.1/,/^- ADR if decision changed\./p') <(sed -n '/^### 7.1/,/^- ADR if decision changed\./p' .specify/memory/constitution.md)` -> no output
- [x] C2c: version is 1.1 -- `grep -n '^Version:'` -> `164:Version: 1.1`. "Last updated" is `2026-09-20`, which is today, so that line did not change.
- [x] C3: 7.4 states (a), (b), (c), names ADR 0006 and converge.yaml -- `sed -n '/^### 7.4/,/^## Amendments/p' ... | grep -c -F -- TEXT` >= 1 for `(a)` `(b)` `(c)` `read-only` `2 consecutive` `converge.yaml` `ADR 0006` (7 PASS)
- [x] C4: ADR has every template section, numbered 0006, no gap -- 12 section greps PASS (title, Status, Date, Deciders, Context, Decision, Consequences, Positive, Negative, Neutral, Alternatives Considered, References); `ls docs/adr` -> 0001 to 0006 contiguous
- [x] C5: fresh agent reads 7.3 and 7.4 together -- verdict NO CONTRADICTION (general-purpose subagent, read-only, given only the file paths). It listed 4 ambiguities in 7.4, recorded in notes.md "Known Issues". None is a contradiction.
- [x] H1: ASCII only -- `grep -c -P '[^\x00-\x7F]'` -> 0 for constitution.md and the ADR (script PASS)
- [x] H2: no `<<` -- `grep -c '<<'` -> 0 for both files (script PASS)
- [x] `make lint` -- output `0 issues.`, exit 0
- [x] `make test` -- exit 0, every package `ok` (results cached, valid because no Go code changed)
- [x] Workflow-only guard -- tasks.md rule 4 command printed nothing (grep exit 1)
- [x] Template dirs identical -- `diff -rq .ai-workflow/templates .specify/templates` printed nothing, exit 0
- [x] `bash -n`: not applicable, T1 touches no scripts
- [x] checklist.md, notes.md, metrics.md updated -- created in this ticket (metrics.md by hand per the bootstrap note)
- [x] Committed -- `git log --oneline -1 e13f2aa` -> `docs(constitution): add Article 7.4 converge`

### T2 to T9

### T2: Reliable write -- ignore rule and writefile.sh

- [x] Tests written first -- first run: 9 passed, 14 failed
- [x] run.sh passes -- output: 23 passed, 0 failed, exit 0
- [x] bash -n clean on 4 scripts -- all SYNTAX OK
- [x] No non-ASCII bytes -- all 4 scripts ASCII clean
- [x] grep -c << writefile.sh is 0 -- output: 0
- [x] git check-ignore works -- 3 PASS in test_ignore_rule.sh
- [x] make lint -- output: 0 issues.
- [x] make test -- exit 0, all packages ok
- [x] Workflow-only guard -- only .specify/ changes
- [x] Not committed yet -- awaiting user
### T3: Edit verification -- verify-edit.sh

- [x] Tests written first -- first run: 20 passed, 1 failed (case6_quote fixture bug)
- [x] run.sh passes -- output: 21 passed, 0 failed, exit 0
- [x] bash -n clean on both scripts -- SYNTAX OK
- [x] ASCII only -- verify-edit.sh and test_verify_edit.sh both clean
- [x] grep -c << on verify-edit.sh is 0 -- output: 0 (used temp file + trap instead of process substitution)
- [x] Exit codes: 0 landed, 1 not landed, 2 unreadable, 64 bad usage -- case5_* tests
- [x] make lint -- output: 0 issues.
- [x] make test -- exit 0, all packages ok
- [x] Workflow-only guard -- only .specify/ changes
- [x] Not committed yet -- awaiting user
- [ ] T4: Templates -- pending
### T5: Config -- converge.yaml

- [x] Tests written first -- first run: 6 passed, 3 failed (yaml missing + comment regex)
- [x] run.sh passes -- output: 9 passed, 0 failed, exit 0
- [x] bash -n clean -- SYNTAX OK on test_converge_yaml.sh
- [x] ASCII only -- converge.yaml and test file both clean
- [x] Three flat keys, positive integers, with one-line comments -- grep -E confirmed
- [x] Defaults match spec behavior 10 -- test reads values from spec.md
- [x] make lint -- output: 0 issues.
- [x] make test -- exit 0, all packages ok
- [x] Workflow-only guard -- only .specify/ changes
- [x] Not committed yet -- awaiting user
- [ ] T6: Converge command and review prompt -- pending
- [ ] T7: Claim verification in implement/build -- pending
- [ ] T8: sync.sh and install -- pending
- [ ] T9: Verification and dogfood -- pending

## Phase 3: Verify

- [ ] Converged: 2 consecutive clean rounds (T9)

## Phase 4: Ship

- [ ] Pending

---

## Notes

- Context clears: 1 (see metrics.md)
- Blockers: none
- Decisions: see notes.md
- ADRs created: docs/adr/0006-bounded-converge.md (T1, commit e13f2aa)
