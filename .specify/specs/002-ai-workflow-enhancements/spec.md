# Spec: AI Workflow Enhancements

**Date:** 2026-09-20
**Author:** ssr0016
**Status:** Clarified (revision 5, awaiting confirmation)

---

## Problem

The workflow leans on habit, not rules, and it cost real time in 001-blog-post-crud. Converge ran 6 rounds over about an hour and reached 114.9K tokens, "clean" was judged case by case, and a scripted edit silently did not apply while the notes claimed it was fixed. Getting files onto disk through terminal paste failed 12+ times (split heredocs, non-ASCII look-alike characters in base64, multi-line commands interleaving), and notes.md and checklist.md were finally written by hand in nano. Nothing measures whether the workflow is getting better.

## Solution

Make convergence bounded and auditable, make every claim in a workflow artifact backed by a check, and give the workflow one reliable way to write any file that does not depend on terminal paste. Record a consistent metrics file per feature so features can be compared. Put the converge rules into the constitution as a new Article 7.4.

## Behaviors

**Converge discipline**

1. Every converge run states what it reviewed: the exact range of changes and the spec it was checked against. If the range has no changes, the run stops and says so. It never reports a review of nothing as clean.
2. A reviewer is never the author of the change under review. Reviewers are fresh each round, and the briefs follow a fixed rotation defined by the workflow, not chosen by the user:
   - Round 1: two reviewers in parallel, one for spec conformance and one for AGENTS.md rules and security. The two count as one round, and the round is clean only if both are clean.
   - Round 2: first verify that the round 1 fixes actually landed, then a general review.
   - Round 3: adversarial inputs and test quality.
   - Round 4: an independent full review with no prior context.
   A fifth round is never needed, because the round cap in behavior 10 stops the run first.
3. Every finding carries a severity: BLOCKER, SHOULD-FIX, or NIT.
4. A round is **clean** only if it has zero BLOCKER and zero SHOULD-FIX findings. NITs do not break a clean round but are still recorded.
5. The feature is **Converged** only after two consecutive clean rounds. Any non-clean round resets the count to zero.
6. The two-consecutive-clean rule itself is unchanged by this spec.
7. Findings are kept in a per-feature ledger. Each entry records round, severity, summary, and disposition: fixed, deferred (with reason), or rejected (with reason). A finding already in the ledger is not raised again as new.
8. Every non-fixed SHOULD-FIX and NIT ends up in the feature's notes with its disposition. Nothing is dropped silently.
9. During a review round the reviewer is read-only. The reviewer changes nothing except the findings report, the ledger, and the metrics file. Fixes are made by the author between rounds, stay inside the scope of the change under review, and out-of-scope problems are recorded as known issues.
10. Converge has hard caps per feature: at most 4 rounds, at most 15 minutes, and at most 90K tokens. The token cap counts everything spent on converge for that feature, all rounds and all reviewers combined.
11. Converge stops on the first of: two consecutive clean rounds (Converged), any cap reached (aborted, Not Converged), or a user interrupt.
12. The caps are configurable per project in a single in-repo settings file, and the values in effect are shown at the start of each run. The defaults are the values in behavior 10.
13. Each run ends with an explicit status (Converged, Not Converged, or Aborted), the consecutive-clean count, which stop condition fired, and any findings still open. An aborted run hands the open findings to the user for a decision, and does not claim the feature is ready to ship. The user may ship after an Aborted run only if they have written an "Accepted open findings" section in the feature's notes, listing each open finding with a one-line reason. Without that written acceptance, a new converge run is required.
14. The caps are hard limits from this spec forward. Features that finished before it, including 001, are grandfathered and are not judged against them.

**Reliable file writing (paste safety)**

15. The workflow has one standard way to write a file from any content, so the user never has to open an editor by hand to get a file written.
16. That standard way takes the content from standard input or from a file. Content is never embedded inside the command that writes it.
17. Content is transported in an encoding that survives terminal paste. Pure ASCII transport is the default. If the content itself must contain non-ASCII characters, they arrive intact and are verified.
18. Every write is verified before it is reported successful. At minimum it checks the line count and that a known signature line is present in the result. On any mismatch the command exits non-zero and says what differed.
19. Multi-line heredocs are not used in committed scripts, or in any command the workflow asks the user to paste.
20. When the workflow needs the user to run something in a terminal, it is a single line or a saved script file. It is never a multi-line block that depends on paste behaving.
21. If unexpected non-ASCII characters (for example a Cyrillic look-alike for a Latin letter) appear in content that should be ASCII, the write fails and reports where they are. It does not write a silently corrupted file.
22. When the target file already exists, the write keeps a timestamped backup of the previous content by default. The caller can explicitly opt out of the backup. Backups are kept out of version control by an ignore rule in each spec directory. On each write, the helper also removes backups it made that are older than 7 days.
23. The write helper and the converge settings each have a copy inside the repo, so a fresh clone works without any setup. A canonical copy also exists in the user's home directory for use across projects. The two are kept from drifting apart silently. Which direction syncs is a plan decision.
24. The helper needs nothing beyond tools already present on the machine. No new dependency is introduced.

