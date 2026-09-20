# Tasks: AI Workflow Enhancements

**Plan:** ./plan.md
**Date:** 2026-09-20
**Base commit:** b92af87 (HEAD when this feature started; used by the "workflow only" guard)

---

## DAG

Edges (`A -> B` means B is blocked by A):

```
T1 -> T6
T4 -> T6
T5 -> T6
T2 -> T7
T3 -> T7
T2 -> T8
T3 -> T8
T5 -> T8
T6 -> T9
T7 -> T9
T8 -> T9
```

Levels:

```
level 0 (parallel): T1  T2  T3  T4  T5
level 1 (parallel): T6  T7  T8
level 2:            T9
```

| Ticket | Slice | Type | blocked_by | Parallel with | RBAC-sensitive |
|---|---|---|---|---|---|
| T1 | S1 | Governance | none | T2, T3, T4, T5 | no |
| T2 | S2 | Tooling (bash) | none | T1, T3, T4, T5 | no |
| T3 | S3 | Tooling (bash) | none | T1, T2, T4, T5 | no |
| T4 | S4 | Templates | none | T1, T2, T3, T5 | no |
| T5 | S5 | Config | none | T1, T2, T3, T4 | no |
| T6 | S6 | Workflow text, **workflow-change** | T1, T4, T5 | T7, T8 | no |
| T7 | S7 | Workflow text, **workflow-change** | T2, T3 | T6, T8 | no |
| T8 | S8 | Tooling (bash) | T2, T3, T5 | T6, T7 | no |
| T9 | S9 | Verification | T6, T7, T8 | none | no |

RBAC-sensitive: none. This feature does not touch the blog app, so the Auth/RBAC checklist and
extra reviewer do not apply.

**Recommended order: T1, T2, T3, T4, T5, T6, T7, T8, T9.** It is a valid topological order. Every
ticket updates `checklist.md`, `notes.md`, and `metrics.md` in this spec directory, so running
tickets in parallel needs separate worktrees and a rebase per merge. Serial avoids that.

**Deviation from your tagging:** T8 is tagged Tooling, not workflow-change. It adds `sync.sh` and
installs copies into `~/.spec-kit/`. It does not edit `.claude/commands/` or `.ai-workflow/prompts/`,
which are what workflow-change covers. It has its own rule below (home-directory writes need approval).

---

## Rules for every ticket

- **Fresh context.** One ticket, one context. Clear before starting the next (Article 7.2). Stay under 100K tokens.
- **Test first.** Write the checks for the ticket before the change, watch them fail (or, for text-only tickets, write the grep and read checks into `checklist.md` first), then make the change.
- **Write files with the agent's file tool, never terminal paste.** No multi-line heredocs anywhere. Once T2 lands, `writefile.sh` is available for terminal use.
- **Verify before claiming.** After any scripted edit, confirm it landed before writing a claim into `notes.md` or `checklist.md`. Before T3 lands, use `grep -F` or re-read the file. After T3, use `verify-edit.sh`. A checked item in `checklist.md` cites the command and result that proved it.
- **No new dependencies.** Only bash, coreutils, grep, awk, sha256sum.
- **ASCII only** in every file this feature creates, except runtime-built fixtures inside tests.
- **Before claiming done** (AGENTS.md):
  1. `make lint` -- zero warnings.
  2. `make test` -- all pass. It cannot be affected by this feature, so it acts as a regression guard.
  3. `bash -n` on every script the ticket touched, and the ticket's own tests pass.
  4. Workflow-only guard, expecting no output:
     `{ git diff --name-only b92af87; git ls-files --others --exclude-standard; } | grep -E '^(internal|pkg|cmd)/|^docs/(API\.md|swagger|docs\.go|ARCHITECTURE\.md)|^(Makefile|AGENTS\.md)$|^\.github/|^\.specify/specs/001-'`
  5. `diff -rq .ai-workflow/templates .specify/templates` -- identical.
  6. Update `checklist.md` and `notes.md` (create from `.ai-workflow/templates/` on first use).
  7. Update `metrics.md`: this ticket's wall-clock minutes, tokens (the session's reported usage, or "not tracked"), whether 100K was crossed, and the reason for the context clear.
- Not applicable to this feature: integration tests, `make swagger`, migrations, the auth smoke test.

