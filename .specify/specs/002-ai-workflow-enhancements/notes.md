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

## Blockers

- None.

## Context Clears

| # | Ticket | Reason | Notes |
|---|---|---|---|
| 1 | T1 | Start of ticket | /clear before T1, per Article 7.2 |

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

## Deferred SHOULD-FIX and NIT

- None yet (no converge round has run).
