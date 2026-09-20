# Notes: AI Workflow Enhancements

**Date:** 2026-09-20

---

## Q&A Log (Grill)

Not recorded here. The resolved questions are in spec.md "Open Questions".

## Decisions

- T1: Article 7.4 contains exactly (a), (b), (c) as specified, plus the pointer to ADR 0006. The
  rotation is documented in the ADR only. -- keeps 7.4 to the three points the spec success
  criteria name.
- T1 (revision, user-directed): 7.4(c) also states the accepted-findings exception: an Aborted run
  does not ship unless notes.md has an "Accepted open findings" section listing each open finding
  with a one-line reason. -- so a reader of the constitution alone knows an Aborted run can ship
  with written acceptance. Resolves the 7.3 tension below.
- T1: 7.4 (c) says a run that reaches a cap "ends Aborted, not Converged". -- states the spec's
  distinction so 7.3's "Converged (2 consecutive clean)" keeps its meaning.
- T1: constitution "Last updated" stays 2026-09-20. -- that is already today's date, so the line
  did not change. Only the version line changed (1.0 -> 1.1).
- T1: ADR "Deciders" is ssr0016 (the git user and spec author). -- the template asks for names.
- T1: 7.1 to 7.3 were not edited. Checked byte-for-byte against HEAD (checklist C2b).

## T2 Decisions

- T2: --lines N counts total payload lines, including the signature line. -- plan.md Design Decisions: the check runs on the whole temp file.
- T2: --from FILE derives --lines and --sig from the source file when they are omitted. -- matches plan.md: "With --from, they default to values derived from the source file."
- T2: ASCII gate rewritten to use awk for exact line:column. -- original grep approach could not compute the column reliably.
- T2: Test helpers split into check (expects success) and refuse (expects non-zero exit). -- needed for the "expected to fail" cases.
- T2: case1, case6, case7, case8 test fixtures fixed during TDD. -- real bugs in the test setup, not in writefile.sh.

## T3 Decisions

- T3: --count TEXT=N splits on the last =, so TEXT may contain = itself. -- safest parse; matches the "flat key" spirit of converge.yaml.
- T3: At least one condition is required, else exit 64. -- a verify with no condition is meaningless.
- T3: Used TMP_LIST=$(mktemp) + trap instead of process substitution to avoid any << sequence. -- the ticket requires grep -c << on the script to be 0; < <(...) would have matched.
- T3: expect_exit helper added to the test file for exact exit-code assertions. -- needed to prove 0/1/2/64 distinctly.
- T3: case6 test fixture had \\x27s typo and a stray duplicate printf; fixed via line-number patch. -- setup bug, not implementation.

## T5 Decisions

- T5: YAML regex tolerates a trailing comment on each key. -- ticket requires one-line comments; the strict `[[:space:]]*$` anchor would have failed the file with comments.
- T5: test hardens the token-divided-by-1000 computation with a guard. -- set -u + empty grep produced "unbound variable" before the yaml existed; guard keeps first run clean and informative.
- T5: values read from spec.md at test time. -- ticket wants a spec change to break the test; assert the current numbers from behavior 10 (4, 15, 90000).

## T4 Decisions

- T4: metrics.md template has a summary block at the end so two features can be diffed side by side (spec behavior 34). -- ticket asks for it explicitly.
- T4: findings.md combines round reports and ledger in one file. -- matches plan.md; a reviewer writes to findings.md and metrics.md only.
- T4: checklist.md template gains an evidence rule near the top. -- so a checked box means a command and a result, not a guess.
- T4: test_templates.sh written before reconciling the feature 002 metrics. -- TDD kept the reconciliation honest; caught tickets_closed mismatch.
- T4: feature 002 metrics reconciled: tickets_closed 4 -> 5 (T1-T5). -- keeps the running file consistent with the finished template.

## T6 Decisions