**Workflow-change rules** (T6 and T7; defined here because AGENTS.md has no such type, analogous to
its API Change ticket):

- [ ] Every changed behavior is mapped to the spec behavior numbers it implements, in `checklist.md`.
- [ ] A fresh agent reads the changed text against those behaviors and against constitution Articles 7.3 and 7.4 and finds no contradiction.
- [ ] No fenced code block in a changed command or prompt contains a multi-line heredoc.
- [ ] If the ticket changes `.claude/commands/`, compare with `~/.claude/commands/` using `diff` and report drift. Re-syncing the home copy needs your approval and is never done silently.
- [ ] Template directories are still identical.

**Bootstrap note.** `metrics.md` for this feature must exist from the first ticket, but its
template is built in T4. Whichever ticket runs first creates `metrics.md` by hand from spec
behaviors 29-31. T4 then checks that file against the finished template and reconciles any difference.

**Commit gate.** Commits are made only when you ask. Before T1, the spec, plan, and tasks files
(currently untracked) are committed as `docs(spec): add 002 ai workflow enhancements spec, plan, tasks`.
Attribution trailer on every commit: `Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>`.

---

## T1: Governance -- ADR 0006 and constitution Article 7.4

**Type:** Governance
**blocked_by:** none
**Parallel:** T2, T3, T4, T5

**Files:**
- `docs/adr/0006-bounded-converge.md` (new)
- `.specify/memory/constitution.md` (add Article 7.4 after 7.3; version 1.0 -> 1.1; update the date)

**Content:**
- ADR (English, using `.ai-workflow/templates/adr.md`): Status Accepted. Context is the 001 converge run (6 rounds, about an hour, 114.9K tokens, no defined "clean"). Decision covers the caps (4 rounds, 15 minutes, 90K tokens, configurable), the read-only reviewer, the fixed four-round rotation, the unchanged two-consecutive-clean rule, and the accepted-findings path after an Aborted run. Alternatives considered: no caps (rejected: this is what 001 did), a program-enforced runner (rejected: out of scope, needs new code), a second reviewer model (rejected: not required). Negative consequences include that caps are enforced by the command text, not by code.
- Article 7.4 "Converge": (a) the reviewer is read-only apart from the findings and metrics artifacts, (b) the two-consecutive-clean rule is unchanged, (c) converge has caps set in `.specify/converge.yaml`. Points to ADR 0006. Articles 7.1 to 7.3 are not edited.

**Test (written first, in `checklist.md`):**
- `grep -n "7.4" .specify/memory/constitution.md` finds the new article after 7.3 and before the Amendments section.
- `git diff` on the constitution shows only additions plus the version and date lines. Nothing in 7.1 to 7.3 changed.
- 7.4 states all three of (a), (b), (c), and names ADR 0006 and `converge.yaml`.
- ADR has every section from the template and is numbered 0006 with no gap in `docs/adr/`.
- A fresh agent reads 7.3 and 7.4 together and reports no contradiction.

**Commit:** `docs(constitution): add Article 7.4 converge`
**Done when:** the checks above pass; ADR and constitution are in the same commit (the amendment rule requires both); no other constitution text changed.

## T2: Reliable write -- ignore rule and `writefile.sh`

**Type:** Tooling (bash)
**blocked_by:** none
**Parallel:** T1, T3, T4, T5

**Files:**
- `.specify/specs/.gitignore` (new; one line, `*.bak.*`)
- `.specify/scripts/writefile.sh` (new)
- `.specify/scripts/test/test_writefile.sh` (new; self-contained assert helpers)
- `.specify/scripts/test/test_ignore_rule.sh` (new)
- `.specify/scripts/test/run.sh` (new; runs every `test_*.sh`, exits non-zero if any fail)

**Interface:** `writefile.sh TARGET --lines N --sig 'text' [--sha256 HEX] [--no-backup] [--allow-utf8] < content`, or `--from FILE` instead of stdin. `--lines` and `--sig` are required with stdin. Exit 0 on success. Non-zero with a message saying what differed on any failure. See plan.md "Design Decisions" for the write procedure, ASCII gate, backup naming, and cleanup rules.