**Claim verification**

25. After any scripted edit (sed, perl in-place, python replace, or similar) the agent confirms the edit landed, by searching for the new content or re-reading the file, before writing any claim about it into notes, the checklist, the ledger, or a commit message.
26. If the check fails, the claim is not written and the failure is shown to the user immediately.
27. A claim recorded as done in a workflow artifact points to something that was actually observed (a test result, a file state, a command output), not to intent.

**Metrics tracking**

28. Each feature has its own metrics file inside its spec directory. It is plain text, readable and diffable without tooling, and separate from notes.
29. Per feature it records: number of tickets; number of context clears and the reason for each; number of converge rounds; findings per round by severity; how each finding was resolved; and how many tickets were reopened after being marked done.
30. Per ticket it records: wall-clock minutes, tokens used, and whether the 100K-token ceiling was crossed.
31. For converge it records total wall-clock time and total tokens, the number of rounds, which stop condition ended it, and the cap values in effect.
32. Token figures are the agent session's own reported usage. If that is unavailable, the field says "not tracked".
33. The file is updated when each event happens (a ticket closes, a round ends), not reconstructed later.
34. The file ends with a short summary block per feature so two features can be compared side by side.
35. Metrics themselves are descriptive and never block a ticket, review, or ship. The converge caps in behavior 10 are the only enforcement.

## Edge Cases

- Converge is run on a branch equal to main and a plain diff is empty -> the run says the range is empty and asks for an explicit range. It does not report Converged.
- Round 4 ends non-clean, or clean with only one clean round in a row -> the cap is reached, status is Aborted (Not Converged), open findings go to the user. No fifth round starts.
- The 15-minute or 90K-token cap is hit in the middle of a round -> the round is finished as far as it can be reported, marked incomplete, and does not count as clean.
- The user raises a cap in the settings file mid-feature -> the new value applies from the next run. The metrics file records the values in effect for each run.
- The user interrupts converge -> status is Not Converged (interrupted), the ledger and metrics are saved as they stand.
- Converge is re-run on a feature that ended Aborted -> it starts a new run with fresh caps only if the user asks for it, and the metrics file records both runs.
- A reviewer finds nothing but the ledger has an unresolved BLOCKER -> the round is not clean.
- A fix in round N introduces a new SHOULD-FIX in round N+1 -> the count resets to zero, and the finding is logged as new.
- A reviewer raises something already in the ledger as rejected -> it is marked as a repeat and ignored. If it comes with new evidence, it is logged as a fresh finding that references the old one.
- A round has only NITs -> the round is clean. The NITs are recorded.
- A reviewer tries to change a source file during a round -> the change is not accepted as part of the round, and it is reported as a finding against the reviewer's brief.
- The round 4 reviewer has no prior context and so cannot know the ledger -> repeats are matched against the ledger afterward by the workflow, not by the reviewer. A match is marked as a repeat, and new evidence makes it a fresh finding.
- In round 1 one of the two reviewers is clean and the other is not -> the round is not clean.
- The user wants to ship after an Aborted run but the "Accepted open findings" section is missing, or lists a finding without a reason -> that finding is not accepted. A new converge run is required unless the section is completed.
- Backups older than 7 days exist from before a write -> they are removed. Backups newer than 7 days are kept. Files the helper did not make are never removed.
- The content to write contains lines that look like heredoc terminators, quotes, backticks, or dollar signs -> it is written exactly as given, because nothing about it is interpreted by the shell.
- The content to write is empty -> the write is refused with an error. It does not create an empty file and report success.
- The target file already exists -> a timestamped backup is kept and the new content is written. With the opt-out, no backup is made.
- The backup cannot be made (no write permission, disk full) -> the write is refused and the original is untouched.
- The helper is given content whose signature line is missing -> non-zero exit, and the target is left in its prior state.
- The in-repo copy and the home-directory copy of the helper differ -> the difference is reported. Neither is silently overwritten.
- A scripted edit matches nothing (pattern absent) -> counts as "did not land". No claim is written and the user sees the failure.
- A scripted edit matches more places than expected -> reported as a mismatch. The claim describes what was actually changed.
- Verification cannot be done (file unreadable) -> the claim is not written. The user is told verification was not possible.
- Tokens or timing cannot be measured for a ticket -> the field says "not tracked" rather than being left blank or guessed. The rounds cap and the time cap still apply.
- A feature finishes with zero context clears or zero converge rounds -> the record shows zero, so "none" is distinguishable from "not tracked".
- A feature is abandoned mid-way -> the metrics file stays with a status noting that. It is not deleted.
- Feature 001 and earlier have no metrics file -> nothing is invented. Where 001's notes hold the data it may be copied over, and gaps are marked "not tracked".

