# Plan: AI Workflow Enhancements

**Spec:** ./spec.md
**Date:** 2026-09-20

---

## Approach

This feature changes the workflow, not the blog. It adds no Go code and touches nothing under
`internal/`, `pkg/`, `cmd/`, `docs/API.md`, swagger, migrations, the Makefile, or CI. Everything is
Markdown, YAML, or bash, and bash is limited to what is already installed (bash, coreutils, grep,
awk, sha256sum). No new dependency, so no dependency ADR. The one ADR is for the constitution
change.

Three groups of change, matching the spec:

1. **Governance.** ADR 0006 and constitution Article 7.4 "Converge".
2. **Tooling.** A write helper, an edit-verification helper, a sync/drift checker, a
   `.gitignore` rule, and a plain-bash test runner for them.
3. **Workflow text.** Templates (metrics, findings ledger), `converge.yaml`, and rewrites of the
   converge command and review prompt, plus claim-verification and no-heredoc rules in the
   implement command, build prompt, and checklist template.

Two things the spec leaves to the plan are decided here (see Design Decisions): sync direction
between the repo and home copies, and the concrete file locations.

## Design Decisions

| Topic | Decision |
|---|---|
| Script location | `.specify/scripts/` in the repo (workflow tooling lives under `.specify/`). Canonical copy also at `~/.spec-kit/`. |
| Sync direction | The repo is the source of truth. `~/.spec-kit/` is an installed copy. `sync.sh check` compares SHA-256 of each pair and reports drift. `sync.sh install` copies repo -> home, and refuses to overwrite a differing home file unless given `--force`. Neither side is ever changed silently (spec edge case). |
| Home directory writes | `install` is the only step that writes outside the repo. It runs when the user asks for it, never as a side effect of another ticket. |
| `writefile.sh` interface | `writefile.sh TARGET --lines N --sig 'text' [--sha256 HEX] [--no-backup] [--allow-utf8] < content`, or `--from FILE` in place of stdin. `--lines` and `--sig` are required with stdin, because the sender must state expectations independently of the payload or the check only verifies the helper's own write. With `--from`, they default to values derived from the source file. |
| Write procedure | Read input into a temp file in the target's directory. Refuse empty. Run the ASCII gate. Verify line count and signature (and sha256 if given) on the temp file. Only then back up the existing target and rename the temp file into place. On any failure the target is untouched and the exit is non-zero with a message saying what differed. |
| ASCII gate | Any byte above 0x7F is rejected and reported as `line:column`. `--allow-utf8` skips the gate for content that must contain non-ASCII, and the line/signature/sha256 checks still run. Known limit: with `--allow-utf8` a look-alike letter cannot be told apart from legitimate text. |
| Heredocs | The helper never reads a heredoc inside its own text, and no committed script or command the workflow asks the user to paste contains a multi-line one. A test greps `.specify/scripts/` and the command files for `<<` and fails on any hit. |
| Backup name | `TARGET.bak.YYYY-MM-DDTHH-MM-SS`. Seconds are included so two writes in the same minute do not overwrite each other's backup. |
| Backup cleanup | On each write, delete files in the target's directory whose names match the strict pattern `*.bak.YYYY-MM-DDTHH-MM-SS` and whose timestamp (from the name, not mtime, because a copy preserves the original's mtime) is over 7 days old. Anything not matching the pattern is never touched. |
| Backup failure | If the backup cannot be made, the write is refused and the original is untouched. |
| Ignore rule | One file, `.specify/specs/.gitignore`, containing `*.bak.*`. A git ignore file applies to every directory beneath it, so every present and future spec directory is covered with no per-feature step to forget. The root `.gitignore` already has `*.bak` and `*.bak[0-9]*`, which do not match `.bak.<timestamp>`. |
| `verify-edit.sh` interface | `verify-edit.sh FILE [--contains TEXT] [--absent TEXT] [--count TEXT=N]`, all fixed-string matches. Exit codes: 0 edit landed, 1 not landed (pattern absent, present when it should be absent, or wrong count), 2 could not verify (file unreadable), 64 bad usage. It is read-only, so reviewers may use it. |
| Claim rule | Stated once in the implement command, the build prompt, and the checklist template: a "done" claim in notes, checklist, ledger, or commit message needs a verify-edit exit 0 or another observed result first. On exit 1 or 2 the claim is not written and the failure is shown. |
| Settings | `.specify/converge.yaml` in the repo, flat keys only: `max_rounds: 4`, `max_minutes: 15`, `max_tokens: 90000`, each with a comment. Precedence: repo file, then `~/.spec-kit/converge.yaml`, then built-in defaults in the command text. Flat keys mean no YAML parser is needed. The command validates that each value is a positive integer and stops with an error on anything else. |
| Cap enforcement | Enforced by the converge command, not by a program. At run start it records the start time (from `date`) and the cap values in `metrics.md`, then checks all three caps before each round and after each reviewer returns. See Risks. |
| Reviewer rotation | Written into the converge command and `06-review.md` as four named briefs, matching spec behavior 2. Round 1 spawns two reviewers in parallel. Round 2 checks the round 1 fixes with `verify-edit.sh` first. Round 4 is given the spec and diff only, no prior findings. |
| Ledger matching | The orchestrating agent, not the reviewer, compares each reported finding against `findings.md` and marks repeats (so round 4 can stay context-free). |
| New artifacts per feature | `metrics.md` (spec behaviors 28-34) and `findings.md` (report + ledger, spec behavior 7). The spec names the ledger but not its location. It gets its own file so a reviewer's write access is exactly `findings.md` and `metrics.md`. |
| Template copies | `.ai-workflow/templates/` and `.specify/templates/` are identical today. New and edited templates go in both. A check in the verification step diffs the two directories. The duplication itself is left alone. |
| Constitution | New Article 7.4 "Converge" after 7.3, covering (a) reviewer read-only apart from findings and metrics artifacts, (b) two-consecutive-clean unchanged, (c) caps, with a pointer to ADR 0006 and `converge.yaml`. Version 1.0 -> 1.1. Commit message `docs(constitution): add Article 7.4 converge`. 7.1 to 7.3 are not edited. |
| Test runner | One self-contained plain bash test file per script (`.specify/scripts/test/test_<name>.sh`, each with its own small assert helpers), plus `.specify/scripts/test/run.sh` that runs every `test_*.sh`. Separate files mean S2 and S3 never edit the same file and can run in parallel. No bats or shellcheck (neither is installed). Non-ASCII fixtures are built with printf byte escapes so the test file itself stays ASCII. |