**Test (written first; every case fails before the script exists):**
- Content with single and double quotes, backticks, dollar signs, and lines that look like heredoc terminators is written byte-identical (compare with `cmp`).
- Empty content is refused and no file is created.
- Missing signature line, and wrong `--lines`, each exit non-zero and leave the target unchanged.
- A Cyrillic look-alike letter, built with a printf byte escape so the test file stays ASCII, is rejected with its `line:column`. With `--allow-utf8` the same content is written intact.
- Overwriting creates `TARGET.bak.YYYY-MM-DDTHH-MM-SS`. `--no-backup` creates none. Two writes within one second produce two distinct backups.
- An unwritable backup location refuses the write and leaves the original untouched.
- Cleanup removes a matching backup dated 8 days ago, keeps one dated 6 days ago, and never touches a file that does not match the name pattern, including one named like a backup in another directory.
- `--sha256` mismatch exits non-zero.
- `git check-ignore` reports a timestamped backup name under a spec directory as ignored, and does not ignore `spec.md` or `metrics.md`.
- `grep -c '<<' .specify/scripts/writefile.sh` is 0.
- Tests use a temporary directory removed by a trap. They never write outside it and never touch the real `~/.spec-kit`.

**Commit:** `feat(workflow): add writefile helper with verified, backed-up writes`
**Done when:** `bash .specify/scripts/test/run.sh` passes; `bash -n` clean; no non-ASCII bytes in the script; the "Before claiming done" list is satisfied.

## T3: Edit verification -- `verify-edit.sh`

**Type:** Tooling (bash)
**blocked_by:** none
**Parallel:** T1, T2, T4, T5

**Files:**
- `.specify/scripts/verify-edit.sh` (new)
- `.specify/scripts/test/test_verify_edit.sh` (new; self-contained assert helpers)

If T3 runs before T2, run this test file directly with `bash`; T2 creates `run.sh`.

**Interface:** `verify-edit.sh FILE [--contains TEXT] [--absent TEXT] [--count TEXT=N]`, fixed-string matching, read-only. Exit 0 landed, 1 not landed, 2 could not verify (file unreadable), 64 bad usage. See plan.md "Design Decisions".

**Test (written first):**
- `--contains` passes when the text is present and exits 1 when it is absent.
- `--absent` passes when the text is gone and exits 1 when it is still there.
- `--count` passes on the exact count, and exits 1 on a different count, including more matches than expected.
- A real `sed -i` whose pattern matches nothing leaves the file unchanged, and `verify-edit.sh` reports it as not landed (exit 1).
- A missing or unreadable file exits 2. No arguments, or an unknown option, exits 64.
- Special characters in the text (regex metacharacters, slashes, quotes) are matched literally.
- The script never modifies the file (checksum before and after).

**Commit:** `feat(workflow): add verify-edit helper for scripted-edit checks`
**Done when:** the test file passes; `bash -n` clean; ASCII only; `grep -c '<<'` on the script is 0; "Before claiming done" satisfied.

## T4: Templates -- metrics, findings, and the evidence rule

**Type:** Templates
**blocked_by:** none
**Parallel:** T1, T2, T3, T5

**Files (each in both `.ai-workflow/templates/` and `.specify/templates/`):**
- `metrics.md` (new)
- `findings.md` (new; round report plus ledger)
- `checklist.md` (edit: a `[x]` item must cite the command and result that proved it)

**Content:**
- `metrics.md`: header (feature, status including abandoned, started); per-feature counts (tickets, context clears with reason, converge rounds, findings per round by severity, resolution counts, reopened tickets); per-ticket table (wall-clock minutes, tokens, whether 100K was crossed); converge section (each run, cap values in effect, total minutes, total tokens, rounds, stop condition); a short summary block of `key: value` lines for side-by-side comparison. Values are `0` or `not tracked`, never blank.
- `findings.md`: one report per round (range reviewed, spec reviewed, reviewer brief, findings with severity) and a ledger table (round, severity, summary, disposition fixed/deferred/rejected, reason, repeat-of).

**Test (written first, in `checklist.md`):**
- `diff -rq .ai-workflow/templates .specify/templates` is clean.
- Every field named in spec behaviors 29-31 appears in `metrics.md` (a grep for each label).
- The ledger columns in `findings.md` cover spec behavior 7, and the round report covers behavior 1 (range and spec reviewed) and behavior 13 (status, clean count, stop condition).
- The checklist template contains the evidence rule.
- This feature's own `metrics.md` (bootstrapped by hand) has every field of the finished template. Reconcile any difference.