## Non-Goals

- Not the application's Prometheus metrics (`http_requests_total`, `blog_post_writes_total`, etc.). This is about the AI workflow, not the running service.
- Not prompt-injection defense, untrusted-data handling, or secret redaction. Those are a different problem from paste reliability and are out of scope.
- Not a fix for the terminal or its paste behavior. The workflow works around the problem. It does not change the terminal.
- No dashboards, charts, or external analytics service. Plain in-repo files are enough.
- No automatic blocking of commits or CI on metrics values. Only converge has caps.
- No second reviewer model or tool. A fresh agent per round with a different brief is enough for now.
- No change to the two-consecutive-clean rule or to the severity levels.
- No renumbering or rewording of existing constitution articles 7.1 to 7.3. The change is one new article.
- No changes to the blog application code, API, database, or its docs.
- No retroactive rewrite of 001's artifacts, and no retroactive application of the caps to 001.

## Success Criteria

- [ ] A converge run on an empty range stops with a clear message and never reports Converged.
- [ ] A round with one SHOULD-FIX resets the clean count. A round with only NITs does not.
- [ ] A converge run stops at 4 rounds, 15 minutes, or 90K tokens, whichever comes first, with status Aborted and the open findings listed.
- [ ] Changing a cap in the settings file changes the limit on the next run, and the run shows the values in effect.
- [ ] Every converge report names the range and spec reviewed, lists findings by severity, and ends with a status, the clean count, and the stop condition.
- [ ] The findings ledger shows every finding with a disposition, and a rejected finding is not re-raised as new.
- [ ] Every deferred SHOULD-FIX and NIT appears in the feature's notes.
- [ ] A reviewer round changes no files other than the findings report, ledger, and metrics file.
- [ ] Any file of any content, including quotes, backticks, dollar signs, and heredoc-like lines, can be written by the standard method with no editor opened by hand.
- [ ] A write whose result does not match its expected line count or signature line exits non-zero and reports the difference.
- [ ] Overwriting an existing file leaves a timestamped backup unless the caller opted out.
- [ ] Backups do not show up as untracked files in git, and backups older than 7 days are gone after the next write.
- [ ] Converge uses the fixed rotation: round 1 has two parallel reviewers, round 2 checks that fixes landed first, round 3 is adversarial, round 4 has no prior context.
- [ ] Shipping after an Aborted run is possible only with a written "Accepted open findings" section that lists every open finding with a reason.
- [ ] A content payload with a look-alike non-ASCII character in a place that should be ASCII is rejected with its position.
- [ ] No committed script and no command the workflow asks the user to paste contains a multi-line heredoc.
- [ ] On a fresh clone, the write helper and the converge settings work without copying anything from the home directory.
- [ ] A scripted edit that did not apply produces no "fixed" claim in notes or checklist, and the user sees the failure.
- [ ] Each feature has a metrics file with every field in behaviors 29-31 filled in (zeros or "not tracked" where applicable), updated as events occurred.
- [ ] Two features can be compared by tickets, clears, rounds, findings, time, and tokens from their metrics files alone.
- [ ] Constitution Article 7.4 "Converge" exists, stating that (a) the reviewer is read-only apart from findings and metrics artifacts, (b) the two-consecutive-clean rule is unchanged, and (c) converge has caps. It is backed by an ADR and a `docs(constitution)` commit.
- [ ] No other existing article or AGENTS.md rule is contradicted.

## Open Questions

All resolved as of revision 4:

- [x] Where the converge rules go -> new Article 7.4 "Converge", a lifecycle step between 7.3 (Before Shipping) and the ship itself. 7.1 and 7.3 are not changed.
- [x] Meaning of "read-only" -> the reviewer is read-only. The author fixes blockers between rounds, which matches the current converge step 6.
- [x] Scope and size of the token cap -> tokens spent on converge per feature, all rounds and reviewers. Raised from 60K to 90K in revision 5, because 001 averaged about 19K per round, so four rounds is about 77K and 60K would fire before round 4. It is a hard limit going forward, and 001 (114.9K) is grandfathered.
- [x] Helper and settings location -> in-repo copies plus a canonical copy in the home directory. The plan decides sync direction.
- [x] Overwrite policy -> keep a timestamped backup by default, with an explicit opt-out.
- [x] Token measurement -> the agent session's own reported usage, otherwise "not tracked".

- [x] Backup files -> ignored by git through a rule in each spec directory, and backups older than 7 days are removed by the helper on each write (behavior 22).
- [x] Shipping after an Aborted converge -> allowed only with a written "Accepted open findings" section in notes, one line of reason per finding. Otherwise a new run is required (behavior 13).
- [x] Reviewer briefs -> a fixed 4-round rotation defined by the workflow, not chosen by the user (behavior 2).

No open questions remain.