- T6: T6 content scope covers spec behaviors 1-14 (converge discipline); T7 covers 15-33 (claim verification, paste-safe writes). The tasks.md Files block originally listed only converge.md + 06-review.md, but the Content section named behaviors 15-20 and 25-33. Resolved by treating both as one session and correcting the Files block (net +3 lines, 334 total).
- T6: speckit-converge.md was rewritten in full (143 lines) rather than patched, because the caps/rotation/stop-condition content was absent entirely and the old text had no natural insert points.
- T6: 06-review.md rewritten with the four round briefs from spec behavior 2. Round 3 is "adversarial inputs and test quality" -- the spec is the only place this is stated; plan.md line 47 did not name it.
- T6: test_no_heredoc.sh scans top-level .specify/scripts/*.sh only, not the test directory. -- test fixtures embed the marker token as data on purpose (test_writefile.sh:41). Comment lines in the test script itself were reworded to avoid the literal token.
- T6: The base64-decode + writefile.sh --from route was used for all four command/prompt rewrites, not heredoc paste. -- the terminal corrupted multi-line heredoc pastes three times. Base64 has no quotes, backticks, or marker tokens for the terminal to misparse.
- T6: tasks.md T6 Files block corrected, not left as-is. -- without the fix, behaviors 25-33 had no file mapped to them.
- T6 (follow-ups, from converge run 1 and the T8 drift review): run 1 found three gaps in the T6 text. The converge command had no cap validation, start-time record or per-round cap check (finding 2), lacked four spec edge cases (finding 3), and the findings template contradicted the fixed rotation (finding 4); finding 10 (stop-condition wording) was deferred. Fixed in 2d9f001 and 4050f45. The drift between the repo command and `~/.claude/commands/` was resolved in T8: a stray `(` in the Empty range text and blank-line nits in 57bb7de, the imprecise "plus" in the Caps sentence became "then" in 53d9113, and the home copy was re-synced after approval three times (backups `*.bak.2026-09-20T11-19-25`, `T11-29-17`, `T12-17-26`). T6's own checks passed while these gaps existed because they test structure and phrases, not meaning.

## T7 Decisions

- T7: Claim verification, no-heredoc, and metrics-per-ticket rules were added to speckit-implement.md and 05-build.md. -- same rules as T6's implement scope; the tasks.md Content section for T6 named these behaviors even though the Files block did not.
- T7: Both test runners are named in speckit-implement.md (make test for Go, bash .specify/scripts/test/run.sh for workflow). -- T6/T7 are workflow tickets, so make test alone would not cover the changed files.
- T7: Metrics updates land "when the ticket closes", matching spec behavior 33 ("updated when each event happens"). -- "for this ticket" was ambiguous about timing.
- T7 (follow-up): the drift on speckit-implement.md was resolved in T8 (a blank-line nit fixed in 57bb7de, home copy re-synced after approval, backup `speckit-implement.md.bak.2026-09-20T11-19-25`). The T7 claim-verification text relies on verify-edit.sh, which converge run 1 found could report "landed" without verifying (findings 7, 8, 13). It was rewritten to fail closed in 19832c8 and hardened in 40e5fbe, so the tool the T7 text points to now does what the text assumes.

## T8 Decisions

- T8: sync.sh syncs exactly 4 files (writefile.sh, verify-edit.sh, sync.sh, converge.yaml), flat into `$SPEC_KIT_HOME` (default `~/.spec-kit`). `.claude/commands/` is NOT part of sync.sh. -- user decision; the command copies are not part of `~/.spec-kit/` and adding them goes beyond the T8 interface in tasks.md.
- T8: The installed `~/.spec-kit/sync.sh` is a repo-checkout tool, not a runtime tool. It finds the repo root from its own location (`../..`), so run from `~/.spec-kit/` it exits 2 with "repo source not readable ... (run from a repo checkout)". -- it is installed only so the set is complete; `check` and `install` are always run from the repo. A `SPEC_KIT_REPO` override was not added (outside the T8 interface).
- T8: `install --force` has no 7-day backup cleanup, unlike writefile.sh. Backups (`<name>.bak.YYYY-MM-DDTHH-MM-SS`) accumulate in `~/.spec-kit/`. -- not required by T8, and forcing an overwrite of `~/.spec-kit/` is rare.
- T8: Test case 7b (unreadable home file exits 2) relies on `chmod 000` being unreadable, which does not hold for root. -- root caveat; the test is meant for a normal user, as in this repo's CI.
- T8: A second backup in the same second overwrites the first, because the name has only second resolution. -- inherited from writefile.sh (`cp` to the same name); not a T8 concern, and any fix is feature 003 scope.
- T8: install applies in three phases: stage every copy to a temp file in the home directory, back up every differing file, then `mv` each into place. A failure while staging or backing up leaves the home directory unchanged; only a failing `mv` mid-way could leave a partial install. -- no mid-`mv` test, because that failure is hard to simulate; user accepted.
- T8: Drift on `speckit-converge.md` and `speckit-implement.md` was resolved by a manual, approved copy (`writefile.sh --from`, backups `*.bak.2026-09-20T11-19-25`) after the diffs were shown. Before copying, a stray `(` in the converge "Empty range" text and two blank-line nits were fixed in the repo copies, landed as the separate commit 57bb7de (`fix(workflow): repair converge typo and tidy command spacing`), not folded into T8 (7a091a0).
- T8: The `(empty` typo is probably prose damage from terminal paste corruption (cause not proven); the T6 checks did not catch it because they test structure, not wording.
- T8: writefile.sh creates the target with mode 600 (from `mktemp`) and does not preserve the mode of an existing target when overwriting. The two home command files came out 600 and were set to 664 by hand. -- known limitation for feature 003; writefile.sh was not changed in T8.


## T9 Execution Notes

T9 started 2026-09-20T11:27:37Z, in the same conversation as T8 (see Process deviation below). The fix commits made during T9 are 57bb7de, 53d9113, 19832c8, 799eab0, 2d9f001, 4050f45, 40e5fbe and 31d7ff1, plus 1dd360d (test only, from walkthrough S3); T8 is 7a091a0.

The T9 wall-clock in metrics.md (124 minutes) runs from that recorded start to a measurement at 13:31:25Z, just before the T9 commit; it includes the time spent waiting for review. Tokens for T9 are "not tracked" (the session's own usage is not available); the converge tokens are in the Converge tables. The feature Status in metrics.md is "Aborted", the converge status, and not "done", because the template's Status list has no "done"; it becomes "Shipped" after the push.

### Runs and cost

| Run | Range | Diff lines | Reviewer tokens | Total tokens | Wall-clock | Stop |
|---|---|---|---|---|---|---|
| 1 (real) | b92af87..53d9113 | 3094 | A 71397, B 45033 | 116430 | 4 min | token cap |
| 2 (real, fresh caps) | 53d9113..4050f45 | 626 | A 45284, B 63407 | 108691 | 8 min | token cap |
| B2 (scratch, max_rounds 1) | e5dde30^..e5dde30 | 250 | A 29760, B 27001 | 56761 | 110 s | round cap |
| B3 (scratch, feature 999-greet) | 33c68aa..06a57a4 | 50 | R1 A 14737 and B 15803, R2 17578, R3 17312, R4 15860 | 81290 | 322 s | round cap |
| S3 (scratch, max_minutes 1) | 799eab0^..799eab0 | 178 | A 27957, B 36799 | 64756 | 188 s | minutes cap |

- Run 1 found 9 SHOULD-FIX and 5 NIT. Run 2 found 1 BLOCKER, 5 SHOULD-FIX and 5 NIT. Every finding was reproduced or re-read by the orchestrator before it was recorded. Findings, ledger and per-round detail are in findings.md.
- Walkthrough B1 (empty range HEAD..HEAD on the real repo) stopped without "Converged" and asked for an explicit range. Walkthrough B2 exercised the Accepted-open-findings gate three ways with a scratch script: no section (not allowed), an incomplete section (not allowed, missing findings named), a complete section (allowed). Walkthrough B3 planted a defect in a scratch repo: the clean count went 1, then 0 (a fix that deleted `set -u` produced a SHOULD-FIX), then 1 (a NIT-only round is clean), then 0 (round 4, see Spec ambiguity).
- Walkthrough S3 closed the last unobserved cap. In a scratch clone of 31d7ff1 with `max_minutes: 1`, reviewer A returned after 87 s of its own run (the orchestrator's check ran at 116 s), against a cap of 60 s, with 27957 of 90000 tokens spent. The minutes cap therefore fired first, while reviewer B was still running. Following the command text, the round was finished as far as it could be reported, marked incomplete and not counted as clean; status Aborted, clean count 0. Reviewer B added 36799 tokens (64756 in all, still under the token cap).
- Scratch runs (B2, B3, S3) used 202807 tokens and are not counted toward the real cap (user decision). The two real runs used 225121.
- All 13 reviewer runs (4 real, 2 in B2, 5 in B3, 2 in S3) left `git status` unchanged before the orchestrator wrote anything. The reviewers were Plan agents, which have no Edit or Write tool. One S3 reviewer created and removed a stray file in /tmp, outside the repository.
- S3 findings (scratch, not in the real ledger). The S3 reviewers examined the tree at commit 799eab0 and each reported 4 SHOULD-FIX and 3 NIT. Most are already known or fixed later: dotfile backups not pruned (fixed in 31d7ff1), the cleanup-failure test that cannot fail (fixed in 31d7ff1), the awk fail-open (finding 11), cleanup deleting any matching name (finding 12, accepted), the mode 600 (candidate 8) and same-second backups (candidate 9). New, and checked by the orchestrator: (a) a target whose name starts with a dash fails in `dirname` (`dirname: invalid option`; `./-x.txt` works); (b) an overwritten symlink target is replaced by a regular file and the linked file is not updated; (c) case 2 of test_writefile.sh cannot fail on the empty-content check: with that check disabled (condition replaced by `false`, `die` line kept) the suite still passed 56 of 56, because the case still refused on a signature mismatch, and an empty `--from` file would have been written as an empty file with exit 0; fixed in 1dd360d (case 2b tests the empty `--from` file and the refusal message, and that mutant now fails 7 cases); (d) case 9 (unwritable backup location) never reaches the backup-failure branch because mktemp fails first, and it fails when run as root; (e) plan.md line 132 promises distinct backups for two writes in one second, but case 8 sleeps 1.1 s between the writes so it cannot test that, and same-second backups still overwrite each other; (f) a last line over about 128 KB in `--from` mode fails with a misleading "signature mismatch".

### Cap tuning data (for feature 003)

- The diff shrank 4.9 times (3094 to 626 lines) and the round-1 cost fell only 6.6 percent (116430 to 108691 tokens). The fixed overhead of a reviewer, which is reading the spec, plan and tasks and running checks, dominates. Diff size is not the cost driver.
- Cost also follows what a reviewer is asked to try. Run 2 reviewer B, asked to check whether every new test can fail and to attack the scripts, cost 63407 tokens on a 626-line diff. A single reviewer on a 50-line diff cost 14737 to 17578 tokens in B3.
- Round 1 runs two reviewers in parallel and the cap is checked after each returns. After the first reviewer returned, both real runs were still under the cap, but the second reviewer was already running, so the total passed the cap when it returned: 26430 over in run 1 and 18691 over in run 2. The command now says to report the overshoot (2d9f001).
- The 90000 cap and converge.yaml were left unchanged in T9 (user decision). Both real runs stopped after round 1 of 4, so neither showed what a round 2 to 4 costs; only B3 measured single-reviewer rounds, on a tiny feature.

### Self-correction: the T2 refusal tests tested nothing

While writing the red run for run 1 finding 1, only 2 of the 11 expected cases failed. The refuse cases in test_writefile.sh ran writefile.sh inside `bash -c` with escaped variables (`$WF`, `$target`, and `$cyr` in case 5). None of them was exported, so the inner command was empty (exit 127) and every refusal passed trivially, without testing writefile.sh.

- Scope, checked with git: 6 refusal cases were vacuous from T2 (commit f99d53f) until T9: cases 2, 3, 4, 5, 9 and 10 (test_writefile.sh at 53d9113, lines 55, 60, 65, 70, 103, 109). Five more that I wrote in T9 in the same style were vacuous too, and were found before they were committed. Earlier in this session I reported "11 since T2"; that was wrong, 11 counts both groups.
- Consequence: the T2 checklist line "run.sh passes" and every later "all tests pass" were true but said nothing about the refusal paths of writefile.sh until commit 799eab0. The refusals themselves were correct: with the variables exported, all six original cases pass for real.
- Fix (799eab0): export WF, tmp, target and cyr, and make `refuse` fail on exit 126 or 127. Shown: with the export lines removed, the same file now fails loudly ("command could not run, exit 127").
- Found by writing a red run for a fix and seeing that the expected failures did not appear. Run 1's reviewers did not test the tests. Run 2 reviewer B was told to check that every new test can fail, which led to the next section.

### Mutation testing

Method: copy the scripts to a temp tree, put one named fault into the copy with sed, run the suite against it, and expect at least one case to fail. A mutant that passes the whole suite is a test gap. Used in commits 40e5fbe (verify-edit.sh), 31d7ff1 and 1dd360d (writefile.sh); 19 mutants in all (8 for verify-edit.sh, 11 for writefile.sh).

- Before the tests were extended, four mutants passed the whole suite: `--contains` matching as a glob (55 of 55 cases passed), `--absent` matching as a glob (55 of 55), the `LC_ALL=C` export removed (55 of 55), and writefile.sh cleanup made fatal (42 of 42). After the new cases every one of the 19 mutants fails at least one case, and 18 of them fail two or more. The one exception is the mutant that only removes `|| true` from the cleanup's `rm`, which fails exactly one case.
- One survivor was found that no reviewer had raised: dropping the strict timestamp pattern from the writefile.sh cleanup still passed 53 of 53, because the age comparison hid it. A name such as `x.bak.2000-01-01` sorts before the cutoff and would have been deleted. Cases 13 (three look-alike names) now protect the pattern, and both that mutant and a looser pattern fail.
- Mutation testing is not part of run.sh. It was done by hand with sed. See Feature 003 Candidates.

### Other self-corrections

- T1 wall-clock: recorded as 3, measured 134 s = 2.2 minutes; metrics.md is corrected. The C5 subagent of T1 used 33366 tokens; the session's own tokens were not tracked, so metrics.md says "not tracked" and the partial figure lives here.
- Walkthrough B2: I rejected the claim that an empty TEXT is silently skipped, because `--contains ""` alone exits 64. That was too broad: run 1 reproduced it when another needle is present (`--contains aaa --absent ""` exits 0, finding 13, fixed in 19832c8).
- Line numbers: reviewers of run 1 and of B2 cited lines that do not exist (verify-edit.sh has 118 lines; they cited up to 211). Every number in findings.md is the orchestrator's `grep -n` result. When the brief required numbers checked with `cat -n` or `grep -n` (B3 and run 2), they matched.
- My own regression: the first rewrite of `--count` copied the string once per occurrence (58 s for 40000 occurrences in 450 KB). A timing case written first (red) caught it before commit 19832c8. The one-pass version is still quadratic in file size; the limit is documented in the script header (40e5fbe).
- Checks of mine that were wrong and were caught by reading their output: `--absent plus` (matches an unrelated "plus a ledger table"), `--absent 'pending | pending'` (matches the T9 row), a diff filter that hid lines starting with a bullet, a findings table cell with two pipe characters, and a hand-counted expected value in a scratch check. None reached a commit.
- One shell call of mine contained a multi-line heredoc (a no-op). It changed nothing, but it goes against the workflow's own rule.

### Environment limit: UTF-8 locale

Case 20 in test_verify_edit.sh runs verify-edit.sh with `LANG=C.UTF-8`. It needs that locale (`locale -a` lists C.utf8 here). Where the locale is missing, the shell falls back to the C locale, the cases pass, and the mutant that removes the `LC_ALL=C` export would go unseen. Feature 003: check that the locale exists, or document the fallback.

### Process deviation: no fresh context for T9

T9 began by pasting a resume message into the conversation that had run T8. No /clear was issued, so Article 7.2 (fresh context per ticket) was not followed for T9. metrics.md keeps Context clears at 3 (T1, T6, T8). At the end of T8 the agent recommended a fresh context for T9; the user continued in the same one. The effect was not measured.

### Spec ambiguity: re-rating a deferred finding

In B3 round 4 a context-free reviewer rated a deferred NIT (the empty-NAME test checks only the exit code) as SHOULD-FIX. The written rules say a repeat of a ledger finding is marked and ignored, but they do not say what to do when the repeat carries a higher severity. Decision (user): a re-rated finding is new information, not a repeat, so the round was not clean and the run ended Aborted at the round cap. Had the repeat been ignored, the round would have been clean and the count 2 (Converged). The converge command does not say this yet; a re-rating clause was deliberately left out of 2d9f001 and is a feature 003 candidate.

## Blockers

- None.

## Context Clears

| # | Ticket | Reason | Notes |
|---|---|---|---|
| 1 | T1 | Start of ticket | /clear before T1, per Article 7.2 |
| 2 | T6 | Start of ticket | /clear before T6, per Article 7.2; T6 and T7 advanced together |
| 3 | T8 | Start of ticket | /clear before T8, per Article 7.2 |

T9 note: no /clear was issued at the start of T9, which ran in the same context as T8. The count stays 3. See "Process deviation" in T9 Execution Notes.

## ADRs Created

- docs/adr/0006-bounded-converge.md (T1, commit e13f2aa)

## Known Issues

- RESOLVED (7.4(c) revised, see Decisions). Was: Article 7.3 lists
  "Converged (2 consecutive clean)" under Before Shipping, while ADR 0006 (from spec behavior 13)
  lets the user ship after an Aborted run with a written "Accepted open findings" section. The ADR
  says this does not make the run Converged. 7.4 (c) says Aborted is not Converged and points to the
  ADR, but it does not itself state the accepted-findings exception. See the T1 hand-off for the
  choice offered to the user.

- C5 fresh-agent read (verdict NO CONTRADICTION) listed ambiguities in 7.4, none a contradiction:
  1. 7.4(c) names only `.specify/converge.yaml`; the ADR adds the `~/.spec-kit/` fallback and defaults.
  2. (Fixed after this read: 7.4(c) now states the accepted-findings rule.) 7.4(c) did not say what Aborted means for shipping.
  3. 7.4(a) "author fixes between rounds" lacks the ADR's "inside the scope of the change" limit.
  4. 7.4(c) does not say whether a cap is checked before or after a round, or whether landing exactly on the cap counts as reaching it. (Since 2d9f001 the converge command checks the caps before each round and each time a reviewer returns; 7.4(c) itself is unchanged. Landing exactly on a cap is still not stated.)

- RESOLVED in T8 and T9 (commits 57bb7de, 7a091a0, 53d9113, 2d9f001): Drift: `.claude/commands/speckit-converge.md` and `.claude/commands/speckit-implement.md` differed from their `~/.claude/commands/` copies after T6. Re-sync was done only after user approval, by a manual copy; `diff -q` is now silent for all four command files. See T8 Decisions.

- RESOLVED in T9 (commit 53d9113, pinned by case 4 of test_converge_yaml.sh). Was: the converge command text, Caps section, says "Caps come from `.specify/converge.yaml` plus `~/.spec-kit/converge.yaml`". Spec behavior 12 and ADR 0006 mean the order "repo file, then home file, then built-in defaults", so "plus" should read "then". Not a contradiction, only imprecise. To be addressed in the T9 walkthrough; deliberately left unchanged in T8. (The word will also need re-syncing to `~/.claude/commands/` when it changes.)

- Feature 003 candidates: see the Feature 003 Candidates section at the end of this file.


## Deferred SHOULD-FIX and NIT

No SHOULD-FIX is deferred. All 14 SHOULD-FIX and the 1 BLOCKER found by the two real converge runs are fixed (run 1 findings 1 to 9, run 2 findings 15 to 20; see the findings.md ledger for the commit of each), and so is NIT 13. Nine NITs are not fixed and are deferred or accepted: run 1 findings 10, 11, 12 and 14, and run 2 findings 21 to 25. Each is listed with its reason under Accepted open findings.

## Accepted open findings

Converge run 1 and run 2 both ended Aborted at the token cap (116430 and 108691 tokens against 90000). No run reached Converged. Under spec behavior 13, shipping after an Aborted run needs this section, which lists every non-fixed finding with a one-line reason. The fixes made after run 2 (40e5fbe and 31d7ff1) were checked by their own tests and by mutation testing, not by a third converge run (user decision: the double-Aborted pattern shows a third full run would Abort too).

Fixed and therefore not listed: 14 SHOULD-FIX, 1 BLOCKER and 1 NIT. Nine NITs remain. A NIT never makes a round unclean (behavior 4), so none of them blocks shipping.

- #10: the stop-condition wording differs between the findings template and the converge command; wording only, no effect on behavior, feature 003.
- #11: the awk ASCII check in writefile.sh falls back to "clean" if awk itself fails; unlikely and minor, and the empty-content, line-count and signature checks still run; feature 003.
- #12: writefile.sh cleanup deletes any `*.bak.<timestamp>` file older than 7 days in the target directory, whether or not it made it; accepted risk, stated in the script header.
- #14: sync.sh uses GNU `chmod --reference`; this repo runs on Linux only.
- #21: the converge command says the start time goes in the "Converge table" but the template has no start-time column; the time is recorded in the Run cell and nothing depends on it, so the column is a feature 003 change.
- #22: the Rules bullet "No repeat findings from previous rounds" reads slightly against the new-evidence clause; the clause is more specific and reads as the exception, to be reworded together with the re-rating clause in feature 003.
- #23: the refusal test for a directory TARGET with backups on also passes without the new directory check, because the backup copy of a directory fails first; the no-backup case does exercise the check, and a symlink to a directory was tried by hand (refused, nothing written).
- #24: verify-edit.sh echoes the needle text to stderr without escaping; terminal cosmetics only, with no security effect for a local tool that reads files the user names.
- #25: the converge text tests depend on single-line phrasing, so a reflow of the prose fails them; this is a known limit of prompt tests and it fails loudly, not silently.

## Feature 003 Candidates

Collected during T9 and T8. None of these was changed in feature 002.

1. Mutation testing integration: make the mutation check a repeatable step (a small list of named faults per tool) for every ticket that adds tests, instead of ad hoc sed runs. It exposed four passing mutants and one unseen gap in T9.
2. UTF-8 availability check: verify that `C.UTF-8` exists before case 20 of test_verify_edit.sh, or document the fallback.
3. Double-Aborted tuning data: run 1 cost 116430 tokens on 3094 lines and run 2 cost 108691 on 626 lines; the fixed overhead per reviewer dominates. Use this evidence to tune max_tokens or the reviewer budget.
4. Vacuous-test audit: audit every `bash -c` case in all test files for missing exports and for passes caused by exit 126 or 127.
5. Re-rating clause: state in the converge command what to do when a context-free reviewer rates a deferred finding higher (T9 decision: a re-rated finding is new information, not a repeat).
6. Reviewer line numbers are unreliable unless the brief demands numbers checked with `cat -n` or `grep -n`; keep that rule in the review prompt and have the orchestrator re-check the numbers.
7. `sync.sh --force` should prune old backups, or the accumulation should be documented as accepted (`~/.spec-kit/` holds four `.bak` files after T9).
8. writefile.sh does not preserve the mode of the target (mktemp gives 600, so a 755 script loses its exec bit) and replaces a symlink target with a regular file: set the mode after the move and resolve symlinks first.
9. Backups made in the same second overwrite each other (writefile.sh and `sync.sh --force`). plan.md line 132 promises distinct backups for two writes in one second, and case 8 of test_writefile.sh sleeps 1.1 s, so it cannot fail on that promise.
10. Markdown lint for prose typos: the stray `(` in the T6 text passed every check.
11. Feature 003 spec: extract the workflow (scripts, templates, commands) into a separate repository.
12. 06-review.md still opens with "senior Go backend engineer" and lists Go-specific review items; make the prompt aware of the project type.
13. Parallel reviewers overshoot the token cap (26430 and 18691 over); consider sequential reviewers in round 1 or a per-reviewer budget.
14. Reduce the fixed overhead per reviewer: give a spec summary instead of the full spec, since diff size was not the cost driver.
15. The `timeout 8` timing guard in case 16 of test_verify_edit.sh may flake on a slow CI machine; adjust it with evidence.
16. Failure injection with a stub on PATH (as in case 15 of test_writefile.sh) as the standard pattern for tests of error paths.
17. One shared `refuse` helper (and `check`, `expect_exit`) that rejects exit 126 and 127, used by every test file.
18. Test case 9 of test_writefile.sh cannot fail on what it names: it never reaches the backup-failure branch (mktemp fails first) and it fails when run as root. Found by walkthrough S3 (see the S3 findings in Runs and cost); fix it with the same mutation check used in 40e5fbe, 31d7ff1 and 1dd360d. Case 2, the other case found there, was fixed in 1dd360d.
19. writefile.sh edge cases from S3: a target starting with a dash fails in `dirname` (use `dirname --` and a `./` mktemp template), a last line over about 128 KB gives a misleading "signature mismatch", and a non-numeric `--lines` prints a stray shell error.