**Commit:** `docs(templates): add metrics and findings templates and checklist evidence rule`
**Done when:** the checks above pass; both template directories identical; ASCII only.

## T5: Config -- `converge.yaml`

**Type:** Config
**blocked_by:** none
**Parallel:** T1, T2, T3, T4

**Files:**
- `.specify/converge.yaml` (new)
- `.specify/scripts/test/test_converge_yaml.sh` (new; self-contained assert helpers)

**Content:** three flat keys, each with a one-line comment: `max_rounds: 4`, `max_minutes: 15`, `max_tokens: 90000`. A comment states the precedence (repo file, then `~/.spec-kit/converge.yaml`, then built-in defaults) and that values must be positive integers.

**Test (written first):**
- Exactly three keys, no nesting, each value a positive integer.
- The defaults equal the numbers in spec behavior 10 (the test reads them from `spec.md`, so a spec change breaks the test).
- No non-ASCII bytes.

**Commit:** `chore(workflow): add converge.yaml defaults`
**Done when:** the test passes; the "Before claiming done" list is satisfied.

## T6: Converge command and review prompt

**Type:** Workflow text, workflow-change
**blocked_by:** T1, T4, T5
**Parallel:** T7, T8

**Files:**
- `.claude/commands/speckit-converge.md` (rewrite: rotation, caps, stop conditions)
- `.claude/commands/speckit-implement.md` (claim verification, no-heredoc, metrics)
- `.ai-workflow/prompts/05-build.md` (same rules as implement)
- `.ai-workflow/prompts/06-review.md` (the four round briefs)
- `.specify/scripts/test/test_no_heredoc.sh` (new; heredoc ban test)
- `~/.claude/commands/speckit-converge.md` (outside the repo; re-sync offered, only with your approval)

**Content to cover (spec behavior numbers in brackets):**
- State the range and spec reviewed; stop on an empty range and ask for an explicit range; never report Converged on it [1].
- Fresh reviewers; fixed rotation: round 1 two parallel reviewers (spec conformance, AGENTS.md and security), round 2 verifies round 1 fixes landed with `verify-edit.sh` then a general review, round 3 adversarial inputs and test quality, round 4 independent full review with no prior context [2].
- Severities, the definition of clean, two consecutive clean, reset on any non-clean round [3-6].
- Ledger in `findings.md`, repeat matching done by the orchestrator, deferred SHOULD-FIX and NIT copied into notes [7, 8].
- Reviewer read-only apart from `findings.md` and `metrics.md`; author fixes between rounds inside the diff scope; out-of-scope problems become known issues [9].
- Read `converge.yaml` with the precedence rule, validate the values, print the values in effect at run start, write the start time and caps to `metrics.md`, check all caps before each round and after each reviewer [10-12, 14].
- End of run: status Converged, Not Converged, or Aborted; clean count; stop condition; open findings. Shipping after Aborted needs an "Accepted open findings" section in `notes.md` with a reason per finding [13].

**Test (written first, in `checklist.md`):**
- A table mapping each of behaviors 1-14 to the place in the command text that implements it. No behavior is unmapped.
- Static checks: the command names `converge.yaml`, the four rounds, "read-only", "Accepted open findings", and the empty-range stop; no `<<` inside a fenced block; ASCII only.
- The workflow-change rules above.
- The end-to-end walkthrough is done in T9, because it needs every other ticket.

**Commit:** `docs(workflow): rewrite converge command with rotation, caps, and ledger`
**Done when:** the mapping table is complete; the workflow-change rules pass; the drift between the repo and home copies of the command is reported.

## T7: Claim verification and paste-safe writes in implement and build

**Type:** Workflow text, workflow-change
**blocked_by:** T2, T3
**Parallel:** T6, T8

**Files:**
- `.claude/commands/speckit-implement.md`
- `.ai-workflow/prompts/05-build.md`
- `.specify/scripts/test/test_no_heredoc.sh` (new; self-contained assert helpers)

**Content [spec behavior numbers]:**
- Verify every scripted edit before writing a claim, and on failure do not write the claim and show the failure [25-27].
- Write files through `writefile.sh` or the agent's file tool, never a multi-line heredoc, and give the user only single-line commands or saved script files [15-20].
- Update `metrics.md` when a ticket closes, and never leave a field blank [28-33].
- The `speckit-implement` steps stay in order and keep "one ticket per fresh context".

