# Findings: 002-ai-workflow-enhancements

Report and ledger for converge. One report per round, plus a running ledger.

**Feature:** 002-ai-workflow-enhancements
**Status:** Aborted (run 1 and run 2 both stopped at the token cap)
**Started:** 2026-09-20

---

## Run 1, Round 1

- **Range reviewed:** b92af87..53d9113 (11 commits, 32 files, 3094 diff lines; HEAD is 53d9113)
- **Spec reviewed:** .specify/specs/002-ai-workflow-enhancements/spec.md (35 behaviors, edge cases, 22 success criteria), with plan.md and tasks.md
- **Reviewer brief:** Round 1: two parallel reviewers, both Plan agents (no Edit or Write tool). A = spec conformance, B = AGENTS.md rules and security. The briefs told them to cite only line numbers checked with `cat -n` or `grep -n`.
- **Caps in effect:** 4 rounds / 15 minutes / 90000 tokens (repo file `.specify/converge.yaml`; the home file says the same)
- **How the findings were checked:** the orchestrator reproduced or re-read each claim before recording it. Reproduced by running the script: #1, #7, #8, #9, #13. Confirmed by search or by reading the file: #2, #3, #4, #6, #10, #11, #14. Read from code only, not run: #5. Not reproduced: #12. The reviewers' line numbers for verify-edit.sh were wrong (the file has 118 lines; they cited lines up to 211), so the line numbers below are the orchestrator's `grep -n` results.
- **Findings:**

| # | Severity | Summary | Disposition | Reason | Repeat of |
|---|---|---|---|---|---|
| 1 | SHOULD-FIX | writefile.sh:70-77: in stdin mode `--lines` and `--sig` are optional, so a write with neither runs no verification and exits 0. Spec behavior 18 and plan.md:35 require both. test_writefile.sh case3_missing_sig passes only because `--lines 1` mismatches the payload | open | Reproduced: no flags, only `--lines`, and only `--sig` all exit 0. Run Aborted at the token cap before a fix | - |
| 2 | SHOULD-FIX | speckit-converge.md `## Caps` (line 14 on): no text to validate cap values as positive integers, to record the start time and caps in metrics.md before round 1, or to check the caps before each round and after each reviewer. tasks.md T6 and plan.md:45-46 call for these | open | Search for "positive integer", "start time", "before each round" returns 0 hits | - |
| 3 | SHOULD-FIX | speckit-converge.md `## Severity` to `## Reviewer is read-only` (lines 70-102): four spec edge cases have no text: an unresolved BLOCKER in the ledger blocks a clean round; a reviewer changing a source file is reported as a finding against its brief; a repeat of a rejected finding with new evidence is logged as fresh; a fix that introduces a new SHOULD-FIX is logged as new | open | Search for each phrase returns 0 hits in the command and in 06-review.md | - |
| 4 | SHOULD-FIX | findings.md template line 15 (both template copies): the Reviewer brief options read "Round 1: general / Round 2: fix verification ..." (separated by pipe characters in the file) which contradicts the fixed rotation (round 1 is spec conformance plus rules/security, general review is round 2) | open | Read the template and the rotation; they disagree | - |
| 5 | SHOULD-FIX | writefile.sh:86-101 (loop at 93): the 7-day backup cleanup runs only when the target already exists and backups are on, and only over that one target's own `.bak.*` files. Spec behavior 22 and plan.md:40 say the helper removes its old backups in the target directory | open | Read from code, not run | - |
| 6 | SHOULD-FIX | metrics.md:49-57: per-ticket cells hold prose ("session (not measured separately)", "3 (134 s measured, start to hand-off)") and a "T1 commit" row sits in the per-ticket table. Behaviors 30 and 32 and the template rule allow only a number or "not tracked" | open | Read the file | - |
| 7 | SHOULD-FIX | verify-edit.sh:7 and 101-106: if `mktemp` fails the needle list is never read, no needle is checked, and the script exits 0 ("landed") | open | Reproduced: `TMPDIR=/nonexistent verify-edit.sh FILE --contains ZZZ_NOT_THERE_ZZZ` exits 0 | - |
| 8 | SHOULD-FIX | verify-edit.sh:68, 77, 92: a directory passes the readability gate and grep then exits 2; `--absent` counts that as absent (exit 0) and `--contains` exits 1 instead of the documented 2 | open | Reproduced: `--absent foo` on a directory exits 0 | - |
| 9 | SHOULD-FIX | writefile.sh:103: with `--no-backup` and a directory as TARGET the write exits 0 and leaves `.writefile.XXXXXX` inside the directory; without `--no-backup` the backup `cp` fails and it refuses. The two modes disagree and no test covers a directory target | open | Reproduced: rc=0 and a stray temp file in the directory | - |
| 10 | NIT | findings.md template line 26 vs speckit-converge.md:123: the stop-condition vocabulary differs ("round cap / time cap / token cap / user interrupt" vs "rounds cap / minutes cap / tokens cap / interrupt / empty range") | open | Read both | - |
| 11 | NIT | writefile.sh:64: the `or true` fallback (a double pipe followed by true) on the awk ASCII check makes an awk failure pass as clean | open | Read the line | - |
| 12 | NIT | writefile.sh:86-101: the cleanup's `date -d` and `rm -f` are not best-effort, and any file named `<target>.bak.<timestamp>` older than 7 days is deleted whether or not writefile.sh made it | open | Not reproduced | - |
| 13 | NIT | verify-edit.sh:67, 76: an empty needle is skipped silently when another needle is given, so `--absent ""` cannot fail | open | Reproduced: `--contains aaa --absent ""` exits 0 | - |
| 14 | NIT | sync.sh:134: `chmod --reference` is GNU-only | open | Confirmed the line; the repo targets Linux | - |

