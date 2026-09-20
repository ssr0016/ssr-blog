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

## T7 Decisions

- T7: Claim verification, no-heredoc, and metrics-per-ticket rules were added to speckit-implement.md and 05-build.md. -- same rules as T6's implement scope; the tasks.md Content section for T6 named these behaviors even though the Files block did not.
- T7: Both test runners are named in speckit-implement.md (make test for Go, bash .specify/scripts/test/run.sh for workflow). -- T6/T7 are workflow tickets, so make test alone would not cover the changed files.
- T7: Metrics updates land "when the ticket closes", matching spec behavior 33 ("updated when each event happens"). -- "for this ticket" was ambiguous about timing.

## T8 Decisions

- T8: sync.sh syncs exactly 4 files (writefile.sh, verify-edit.sh, sync.sh, converge.yaml), flat into `$SPEC_KIT_HOME` (default `~/.spec-kit`). `.claude/commands/` is NOT part of sync.sh. -- user decision; the command copies are not part of `~/.spec-kit/` and adding them goes beyond the T8 interface in tasks.md.
- T8: The installed `~/.spec-kit/sync.sh` is a repo-checkout tool, not a runtime tool. It finds the repo root from its own location (`../..`), so run from `~/.spec-kit/` it exits 2 with "repo source not readable ... (run from a repo checkout)". -- it is installed only so the set is complete; `check` and `install` are always run from the repo. A `SPEC_KIT_REPO` override was not added (outside the T8 interface).
- T8: `install --force` has no 7-day backup cleanup, unlike writefile.sh. Backups (`<name>.bak.YYYY-MM-DDTHH-MM-SS`) accumulate in `~/.spec-kit/`. -- not required by T8, and forcing an overwrite of `~/.spec-kit/` is rare.
- T8: Test case 7b (unreadable home file exits 2) relies on `chmod 000` being unreadable, which does not hold for root. -- root caveat; the test is meant for a normal user, as in this repo's CI.
- T8: A second backup in the same second overwrites the first, because the name has only second resolution. -- inherited from writefile.sh (`cp` to the same name); not a T8 concern, and any fix is feature 003 scope.
- T8: install applies in three phases: stage every copy to a temp file in the home directory, back up every differing file, then `mv` each into place. A failure while staging or backing up leaves the home directory unchanged; only a failing `mv` mid-way could leave a partial install. -- no mid-`mv` test, because that failure is hard to simulate; user accepted.
- T8: Drift on `speckit-converge.md` and `speckit-implement.md` was resolved by a manual, approved copy (`writefile.sh --from`, backups `*.bak.2026-09-20T11-19-25`) after the diffs were shown. Before copying, a stray `(` in the converge "Empty range" text and two blank-line nits were fixed in the repo copies, to land as a separate `fix(workflow)` commit (pending), not folded into T8.
- T8: The `(empty` typo is probably prose damage from terminal paste corruption (cause not proven); the T6 checks did not catch it because they test structure, not wording.
- T8: writefile.sh creates the target with mode 600 (from `mktemp`) and does not preserve the mode of an existing target when overwriting. The two home command files came out 600 and were set to 664 by hand. -- known limitation for feature 003; writefile.sh was not changed in T8.


## Blockers

- None.

## Context Clears

| # | Ticket | Reason | Notes |
|---|---|---|---|
| 1 | T1 | Start of ticket | /clear before T1, per Article 7.2 |
| 2 | T6 | Start of ticket | /clear before T6, per Article 7.2; T6 and T7 advanced together |
| 3 | T8 | Start of ticket | /clear before T8, per Article 7.2 |

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
  4. 7.4(c) does not say whether a cap is checked before or after a round, or whether landing exactly on the cap counts as reaching it.

- RESOLVED in T8 (commit reference pending): Drift: `.claude/commands/speckit-converge.md` and `.claude/commands/speckit-implement.md` differed from their `~/.claude/commands/` copies after T6. Re-sync was done only after user approval, by a manual copy; `diff -q` is now silent for all four command files. See T8 Decisions.

- T9 cleanup: the converge command text, Caps section, says "Caps come from `.specify/converge.yaml` plus `~/.spec-kit/converge.yaml`". Spec behavior 12 and ADR 0006 mean the order "repo file, then home file, then built-in defaults", so "plus" should read "then". Not a contradiction, only imprecise. To be addressed in the T9 walkthrough; deliberately left unchanged in T8. (The word will also need re-syncing to `~/.claude/commands/` when it changes.)

- Feature 003 candidates (not T8): writefile.sh does not preserve the target's mode (mktemp gives 600); same-second backups overwrite each other (also true of sync.sh --force).


## Deferred SHOULD-FIX and NIT

- None yet (no converge round has run).
