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
### T4: Templates -- metrics, findings, and the evidence rule

- [x] Tests written first -- test_templates.sh created before template reconciliation
- [x] run.sh passes -- 43 passed, 0 failed (5 test files, 99 checks total)
- [x] bash -n clean -- SYNTAX OK on test_templates.sh
- [x] ASCII only -- 7 template/test files clean
- [x] diff -rq .ai-workflow/templates .specify/templates is clean -- identical
- [x] metrics.md fields cover spec behaviors 29-31 -- 9 labels + 12 summary keys tested
- [x] findings.md covers behaviors 1, 7, 13 -- range, spec, ledger, columns, stop condition
- [x] checklist.md has evidence rule -- grep confirmed
- [x] feature 002 metrics reconciled with template -- tickets_closed updated to 4
- [x] make lint -- output: 0 issues.
- [x] make test -- exit 0, all packages ok
- [x] Workflow-only guard -- only .specify/ and .ai-workflow/templates/ changes
- [x] Not committed yet -- awaiting user
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

### T6: Converge command and review prompt

- [x] speckit-converge.md rewritten -- 143 lines, 12 sections: caps, four-round rotation, severity/clean rules, ledger, read-only reviewer, empty-range stop, stop conditions, output status, Accepted-open-findings path
- [x] 06-review.md rewritten -- 71 lines: round briefs (round 1 two parallel reviewers, round 2 verify-edit then general, round 3 adversarial + test quality, round 4 context-free), severity, read-only note
- [x] test_no_heredoc.sh created first -- 75 lines, scans top-level `.specify/scripts/*.sh` for the heredoc marker, scans command/prompt `.md` for the marker inside fences only
- [x] Static checks: `converge.yaml` named 3x, `**Round` 4x, `read-only` present, `Accepted open findings` present, empty-range stop present, 0 fences, 0 marker tokens, ASCII only
- [x] Heredoc test passes: 13 passed, 0 failed
- [x] Drift reported: `.claude/commands/speckit-converge.md` and `speckit-implement.md` differ from `~/.claude/commands/`. Re-sync NOT done; needs user approval. See notes.md.
  RESOLVED in T8: repo copies were re-synced to `~/.claude/commands/` after user approval; `diff -q` is silent for all four command files (see T8).
- [x] tasks.md T6 Files: block corrected -- added speckit-implement.md, 05-build.md, test_no_heredoc.sh (334 lines total)
- [x] `.gitignore` `*.bak.*` fix landed as separate commit 058692b

#### Behavior mapping (spec behaviors 1-14)

| # | Where in speckit-converge.md |
|---|---|
| 1 range + spec reviewed; empty range stops | ## Process step 1; ## Empty range |
| 2 fresh reviewers, fixed 4-round rotation | ## Rotation |
| 3 severities BLOCKER/SHOULD-FIX/NIT | ## Severity |
| 4 clean = 0 BLOCKER + 0 SHOULD-FIX | ## Severity |
| 5 two consecutive clean; reset on non-clean | ## Severity |
| 6 rule unchanged | (not modified) |
| 7 ledger in findings.md; repeats marked | ## Ledger |
| 8 non-fixed findings to notes.md | ## Ledger |
| 9 reviewer read-only | ## Reviewer is read-only |
| 10 caps 4 / 15 / 90000; token accounting | ## Caps |
| 11 stop conditions | ## Process step 5 |
| 12 read converge.yaml, print at start | ## Caps |
| 13 status + open findings + Accepted-open-findings | ## Output |
| 14 grandfathered features | (spec-only; not in command text) |

#### Workflow-change rules

- [x] Behaviors mapped above; no behavior unmapped
- [x] No contradiction with constitution 7.3, 7.4 -- see fresh-agent note
- [x] No multi-line heredoc inside any fence in changed files -- 0 fences
- [x] Drift reported, re-sync deferred to user
- [x] Templates identical