- **Findings count (verified):** BLOCKER 0, SHOULD-FIX 9, NIT 5. Raw reviewer counts: A SHOULD-FIX 6 / NIT 1, B SHOULD-FIX 3 / NIT 4. No overlap between the two reports.
- **Round is complete:** yes, both reviewers returned their full reports. It is not marked incomplete.
- **Round is clean:** no (9 SHOULD-FIX).
- **Reviewer read-only check:** `git status --short` had 0 lines before and 0 lines after the round, before the orchestrator wrote this file.
- **Tokens:** reviewer A 71397, reviewer B 45033, total 116430. The cap of 90000 was still unspent after A returned (71397). Reviewer B was already running in parallel, so the total passed the cap when B returned: 26430 over.
- **Wall-clock:** 4 minutes (11:47:51Z to 11:51:54Z, including the orchestrator's checks; the two reviewers returned after 102 s and 118 s).
- **Status:** Aborted (Not Converged)
- **Consecutive clean count:** 0
- **Stop condition:** token cap (116430 of 90000 after round 1; rounds 1 of 4, minutes 4 of 15)
- **Open findings:** #1 to #14 (9 SHOULD-FIX, 5 NIT) at the time of the round. The dispositions in the table above are the ones recorded then; the final ones are in the Ledger.

## Run 2, Round 1

- **Range reviewed:** 53d9113..4050f45 (4 fix commits, 10 files, 626 diff lines): 19832c8 verify-edit, 799eab0 writefile, 2d9f001 converge command, 4050f45 findings template
- **Spec reviewed:** .specify/specs/002-ai-workflow-enhancements/spec.md (35 behaviors), with plan.md and tasks.md
- **Why a second run:** run 1 ended Aborted. The user asked for a new run on the fix commits only. Fresh caps apply to a new run after Aborted, and metrics.md records both runs.
- **Reviewer brief:** Round 1: two parallel reviewers, both Plan agents (no Edit or Write tool). A = spec conformance of the fixes (does each claimed fix close its run 1 finding without a regression). B = AGENTS.md rules, shell safety, and test quality (can every new test case actually fail). The briefs listed the accepted decisions and the deferred NITs as known, and required line numbers checked with `cat -n` or `grep -n`.
- **Caps in effect:** 4 rounds / 15 minutes / 90000 tokens (repo file; validated as positive integers before the round)
- **How the findings were checked:** the orchestrator reproduced each claim before recording it. Run and measured: #15, #16, #17. Confirmed by mutation (the suite was run against a copy of the script with the named defect put in, and it still passed): #18 (mutants M1 and M1b), #19 (M2), #20 (M3); a fourth mutant (prune never called) was caught by 3 cases, so the prune tests do discriminate. Confirmed by reading the file: #21, #22, #23. Not reproduced: #24, #25 (both are NIT-level and read-only observations).
- **Line numbers:** the reviewers' line numbers matched `grep -n` this time.
- **Fix verdicts from reviewer A:** #1, #3, #4, #7, #8, #9, #13, multi-line needles, and `--count` semantics are fixed. #2 is fixed with one gap (NIT #21). #5 is fixed but the new glob has a regression (finding #16). Reviewer B found writefile.sh, AGENTS.md and constitution compliance clean and the prune function safe (it only removes strict-pattern regular files in the target directory).
- **Findings:**

| # | Severity | Summary | Disposition | Reason | Repeat of |
|---|---|---|---|---|---|
| 15 | BLOCKER | verify-edit.sh:124: a `--count` value too large for the shell (for example 99999999999999999999 or 2^63) makes `[` fail with "integer expression expected", the `if` reads that as false, and the script exits 0 (landed). It is the same fail-open class as run 1 finding #7, in the script that claims to fail closed | open | Reproduced: rc 0 with the shell error on stderr. A rated it SHOULD-FIX, B rated it BLOCKER; the higher rating is kept because it is a false landed. Run stopped at the token cap before a fix | - |
| 16 | SHOULD-FIX | writefile.sh:112: the new cleanup loop `"$tmpdir"/*.bak.*` does not match names that start with a dot, so old backups of dotfile targets (for example .gitignore) are never pruned. The old per-target pattern did match them, so this is a regression of run 1 fix #5 and misses spec behavior 22 for those files | open | Reproduced: an 8-day-old `.gitignore.bak.<ts>` survived a write, while `plain.bak.<ts>` was pruned | 5 |
| 17 | SHOULD-FIX | verify-edit.sh:95-97: `--count` is quadratic in file size, not linear: 218 KB takes 57 ms, 449 KB 198 ms, 909 KB 818 ms, 1.8 MB 4.3 s, and a 9.5 MB file ran over two minutes. The case 16 comment in test_verify_edit.sh says "close to linear", the header says nothing, and a 220 KB timing test cannot show quadratic growth | open | Measured. A rated it NIT, B rated it SHOULD-FIX; the higher rating is kept because the comment is wrong and there is no size guard. Ordinary workflow files (tens of KB) are unaffected | - |
| 18 | SHOULD-FIX | test_verify_edit.sh: glob-literal matching for `--contains` and `--absent` is untested. Dropping the quotes so the text matches as a glob passes all 55 cases | open | Confirmed by mutation (mutants M1 and M1b, 55 of 55 still pass) | - |
| 19 | SHOULD-FIX | test_verify_edit.sh: no non-ASCII file case, so the `export LC_ALL=C` line in verify-edit.sh is unguarded. Removing it passes all 55 cases even though a UTF-8 file then exits 2 | open | Confirmed by mutation (mutant M2, 55 of 55 pass under LANG=C.UTF-8, and the mutant exits 2 on a file with a UTF-8 letter) | - |
| 20 | SHOULD-FIX | test_writefile.sh:184-188: the case "cleanup failure does not fail the write" cannot fail. A directory named like a backup is skipped by the regular-file guard at writefile.sh:113, so rm is never tried, and the case only greps the target content and never checks the exit status | open | Confirmed by mutation (mutant M3, which makes cleanup fatal, still passes 42 of 42) | - |
| 21 | NIT | speckit-converge.md:28: the command says to record the start time in the "Converge table" of metrics.md, but that table has no start-time column (templates/metrics.md:49). The run 2 start time was put in the Run cell | open | Confirmed by reading both files | - |
| 22 | NIT | speckit-converge.md:165: the Rules bullet "No repeat findings from previous rounds" is slightly at odds with the new clause at lines 107-109 that allows a fresh finding when a rejected repeat comes with new evidence | open | Confirmed by reading | - |
| 23 | NIT | test_writefile.sh:148: the with-backup directory refusal also passes without the new directory check, because the backup copy of a directory fails first; a symlink to a directory as TARGET is not tested | open | Read from code; case 12 no-backup (line 147) is the one that exercises the new check | - |
| 24 | NIT | verify-edit.sh:105, 113, 125: needle text is echoed to stderr unescaped, so control characters in a needle reach the terminal | open | Not reproduced | - |
| 25 | NIT | test_converge_yaml.sh:67-86: the text-presence checks depend on single-line phrasing; reflowing the prose would fail them | open | Known limitation of prompt tests, documented | - |

- **Findings count (verified):** BLOCKER 1, SHOULD-FIX 5, NIT 5. Raw reviewer counts: A SHOULD-FIX 2 / NIT 3, B BLOCKER 1 / SHOULD-FIX 4 / NIT 3. Two pairs were the same defect: the oversized count (A SHOULD-FIX, B BLOCKER) and the quadratic count (A NIT, B SHOULD-FIX).
- **Round is complete:** yes, both reviewers returned full reports.
- **Round is clean:** no (1 BLOCKER, 5 SHOULD-FIX).
- **Reviewer read-only check:** `git status --short` was the same before and after the round (metrics.md modified and findings.md untracked, both from the orchestrator's earlier bookkeeping).
- **Tokens:** reviewer A 45284, reviewer B 63407, total 108691. After A returned the total was 45284, under the cap. Reviewer B was already running in parallel, so the total passed the cap when B returned: 18691 over.
- **Wall-clock:** 8 minutes (12:20:29Z to 12:28:42Z, including the orchestrator's reproduction and mutation checks; the reviewers themselves ran 230 s and 338 s).
- **Status:** Aborted (Not Converged)
- **Consecutive clean count:** 0
- **Stop condition:** token cap (108691 of 90000 after round 1; rounds 1 of 4, minutes 8 of 15)
- **Open findings:** #15 to #25 of this run, plus the run 1 findings still open, at the time of the round. The dispositions in the table above are the ones recorded then; the final ones are in the Ledger.

---

## Ledger

Running list across all rounds and runs. The Round column is run.round. A finding already in this table is not raised again as new. Final dispositions, set at the T9 commit: 25 findings, 16 fixed, 9 deferred (accepted NITs, listed with a reason in notes.md under Accepted open findings), 0 rejected, 0 open.

| Round | Severity | Summary | Disposition | Reason | Repeat of |
|---|---|---|---|---|---|
| 1.1 | SHOULD-FIX | #1 writefile.sh stdin mode does not require --lines and --sig | fixed | Commit 799eab0; confirmed fixed by run 2 reviewer A | - |
| 1.1 | SHOULD-FIX | #2 converge command lacks cap validation, start-time record, per-round cap check | fixed | Commit 2d9f001; run 2 reviewer A found one gap, accepted as NIT #21 | - |
| 1.1 | SHOULD-FIX | #3 converge command lacks four spec edge cases | fixed | Commit 2d9f001; confirmed by run 2 reviewer A | - |
| 1.1 | SHOULD-FIX | #4 findings template brief options contradict the rotation | fixed | Commit 4050f45; confirmed by run 2 reviewer A | - |
| 1.1 | SHOULD-FIX | #5 backup cleanup narrower than spec | fixed | Commit 799eab0 widened it; its dotfile regression #16 is fixed in 31d7ff1 | - |
| 1.1 | SHOULD-FIX | #6 metrics.md per-ticket cells are prose, stray commit row | fixed | T9 bookkeeping, done in the T9 docs commit (docs(spec): record 002 verification and metrics) | - |
| 1.1 | SHOULD-FIX | #7 verify-edit.sh false landed when mktemp fails | fixed | Commit 19832c8 (no temp file); confirmed by run 2 reviewer A | - |
| 1.1 | SHOULD-FIX | #8 verify-edit.sh directory target false pass | fixed | Commit 19832c8; confirmed by run 2 reviewer A | - |
| 1.1 | SHOULD-FIX | #9 writefile.sh --no-backup writes into a directory TARGET | fixed | Commit 799eab0; confirmed by run 2 reviewer A | - |
| 1.1 | NIT | #10 stop-condition vocabulary differs between template and command | deferred | Accepted NIT: wording only; feature 003 | - |
| 1.1 | NIT | #11 writefile.sh awk check fails open | deferred | Accepted NIT: unlikely and minor; feature 003 | - |
| 1.1 | NIT | #12 cleanup deletes any matching name, not only its own | deferred | Accepted risk, stated in the writefile.sh header | - |
| 1.1 | NIT | #13 verify-edit.sh empty needle skipped | fixed | Commit 19832c8 (empty needle is a usage error); confirmed by run 2 reviewer A | - |
| 1.1 | NIT | #14 sync.sh chmod --reference is GNU-only | deferred | Accepted NIT: the repo runs on Linux only | - |
| 2.1 | BLOCKER | #15 verify-edit.sh oversized --count value exits 0 (false landed) | fixed | Commit 40e5fbe: more than 15 digits is a usage error, comparison written as ! -eq; the mutants that remove the cap, both defenses, or the zero stripping fail 3 cases each | - |
| 2.1 | SHOULD-FIX | #16 writefile.sh cleanup glob skips dotfile backups (regression of #5) | fixed | Commit 31d7ff1: second glob for dotfiles, cases 14; mutation-checked | 5 |
| 2.1 | SHOULD-FIX | #17 verify-edit.sh --count is quadratic, comment says linear | fixed | Commit 40e5fbe: header and comment now document the limit; the algorithm was left unchanged by decision (real files are about 21 KB) | - |
| 2.1 | SHOULD-FIX | #18 glob-literal matching for --contains and --absent untested | fixed | Commit 40e5fbe: cases 19; the glob mutants now fail 5, 4 and 3 cases | - |
| 2.1 | SHOULD-FIX | #19 no non-ASCII file test, LC_ALL export unguarded | fixed | Commit 40e5fbe: cases 20 (needs the C.UTF-8 locale); the mutant now fails 4 cases | - |
| 2.1 | SHOULD-FIX | #20 cleanup-failure test cannot fail | fixed | Commit 31d7ff1: cases 13 and 15 (stub rm on PATH, exit status asserted); the mutants fail | - |
| 2.1 | NIT | #21 converge command names a start-time column that does not exist | deferred | Accepted NIT: the time is recorded in the Run cell; feature 003 | - |
| 2.1 | NIT | #22 Rules bullet on repeats slightly at odds with the new-evidence clause | deferred | Accepted NIT: read as the exception; feature 003 with the re-rating clause | - |
| 2.1 | NIT | #23 with-backup directory refusal case does not exercise the new check | deferred | Accepted NIT: the no-backup case exercises it | - |
| 2.1 | NIT | #24 needle text echoed to stderr unescaped | deferred | Accepted NIT: terminal cosmetics only | - |
| 2.1 | NIT | #25 converge text checks depend on line wrapping | deferred | Accepted NIT: known limit of prompt tests, fails loudly | - |

Not in this ledger: commit 1dd360d closes a test gap found by scratch walkthrough S3 (the empty-content check of writefile.sh had no test that could fail). Scratch findings are recorded in notes.md, not here.
