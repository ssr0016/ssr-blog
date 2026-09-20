# ADR 0006: Bounded Converge

**Status:** Accepted
**Date:** 2026-09-20
**Deciders:** ssr0016

---

## Context

The converge step (multi-agent review until two consecutive clean rounds, constitution Article 7.3)
had no bounds and no definition of "clean". On feature 001-blog-post-crud it ran 6 rounds over about
an hour and used 114.9K tokens. "Clean" was judged case by case. A scripted edit silently did not
apply while the notes claimed the finding was fixed. Nothing recorded which findings had been seen
before, so the same issues could be raised again as new.

Spec 002 (AI workflow enhancements) asks for converge that is bounded and auditable. That needs a
constitution change, which by the Amendments rule needs this ADR.

## Decision

Converge gets hard caps, a fixed reviewer rotation, and a read-only reviewer. The
two-consecutive-clean rule is not changed.

- **Caps:** at most 4 rounds, 15 minutes, and 90K tokens per feature. The token cap counts all
  rounds and all reviewers together. The values live in `.specify/converge.yaml`, and the values in
  effect are printed at the start of each run. Precedence: the repo file, then
  `~/.spec-kit/converge.yaml`, then built-in defaults. 90K is chosen because 001 averaged about 19K
  per round, so four rounds is about 77K; 60K would fire before round 4. The caps apply from this
  ADR forward. 001 (114.9K tokens) is grandfathered.
- **Rotation (fixed by the workflow, not chosen by the user):**
  - Round 1: two fresh reviewers in parallel, one for spec conformance and one for AGENTS.md rules
    and security. Together they count as one round, and it is clean only if both are clean.
  - Round 2: first verify that the round 1 fixes landed, then a general review.
  - Round 3: adversarial inputs and test quality.
  - Round 4: an independent full review with no prior context.
- **Read-only reviewer:** a reviewer changes nothing except the findings report/ledger
  (`findings.md`) and `metrics.md`. The author fixes between rounds, inside the scope of the change
  under review. Out-of-scope problems are recorded as known issues.
- **Clean and Converged, unchanged:** a round is clean with zero BLOCKER and zero SHOULD-FIX (NITs
  are recorded but do not break it). The feature is Converged after two consecutive clean rounds.
  Any non-clean round resets the count.
- **Stop conditions:** Converged, any cap reached (Aborted, Not Converged), or a user interrupt.
  Every run ends with a status, the clean count, the stop condition, and the open findings.
- **Accepted-findings path:** an Aborted run never claims the feature is ready to ship. The user
  may ship after an Aborted run only if the feature's `notes.md` has an "Accepted open findings"
  section listing every open finding with a one-line reason. Without it, a new converge run is
  required. This is a written risk acceptance by the user. It does not make the run Converged.

Constitution Article 7.4 "Converge" records the read-only reviewer, the unchanged rule, the caps, and the accepted-findings exception, and points here.

## Consequences

### Positive
- A converge run has a known worst case: 4 rounds, 15 minutes, 90K tokens.
- "Clean" has one definition, so the result does not depend on who judged it.
- The ledger stops repeat findings from being raised as new, and nothing deferred is dropped
  silently.
- Read-only reviewers cannot change the code they are judging.
- The rotation covers spec conformance, rules and security, fix verification, adversarial inputs, and
  an independent look, without the user choosing briefs each time.

### Negative
- The caps are enforced by the converge command text, not by code. An agent that ignores the text
  can run past them. Mitigation: the start time and cap values are written to `metrics.md` before
  round 1, checks run before each round and after each reviewer, and a missing start record is itself
  a finding in the next review.
- A run can end Aborted with real findings still open. The decision is then pushed to the user, which
  costs time and needs a written acceptance to ship.
- The 90K token cap can still fire before round 4 if rounds run heavier than 001's. Round 1 runs two
  reviewers and may cost more than the average. The cap is tunable in `converge.yaml` using the
  per-round tokens recorded in `metrics.md`.
- Token usage may not be measurable. Then it is recorded as "not tracked", and only the round and
  time caps apply.
- More artifacts to keep current per feature (`findings.md`, `metrics.md`).
- Round 4 has no prior context, so it can repeat known findings. The orchestrator matches them
  against the ledger afterward, which costs some tokens.

### Neutral
- Article 7.3 is not edited and still requires two consecutive clean rounds. Article 7.4 adds bounds
  around that rule and does not change it.
- Blog application code, API, and database are not affected.

## Alternatives Considered

### No caps
- **Why rejected:** This is what 001 did: 6 rounds, about an hour, 114.9K tokens, and no stopping
  rule beyond judgement.

### Program-enforced runner
- **Why rejected:** Out of scope for spec 002. It needs new code, and a runner is a larger change
  than the problem calls for. It remains possible later if command-text enforcement proves
  unreliable.

### Second reviewer model
- **Why rejected:** Not required. A fresh agent per round with a different brief is enough for now.
  Adding another model or tool is a separate decision.

## References

- .specify/specs/002-ai-workflow-enhancements/spec.md (behaviors 1-14)
- .specify/specs/002-ai-workflow-enhancements/plan.md
- .specify/memory/constitution.md (Articles 7.3 and 7.4)
- .specify/converge.yaml (added in ticket T5)
- .specify/specs/001-blog-post-crud/notes.md (the converge run this responds to)