Fresh-agent read of 7.3 + 7.4 against the changed text: 7.4(a) read-only reviewer -> converge.md "Reviewer is read-only"; 7.4(b) two-consecutive-clean unchanged -> "Severity"; 7.4(c) caps in converge.yaml -> "Caps". No contradiction.

### T7: Claim verification and paste-safe writes in implement and build

Advanced during the T6 session; same workflow-change type, no RBAC surface.

- [x] speckit-implement.md rewritten -- 65 lines: verify-before-claim, no-heredoc rule, per-ticket metrics update, both test runners (`make test`, `bash .specify/scripts/test/run.sh`)
- [x] 05-build.md rewritten -- 42 lines, same rules as implement
- [x] Both files ASCII only, 0 fences, 0 marker tokens
- [x] No contradiction with constitution 7.3, 7.4

### T8: Sync -- sync.sh and install

- [x] Tests written first -- `bash .specify/scripts/test/test_sync.sh` before sync.sh existed: 14 passed, 41 failed (every sync.sh call exit 127; the 14 passes were "nothing was created" asserts that hold trivially)
- [x] test_sync.sh passes -- output: 55 passed, 0 failed
- [x] run.sh passes -- 7 test files PASS (converge_yaml 9, ignore_rule 14, no_heredoc 14, sync 55, templates 43, verify_edit 21, writefile 23); summary line `7 passed, 0 failed`
- [x] sync.sh static checks -- `wc -l` 169, `bash -n` OK, non-ASCII 0, `grep -c '<<'` 0, mode 775
- [x] Interface -- exit 0 identical/installed, 1 drift or refused, 2 unreadable, 64 usage; covered by test cases 1-8b
- [x] Install is all-or-nothing -- case 5 (differing file blocks install of missing files), case 6 (`--force` backs up only the differing file as `<name>.bak.YYYY-MM-DDTHH-MM-SS`)
- [x] Tests never touch the real home -- case 10 compares a snapshot of `~/.spec-kit` before and after; equal
- [x] Real check before install (user approved) -- 4 missing, 0 differing, exit 1
- [x] Real install (user approved, no `--force`) -- `install: 4 installed, 0 updated, 0 unchanged`, exit 0; `ls -la ~/.spec-kit/` shows 4 files, no temp or .bak files
- [x] Real check after install -- `ok: 4 of 4 identical`, exit 0 (T8 Done-when met)
- [x] Drift resolved -- diffs shown and approved; typo `(empty` and two blank-line nits fixed in the repo copies first (`verify-edit.sh --absent '(empty'` and `--contains 'range is empty'` exit 0); copied with `writefile.sh --from` (backups `*.bak.2026-09-20T11-19-25`); `diff -q` silent and exit 0 for converge, implement, plan, specify; plan and specify `sha256sum -c` OK (untouched); home copies set to mode 664
- [x] Scope held -- `.claude/commands/` is NOT part of sync.sh; T8 sync scope is the 4 files only (user decision)
- [x] make lint -- output: `0 issues.`, exit 0
- [x] make test -- exit 0, every package ok (results cached, valid because no Go code changed)
- [x] Workflow-only guard -- `git diff --name-only main -- internal pkg cmd docs/API.md` printed nothing; `diff -rq .ai-workflow/templates .specify/templates` printed nothing, exit 0
- [x] checklist.md, notes.md, metrics.md updated
- [ ] Committed -- pending user approval; two commits proposed: `fix(workflow): repair converge typo and tidy command spacing`, then `feat(workflow): add sync helper for repo and home copies`

### T9: Verification and dogfood

- [ ] Pending

## Phase 3: Verify

- [ ] Converged: 2 consecutive clean rounds (T9)

## Phase 4: Ship

- [ ] Pending

---

## Notes

- Context clears: 3 (see metrics.md)
- Blockers: none
- Decisions: see notes.md
- ADRs created: docs/adr/0006-bounded-converge.md (T1, commit e13f2aa)
- Drift: RESOLVED in T8. Repo and home copies of all four speckit command files are identical (`diff -q` silent). Commit reference pending.
