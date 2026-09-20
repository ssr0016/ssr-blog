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
- [x] Committed -- `git log --oneline -1 f99d53f` -> `feat(workflow): add writefile helper with verified, backed-up writes` (the ignore-rule fix landed separately as 058692b)
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
- [x] Committed -- `git log --oneline -1 e5dde30` -> `feat(workflow): add verify-edit helper for scripted-edit checks`
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
- [x] Committed -- `git log --oneline -1 b80e7de` -> `docs(templates): add metrics and findings templates and checklist evidence rule`
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
- [x] Committed -- `git log --oneline -1 f40e932` -> `chore(workflow): add converge.yaml defaults`

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
- [x] Committed -- `git log --oneline` shows 57bb7de `fix(workflow): repair converge typo and tidy command spacing` and 7a091a0 `feat(workflow): add sync helper for repo and home copies`

### T9: Verification and dogfood

Started 2026-09-20T11:27:37Z. Evidence below was gathered at HEAD 31d7ff1, and the later items (S3's minutes-cap run, S21, S22) at 1dd360d, which only adds a test (fix commits 57bb7de, 53d9113, 19832c8, 799eab0, 2d9f001, 4050f45, 40e5fbe, 31d7ff1 and the test-only 1dd360d; T8 is 7a091a0). Walkthrough runs B1 to B3 were scratch runs: their record is in notes.md "T9 Execution Notes", and the scratch files themselves are not part of the repo. A criterion stays `[ ]` when a part of it has no observed result, and says what is missing.

#### Work items (tasks.md T9)

- [x] 1. Run everything -- baseline before any T9 change: `bash .specify/scripts/test/run.sh` 7 files pass; `bash -n` OK on 11 scripts; `make lint` -> `0 issues.`; `make test` exit 0; workflow-only guard `git diff --name-only main -- internal pkg cmd docs/API.md` printed nothing; `diff -rq .ai-workflow/templates .specify/templates` printed nothing. Repeated before each fix commit; last run at 31d7ff1: run.sh 7 passed, lint `0 issues.`, `make test` exit 0.
- [x] 2. Fresh-clone check -- see S17.
- [x] 3. Walkthrough with the new converge command -- B1 (empty range), B2 (Aborted path, max_rounds 1), B3 (planted defect: reset and NIT-only), and two real runs on this feature (run 1 b92af87..53d9113, run 2 53d9113..4050f45). See S1 to S6, S13 and S14.
- [ ] 4. Check the 22 success criteria -- 21 of 22 are `[x]` below; S20 stays open by decision (structurally verified; comparison awaits feature 003, which will give a second metrics file).
- [x] 5. Finalize metrics.md and notes.md -- notes.md has T9 Execution Notes, Accepted open findings and Feature 003 Candidates; metrics.md has the T9 row, `tickets_closed` 9 and the final finding resolutions (16 fixed, 9 deferred, 0 rejected); the findings.md ledger has a final disposition for all 25 findings.
- [x] 6. Confirm constitution 1.1, ADR 0006, no other article or AGENTS.md rule changed -- see S21 and S22: `Version: 1.1`, ADR 0006 present, the constitution diff is 6 insertions and 1 deletion (Article 7.4 and the version line), AGENTS.md identical to b92af87.

#### Success criteria (22, from spec.md)

- [x] **S1. An empty range stops with a clear message and never reports Converged.**
  - Command: `git rev-list --count HEAD..HEAD; git diff HEAD..HEAD | wc -c; grep -n -E 'range is empty|Never report an empty review' .claude/commands/speckit-converge.md`
  - Result: `0`, `0`, and command text at lines 45-46 and 129. Walkthrough B1 followed the text on HEAD..HEAD: it printed the caps, said the range is empty, asked for an explicit range, and did not print "Converged".
  - Limit: a prompt is not a script; the walkthrough was done by the orchestrating agent following the text.
- [x] **S2. A round with one SHOULD-FIX resets the clean count; a round with only NITs does not.**
  - Command: `grep -n -E 'resets the count|introduces a new SHOULD-FIX|NITs do not break' .claude/commands/speckit-converge.md`; walkthrough B3 in a scratch repo.
  - Result: text at lines 86, 92 and 93. B3 clean count: round 1 clean with 3 NITs -> 1; round 2 with 1 SHOULD-FIX (an author fix had deleted `set -u`) -> 0; round 3 with only 4 NITs -> 1 (clean, no reset). `test_converge_yaml.sh` case 6 pins the wording (26 passed).
- [x] **S3. A converge run stops at 4 rounds, 15 minutes or 90K tokens, whichever comes first, with status Aborted and the open findings listed.**
  - Command: `grep -n -E 'Stop condition:|Open findings:' .specify/specs/002-ai-workflow-enhancements/findings.md`; walkthroughs B2 and B3; walkthrough S3: in a scratch clone of 31d7ff1, `sed -i 's/^max_minutes: 15 /max_minutes: 1 /' .specify/converge.yaml` (checked with `verify-edit.sh`, exit 0), then a converge run on 799eab0^..799eab0 with two Plan reviewers, checking the elapsed time each time a reviewer returned.
  - Result: token cap: both real runs (findings.md lines 45-46 and 82-83: 116430 and 108691 of 90000, status Aborted, open findings listed). Round cap: scratch B2 (1 of 1) and B3 (4 of 4), status Aborted, open findings listed. Minutes cap: S3 printed `max_minutes: 1` at the start; when reviewer A returned the elapsed time was 116 s against 60 s, with 27957 of 90000 tokens spent and round 1 of 4, so the minutes cap fired first. Reviewer B was still running, so the round was finished as far as it could be reported, marked incomplete, and not counted as clean; status Aborted, clean count 0, and the open findings (4 SHOULD-FIX and 3 NIT from each reviewer, listed in notes.md "T9 Execution Notes") were handed over. Reviewer B's tokens brought the total to 64756, still under the token cap.
  - Limit: S3, B2 and B3 are scratch runs; their tokens are not counted toward the real cap (metrics.md, "Walkthrough runs").
- [x] **S4. Changing a cap in the settings file changes the limit on the next run, and the run shows the values in effect.**
  - Command: in a scratch clone `sed -i 's/^max_rounds: 4 /max_rounds: 1 /' .specify/converge.yaml`, checked with `verify-edit.sh` (exit 0), then a converge run (walkthrough B2); real file checked with `grep -E '^max_' .specify/converge.yaml`.
  - Result: B2 printed `max_rounds: 1 | max_minutes: 15 | max_tokens: 90000` at the start (the repo file won over the home file, which says 4) and stopped after 1 round, Aborted at the round cap. The real repo file still says 4 / 15 / 90000. Both real runs printed the values at start.
- [x] **S5. Every report names the range and spec reviewed, lists findings by severity, and ends with status, clean count and stop condition.**
  - Command: `grep -c -E 'Range reviewed|Spec reviewed|Consecutive clean count|Stop condition' findings.md`, run once per field.
  - Result: 2, 2, 2, 2, one per round of the two real runs; each round lists its findings in a table with a Severity column and ends with Status, Consecutive clean count and Stop condition lines (findings.md lines 43-45 and 80-82).
- [x] **S6. The ledger shows every finding with a disposition, and a rejected finding is not re-raised as new.**
  - Command: `sed -n '/^## Ledger/,$p' findings.md`, counting the Disposition column; `grep` of the Repeat of column.
  - Result: 25 ledger rows, every one with a disposition (at this step: 9 fixed, 4 deferred, 12 open; the open ones become fixed or deferred when the ledger is finalized at the T9 commit). Repeats are marked in the Repeat of column: real run 2 finding 16 repeats 5; scratch B3 round 4 marked its 3 findings as repeats of earlier ledger entries.
  - Limit: no finding with disposition "rejected" was re-raised in T9; the repeat rule was exercised on fixed and deferred findings.
- [x] **S7. Every deferred SHOULD-FIX and NIT appears in the feature's notes.**
  - Command: `sed -n '/^## Accepted open findings/,/^## Feature 003/p' notes.md | grep -E '^- #N: '` for N in 10, 11, 12, 14, 21, 22, 23, 24, 25.
  - Result: all 9 present, each with a one-line reason; they are also summarized under Deferred SHOULD-FIX and NIT. No SHOULD-FIX is deferred (all 14 are fixed).
- [x] **S8. A reviewer round changes no file other than the findings report, ledger and metrics file.**
  - Command: `git status --short` before and after each round, compared.
  - Result: identical before and after in both real rounds (findings.md lines 40 and 77) and in all 13 reviewer runs (4 real, 2 in B2, 5 in B3, 2 in S3). The reviewers were Plan agents, which have no Edit or Write tool.
- [x] **S9. Any content, including quotes, backticks, dollar signs and heredoc-like lines, can be written by the standard method with no editor.**
  - Command: a 4-line payload with double quotes, a backtick command, `$HOME`, an `EOF` line and single quotes, written with `writefile.sh out --lines 4 --sig sig-9 < payload`, then `cmp payload out`.
  - Result: exit 0, `cmp` byte-identical, the `EOF` line and the backticks present in the output. `test_writefile.sh` case 1 (`case1_byte_identical`) PASS.
- [x] **S10. A write whose result does not match its line count or signature exits non-zero and reports the difference.**
  - Command: `printf 'x\ny\n' | writefile.sh tgt --lines 99 --sig y --no-backup`, then the same with `--lines 2 --sig zzz`.
  - Result: `writefile: line count mismatch: expected 99, got 2` (exit 1) and `writefile: signature mismatch: last line is not: zzz` (exit 1); the existing target was unchanged (`a|b|`). Since 799eab0 both flags are also required in stdin mode (`test_writefile.sh` case 3, 7 assertions, now real refusals).
- [x] **S11. Overwriting an existing file leaves a timestamped backup unless the caller opted out.**
  - Command: `writefile.sh t --from src` over an existing `t`; then the same with `--no-backup`; `ls t.bak.*`.
  - Result: one backup `t.bak.2026-09-20T13-02-23` by default and none with `--no-backup`. `test_writefile.sh` case 7 PASS.
- [x] **S12. Backups are ignored by git, and backups older than 7 days are gone after the next write.**
  - Command: `git check-ignore -v .specify/scripts/writefile.sh.bak.2026-09-20T12-00-00 .specify/specs/002-ai-workflow-enhancements/notes.md.bak.2026-09-20T12-00-00`; `test_writefile.sh` cases 11, 13, 14.
  - Result: ignored by `.gitignore:19:*.bak.*` and `.specify/specs/.gitignore:1:*.bak.*`; `test_ignore_rule.sh` PASS. Cleanup: 8-day-old backups removed, 6-day-old kept, non-matching names untouched, backups of other files and of dotfiles removed (cases 11, 13, 14 PASS; mutation-checked in 31d7ff1).
- [x] **S13. Converge uses the fixed rotation: two parallel reviewers, then fix verification, then adversarial, then no prior context.**
  - Command: `grep -n -E '^\*\*Round [1-4]\.' .claude/commands/speckit-converge.md`.
  - Result: lines 61, 69, 74 and 78 state the four rounds. Round 1 ran with two parallel reviewers in both real runs and in B2 and B3. Rounds 2 to 4 were exercised in scratch B3: round 2 began with `verify-edit.sh` on the round 1 fix (exit 0) before the reviewer ran; round 3 was adversarial; round 4 gave the reviewer only the spec and the diff. Both real runs stopped after round 1 at the token cap.
- [x] **S14. Shipping after an Aborted run is possible only with a written "Accepted open findings" section listing every open finding with a reason.**
  - Command: walkthrough B2 with a scratch check script against the scratch ledger (no section, incomplete section, complete section, and a complete section with one finding removed); `grep -c '^## Accepted open findings' notes.md`.
  - Result: no section -> "ship: NOT allowed ... A new converge run is required"; incomplete -> "Not accepted (missing or no reason): F2 F3 F4 F5 F6"; complete -> "ship: allowed"; complete minus one -> not allowed, naming F6. The real notes.md has the section (count 1) with all 9 non-fixed NITs and a reason for each.
  - Limit: the rule lives in the command text; the check script was a scratch tool, and no repo tool enforces it.
- [x] **S15. A look-alike non-ASCII character where ASCII is expected is rejected with its position.**
  - Command: `printf 'ab%s\nsig\n' "$cyr" | writefile.sh out --lines 2 --sig sig --no-backup` with `cyr=$(printf '\320\260')` (a Cyrillic a).
  - Result: `writefile: non-ASCII byte at line:column 1:3`, exit 1, no file created. `test_writefile.sh` `case5_cyrillic_rejected` PASS (a real refusal since 799eab0; before that the case was vacuous, see notes.md).
- [x] **S16. No committed script and no command the workflow asks the user to paste contains a multi-line heredoc.**
  - Command: `bash .specify/scripts/test/test_no_heredoc.sh`; `grep -c -F` of the heredoc marker in each script, command and prompt.
  - Result: 14 passed, 0 failed; the marker count is 0 in sync.sh, verify-edit.sh, writefile.sh, the four command files, 05-build.md and 06-review.md. Test files are exempt by design and hold the marker as fixture data.
- [x] **S17. On a fresh clone, the write helper and the converge settings work without copying anything from the home directory.**
  - Command: `git clone` to a temp dir, `HOME` set to an empty temp dir, `SPEC_KIT_HOME` unset, then `bash .specify/scripts/test/run.sh`; then `printf 'alpha\nsig\n' | .specify/scripts/writefile.sh OUT --lines 2 --sig sig` and `grep -E '^max_' .specify/converge.yaml` in the clone.
  - Result: at HEAD 31d7ff1 (160 tracked files, 0 backup files): 7 passed, 0 failed; the helper exit 0; converge.yaml 4 / 15 / 90000; the empty home stayed empty (0 entries). Also done at A3 (53d9113). Run again at HEAD 1dd360d for the T9 commit: 160 tracked files, 0 backup files, `run.sh` 7 passed, 0 failed, the helper and `verify-edit.sh` run from the clone with exit 0, converge.yaml 4 / 15 / 90000, the empty home stayed empty (0 entries).
- [x] **S18. A scripted edit that did not apply produces no "fixed" claim, and the user sees the failure.**
  - Command: `printf 'keep this\n' > f; sed -i 's/NEVER_MATCHES/xxx/' f; verify-edit.sh f --contains xxx`.
  - Result: `verify-edit: not landed: missing: xxx`, exit 1. During T9 the failure was shown to the user twice and the claim was withheld until re-checked: `--absent 'plus'` in the Commit 3 check (it matched an unrelated "plus a ledger table") and `--absent 'pending | pending'` in the run 2 metrics check (it matched the T9 row). `test_verify_edit.sh` 56 passed.
- [x] **S19. Each feature has a metrics file with every field of behaviors 29-31 filled in, updated as events occurred.**
  - Command: `grep -c -F` for each of the 13 field labels in metrics.md; `grep -n -E '^\| T9 \||^(status|tickets_closed):|^\| Tickets closed \|' metrics.md`; an awk check of every per-ticket cell; `bash .specify/scripts/test/test_templates.sh`.
  - Result: all 13 labels are present (count 1 each, except the labels repeated by the second Converge table: "Context clears" 2, "Caps in effect", "Total minutes", "Total tokens" and "Stop condition" 2 each). The T9 row is `| T9 | 124 | not tracked | not tracked |`, `tickets_closed` is 9 in the table and in the summary block, and `status: aborted` (the converge status; the template has no "done"). All 9 per-ticket rows hold only a number or "not tracked" (0 bad cells, no "pending" left). The Converge table has both real runs (caps, minutes, tokens, rounds, stop condition) and the scratch runs sit in their own table. The two real-run rows were written as the events happened: the run 2 row with its start time before round 1, then completed when the round ended. `test_templates.sh` 53 passed, including the feature summary keys.
  - Note: T2 to T8 wall-clock and tokens are "not tracked", not estimated; T9's 124 minutes runs from the recorded start to a measurement before the commit and includes time waiting for review.
- [ ] **S20. Two features can be compared by tickets, clears, rounds, findings, time and tokens from their metrics files alone.**
  - Command: `grep -E '^[a-z_]+:' metrics.md` (the summary keys); `ls .specify/specs/*/metrics.md`; `test_templates.sh` feat002 summary cases.
  - Result: the summary block has the 12 comparison keys (feature, status, tickets, tickets_closed, context_clears, converge_rounds, converge_minutes, converge_tokens, findings_blocker, findings_should_fix, findings_nit, tickets_reopened) and matches the template.
  - Missing: only one feature (002) has a metrics file; feature 001 has none (by design, nothing is invented). A real comparison needs a second file, which feature 003 will provide.
- [x] **S21. Constitution Article 7.4 exists with (a) reviewer read-only, (b) two-consecutive-clean unchanged, (c) caps, backed by an ADR and a `docs(constitution)` commit.**
  - Command: `grep -n '^Version:' .specify/memory/constitution.md` (and the same on `git show b92af87:` of it); `git log --oneline b92af87..HEAD -- .specify/memory/constitution.md`; `grep -c -F` of `(a)`, `(b)`, `(c)`, `read-only`, `2 consecutive`, `converge.yaml`, `ADR 0006` in the 7.4 block; `ls docs/adr`; `grep -n -E '^#{1,2} '` on the ADR; `git log --oneline --grep='docs(constitution)'`.
  - Result: `Version: 1.1` at line 164 (it was 1.0 at b92af87). The only commit touching the constitution is e13f2aa `docs(constitution): add Article 7.4 converge`. Article 7.4 (lines 148-153, after 7.3 and before Amendments) states (a) the reviewer is read-only apart from the findings and metrics artifacts, (b) the 7.3 rule is unchanged (Converged means 2 consecutive clean rounds), and (c) caps set in `.specify/converge.yaml`, with Aborted and the Accepted-open-findings rule; each key term counts 1 in the block, and it names ADR 0006. `docs/adr/` holds 0001 to 0006 without a gap; 0006-bounded-converge.md has Status Accepted, Date, Deciders, Context, Decision, Consequences (Positive, Negative, Neutral), Alternatives Considered and References, and states the repo file, home file, defaults precedence.
- [x] **S22. No other existing article or AGENTS.md rule is contradicted.**
  - Command: `git diff b92af87..HEAD --stat -- .specify/memory/constitution.md AGENTS.md`; `diff <(git show b92af87:.specify/memory/constitution.md | sed -n '/^### 7.1/,/^- ADR if decision changed\./p') <(sed -n '/^### 7.1/,/^- ADR if decision changed\./p' .specify/memory/constitution.md)`; a `diff` of the whole constitution against b92af87 with the 7.4 block and the Version line removed; `git diff --quiet b92af87..HEAD -- AGENTS.md`; `git diff --name-status b92af87..HEAD -- docs/adr`.
  - Result: the constitution diff is 6 insertions and 1 deletion (the five lines of Article 7.4 and `Version: 1.0` -> `Version: 1.1`). Articles 7.1 to 7.3 are byte-identical (no diff output), and so is every other line of the file. AGENTS.md is identical to b92af87 (`git diff --quiet` exit 0; last changed by 47dda7d, before this feature). Only ADR 0006 was added; no other ADR changed. The T1 fresh-agent read of 7.3 and 7.4 together (C5) found no contradiction.
  - Limit: the mechanical part is a diff, so "nothing else changed" is exact. "No contradiction" is a reading: it was made by the T1 fresh reader before the T9 command changes (2d9f001), which touch no article and no AGENTS.md rule.

## Phase 3: Verify

- [ ] Converged: 2 consecutive clean rounds -- not reached. Both real converge runs ended Aborted at the token cap after round 1 (findings.md).
- [x] Aborted path taken -- the written "Accepted open findings" section in notes.md lists every non-fixed finding with a reason (9 NITs; all 14 SHOULD-FIX and the BLOCKER are fixed), and the user approved it at the review of the notes step. The fixes after run 2 were checked by tests and mutation testing, not by a third run (user decision).

## Phase 4: Ship

- [ ] Pending

---

## Notes

- Context clears: 3 (see metrics.md)
- Blockers: none
- Decisions: see notes.md
- ADRs created: docs/adr/0006-bounded-converge.md (T1, commit e13f2aa)
- Drift: RESOLVED in T8. Repo and home copies of all four speckit command files are identical (`diff -q` silent). Commits 57bb7de, 7a091a0, 53d9113 and 2d9f001.