## Files to Touch

| File | Change | Risk |
|---|---|---|
| docs/adr/0006-bounded-converge.md | New ADR: caps, rotation, read-only reviewer, accepted-findings path | Low |
| .specify/memory/constitution.md | Article 7.4, version 1.1 | Medium (governing document) |
| .specify/specs/.gitignore | `*.bak.*` | Low |
| .specify/scripts/writefile.sh | New write helper | Medium |
| .specify/scripts/verify-edit.sh | New edit verifier | Low |
| .specify/scripts/sync.sh | New `check` and `install` between repo and `~/.spec-kit/` | Medium (writes outside repo) |
| .specify/scripts/test/run.sh, test_*.sh | `run.sh` runs all `test_*.sh`. One test file each for writefile, verify-edit, sync, converge.yaml, the ignore rule, and the heredoc ban | Low |
| .specify/converge.yaml | Defaults 4 / 15 / 90000 | Low |
| .ai-workflow/templates/metrics.md, .specify/templates/metrics.md | New template | Low |
| .ai-workflow/templates/findings.md, .specify/templates/findings.md | New template (report + ledger) | Low |
| .ai-workflow/templates/checklist.md, .specify/templates/checklist.md | Evidence rule for `[x]` items | Low |
| .claude/commands/speckit-converge.md | Rotation, caps, stop conditions, status output, empty-range stop, read-only reviewer, accepted-findings rule | High (drives the review loop) |
| .claude/commands/speckit-implement.md | Claim verification, no-heredoc rule, metrics updates per ticket | Medium |
| .ai-workflow/prompts/05-build.md, 06-review.md | Same rules; the four round briefs | Medium |
| ~/.spec-kit/* (outside repo) | Installed copies of the scripts and `converge.yaml` | Low, removable |
| ~/.claude/commands/speckit-converge.md (outside repo) | Currently identical to the repo copy. Re-sync after S6, only with user approval. | Low |
| .specify/specs/002-ai-workflow-enhancements/metrics.md, findings.md, notes.md, checklist.md | Created while doing this feature (dogfood) | Low |

Not touched: `internal/`, `pkg/`, `cmd/`, `docs/API.md`, swagger, migrations, Makefile, CI,
`AGENTS.md`, `docs/ARCHITECTURE.md`, and every 001 artifact.

## Order (DAG)

These are work slices, not tickets. `tasks.md` is not written until you approve this plan.

```
Roots (no blockers, parallel):

  S1 governance (ADR 0006 + Article 7.4)
  S2 reliable write (.gitignore + writefile + tests)
  S3 verify-edit (+ tests)
  S4 templates (metrics, findings, checklist evidence rule)
  S5 converge.yaml

Dependent slices:

  S6 converge command + review prompt   blocked_by: S1, S4, S5
  S7 claim rules in implement/build     blocked_by: S2, S3
  S8 sync + install to ~/.spec-kit      blocked_by: S2, S3, S5
  S9 dogfood + verify                   blocked_by: S6, S7, S8
```

- S1, S2, S3, S4, S5 have no dependencies and can run in parallel. They touch disjoint files,
  except that every slice also updates this feature's `checklist.md`, `notes.md`, and `metrics.md`,
  so parallel work needs separate worktrees and a rebase per merge. Serial order S1 to S9 is valid
  and conflict-free.
- S6 is blocked by S1 (7.4 wording), S4 (files it refers to), S5 (settings it reads).
- S7 is blocked by S2 and S3 (it points at those tools).
- S8 is blocked by S2, S3, S5 (it syncs their outputs).
- S9 is blocked by S6, S7, S8.
- Each slice is complete on its own: it carries its files, its tests or manual check, and its doc
  mention. S2 includes its `.gitignore` rule because writefile creates the backups it ignores.
- One slice = one fresh context, per Article 7.2. Bootstrap note: the helpers do not exist yet, so
  S1 to S3 create their files with the agent's file-writing tool, not terminal paste.

## Risks

- **Caps depend on the agent following the command, not on code.** The agent could skip a check. -> The run start time and cap values are written to `metrics.md` before round 1, checks happen before each round and after each reviewer, and a missing start record is itself a finding in the next review. A program-enforced cap is possible later but would need a runner, which is out of scope.
- **The 90K token cap may still fire before round 4 if rounds run heavier than 001's.** 001 averaged about 19K per round (114.9K over 6), so four rounds is about 77K, and 90K leaves about 13K of headroom. Round 1 runs two reviewers in parallel and could cost more than the average. -> Each round's tokens go in `metrics.md`, so the cap can be tuned in `converge.yaml` with evidence. Reviewers get the spec and diff, not the whole tree. The token cap is checked after each reviewer returns, so round 1 can stop between its two reviewers if the budget is already spent.
- **Token figures may be unavailable.** -> Recorded as "not tracked" (spec behavior 32). The rounds and time caps still apply.
- **Repo and home copies drift.** -> `sync.sh check` reports it and neither side is overwritten silently. `install` needs `--force` to overwrite.
- **The helper is itself corrupted by paste.** -> It is created with file tools, is pure ASCII, and is tested against its own heredoc ban. The user only ever runs a single-line command.
- **ASCII gate blocks legitimate content.** -> `--allow-utf8` exists, with the limit stated in Design Decisions.
- **Line count and signature can pass on a partly wrong payload.** -> The optional `--sha256` gives full integrity when the sender can supply it.
- **Backup cleanup deletes something wanted.** -> Strict name pattern, timestamp from the name, target's own directory only, 7-day threshold, and a test proving non-matching and 6-day-old files survive.
- **Constitution change conflicts with 7.3.** -> 7.3 says "Converged (2 consecutive clean)". 7.4 adds bounds and does not change that rule. The ADR says so explicitly.
- **Two template directories drift.** -> Diff check in the verification step.
- **Round 4 with no prior context repeats known findings.** -> The orchestrator matches against the ledger afterward, so repeats cost tokens but do not reset the clean count wrongly.
- **`~/.claude/commands/speckit-converge.md` is a second copy** of the repo command and is what a bare `/speckit-converge` may resolve to. -> S6 edits the repo copy. Re-syncing the home copy is offered and done only with approval.
- **Dogfooding is circular.** Converging feature 002 with the new rules tests them on themselves. -> Accepted and useful: that run is also the first real metrics file. Any rule that fails there is fixed before this feature is called done.

## Test Strategy

- **Scripts (automated, `.specify/scripts/test/run.sh` runs every `test_*.sh`):**
  - writefile: content containing quotes, backticks, dollar signs, and heredoc-terminator lines is written byte-identical; empty content refused; missing signature and wrong line count exit non-zero with the target unchanged; a Cyrillic look-alike (built by byte escape) is rejected with `line:column`; `--allow-utf8` writes it intact; backup created with the timestamp name; `--no-backup` makes none; two writes in one second get distinct backups; unwritable backup location refuses the write; cleanup removes an 8-day-old matching backup, keeps a 6-day-old one, and never touches a non-matching file.
  - verify-edit: contains present and absent; wrong count; a `sed` that matched nothing reports not landed (exit 1); unreadable file exits 2; usage errors exit 64.
  - sync: identical copies pass; a differing home copy is reported and left unchanged; `--force` overwrites.
  - ignore rule: `git check-ignore` matches a timestamped backup name and does not match `spec.md` or `metrics.md`.
  - heredoc ban: grep of `.specify/scripts/` and the command files finds no multi-line heredoc.
- **Static checks:** `bash -n` on each script. Every new file is checked for non-ASCII (`grep -P '[^\x00-\x7F]'`), except where a non-ASCII fixture is intentional and built at runtime.
- **Workflow text (manual walkthrough, because prompts cannot be unit tested):** run the converge command on this feature and confirm each of: empty range stops without "Converged"; the cap values are printed at start; round 1 spawns two reviewers; a round with one SHOULD-FIX resets the count; a NIT-only round does not; a reviewer round changes no file except `findings.md` and `metrics.md`; lowering `max_rounds` to 1 in a scratch copy of the settings gives status Aborted with open findings listed; the accepted-findings path is required to ship after Aborted.
- **Regression guards for "workflow only":** `git diff --name-only main` contains nothing under `internal/`, `pkg/`, `cmd/`, or `docs/API.md`. `make lint` and `make test` still pass, which AGENTS.md requires before any commit. `diff -rq .ai-workflow/templates .specify/templates` is clean.
- **Not run:** integration tests, `make swagger`, migrations. Nothing in scope touches them.
- **Spec coverage:** the 22 success criteria map to the checks above. `checklist.md` records the observed result for each with evidence, using the new evidence rule.

## Rollback

All changes are additive files or text edits, each in its own commit. Revert the commits to roll back;
reverting the constitution commit restores version 1.0 and 7.4 disappears. The installed
`~/.spec-kit/` directory is removable with nothing depending on it, since the repo copies are the
ones the workflow uses. No data, schema, or running service is involved.

## Deviations From the Brief

- **Extra files beyond your deliverable list:** `findings.md` (ledger location the spec left open), `sync.sh` (needed for spec behavior 23, "kept from drifting apart silently"), and the test runner (needed to show the helpers work).
- **One ignore file, not one per spec directory:** `.specify/specs/.gitignore` covers all of them. This meets the intent with less to forget. If you want a literal file in each spec directory, say so.
- **`--lines` and `--sig` are required on the write helper**, stricter than "at minimum verify".
- **No config parser script.** The converge command reads `converge.yaml` itself.