**Test (written first):**
- `test_no_heredoc.sh` scans every `*.sh` under `.specify/scripts/` for `<<`, and scans the command and prompt files for `<<` inside fenced code blocks (prose mentions outside a fence are allowed). It fails on any hit.
- Mapping table of behaviors 15-20 and 25-27 to the changed text, as in T6.
- The workflow-change rules above.

**Commit:** `docs(workflow): require verified claims and paste-safe writes in implement`
**Done when:** the heredoc test passes against every script and command file in the repo; the mapping table is complete; the workflow-change rules pass.

## T8: Sync -- `sync.sh` and install to `~/.spec-kit/`

**Type:** Tooling (bash)
**blocked_by:** T2, T3, T5
**Parallel:** T6, T7

**Files:**
- `.specify/scripts/sync.sh` (new)
- `.specify/scripts/test/test_sync.sh` (new; self-contained assert helpers)
- `~/.spec-kit/writefile.sh`, `verify-edit.sh`, `sync.sh`, `converge.yaml` (outside the repo, written only by `install`)

**Interface:** `sync.sh check` compares the SHA-256 of each repo file and its home copy and reports missing, identical, and differing. Exit 0 only when all are identical. `sync.sh install` copies repo to home, and refuses to overwrite a differing home file unless `--force` is given. The home directory is `$SPEC_KIT_HOME`, defaulting to `~/.spec-kit`, so tests can point it elsewhere.

**Home-directory rule:** the real `install` is run only when you approve it. Show the `check` output first.

**Test (written first):**
- Identical copies: `check` exits 0.
- A differing home copy is reported by name, `check` exits non-zero, and the file is unchanged.
- `install` refuses to overwrite a differing file without `--force`, and overwrites with it.
- A missing home directory is created by `install`.
- All test cases use `SPEC_KIT_HOME` in a temporary directory. The real home is never touched by tests.
- `bash -n`; no `<<`; ASCII only.

**Commit:** `feat(workflow): add sync helper for repo and home copies`
**Done when:** the test passes. After your approval, the real install is done and `sync.sh check` exits 0.

## T9: Verification and dogfood

**Type:** Verification
**blocked_by:** T6, T7, T8
**Parallel:** none

**Files:**
- `.specify/specs/002-ai-workflow-enhancements/checklist.md`, `notes.md`, `metrics.md`, `findings.md` (final state)
- Fixes for anything the run finds go in the file they belong to, each as its own `fix(workflow): ...` commit.

**Work:**
1. Run everything: `bash .specify/scripts/test/run.sh`, `bash -n` on all scripts, `make lint`, `make test`, the workflow-only guard, the template diff.
2. Fresh-clone check: clone the repo to a temporary directory, point `HOME` at an empty temporary directory, run the test suite there. It must pass with no files from the real home (success criterion 17).
3. Walkthrough on this feature using the new converge command, with an explicit range `b92af87..HEAD` because the branch may equal main:
   - empty range stops without "Converged" (run on a scratch range such as `HEAD..HEAD`);
   - the cap values in effect print at run start;
   - round 1 spawns two reviewers; a SHOULD-FIX resets the clean count; a NIT-only round does not;
   - a reviewer changes no file except `findings.md` and `metrics.md` (check with `git status` after each round);
   - the Aborted path: in a scratch clone with `max_rounds: 1`, status is Aborted, open findings are listed, and shipping requires the "Accepted open findings" section;
   - the real converge run on this feature is subject to its own caps (4 rounds, 15 minutes, 90K tokens).
4. Success criteria: check each of the 22 in the spec, with evidence, in `checklist.md`.
5. Finalize `metrics.md` (all tickets, all runs, the summary block) and `notes.md` (decisions, context clears, blockers, known issues, Accepted open findings if any).
6. Confirm constitution version 1.1 and ADR 0006 are in place, and that no other article or AGENTS.md rule was changed.

**Test:** the steps above are the test. Every step records its command and observed result.

**Commit:** `docs(spec): record 002 verification, findings, and metrics`
**Done when:** all 22 success criteria are checked with evidence; converge status is Converged (two consecutive clean rounds), or Aborted with a written "Accepted open findings" section that you approve; `metrics.md` has no blank fields; guards are clean.
