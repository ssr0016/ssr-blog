# Metrics: AI Workflow Enhancements

Descriptive only. Never blocks a ticket, review, or ship. Updated when each event happens.
A value is a number (0 is a real value) or "not tracked". "pending" is used only for an event
that has not happened yet. Never blank.

Bootstrapped by hand in T1. The metrics template is built in T4; T4 reconciles any difference.

**Feature:** 002-ai-workflow-enhancements
**Status:** Shipped
**Started:** 2026-09-20

---

## Per-feature counts

| Metric | Value |
|---|---|
| Tickets | 9 |
| Tickets closed | 9 |
| Context clears | 3 |
| Converge rounds | 2 |
| Tickets reopened after done | 0 |

### Context clears

| # | Ticket | Reason |
|---|---|---|
| 1 | T1 | Start of ticket; fresh context per Article 7.2 |
| 2 | T6 | Start of ticket; fresh context per Article 7.2 |
| 3 | T8 | Start of ticket; fresh context per Article 7.2 |

### Findings per round

| Round | BLOCKER | SHOULD-FIX | NIT |
|---|---|---|---|
| 1.1 (run 1, round 1) | 0 | 9 | 5 |
| 2.1 (run 2, round 1) | 1 | 5 | 5 |

### Finding resolutions

| Fixed | Deferred | Rejected |
|---|---|---|
| 16 | 9 | 0 |

Of the 25 findings of the two real runs: 16 fixed (run 1 #1 to #9 and #13, run 2 #15 to #20) and 9 deferred as accepted NITs (run 1 #10, #11, #12, #14 and run 2 #21 to #25). None rejected, none open. The findings.md ledger has the commit of each.

## Per-ticket

| Ticket | Wall-clock minutes | Tokens | 100K crossed |
|---|---|---|---|
| T1 | 2.2 | not tracked | not tracked |
| T2 | not tracked | not tracked | not tracked |
| T3 | not tracked | not tracked | not tracked |
| T4 | not tracked | not tracked | not tracked |
| T5 | not tracked | not tracked | not tracked |
| T6 | not tracked | not tracked | not tracked |
| T7 | not tracked | not tracked | not tracked |
| T8 | not tracked | not tracked | not tracked |
| T9 | 124 | not tracked | not tracked |

## Converge

| Run | Caps in effect (rounds / minutes / tokens) | Total minutes | Total tokens | Rounds | Stop condition |
|---|---|---|---|---|---|
| 1 (real, b92af87..53d9113) | 4 / 15 / 90000 | 4 | 116430 | 1 | token cap (116430 of 90000; rounds 1 of 4, minutes 4 of 15) |
| 2 (real, fix range 53d9113..4050f45, started 12:20:29Z, fresh caps after Aborted) | 4 / 15 / 90000 | 8 | 108691 | 1 | token cap (108691 of 90000; rounds 1 of 4, minutes 8 of 15) |

### Walkthrough runs (scratch, NOT counted toward the real caps or the summary block)

Scratch runs test the converge mechanism itself, in scratch clones or a scratch repo. Their tokens are not part of the 90000 cap of the real runs above.

| Run | Caps in effect (rounds / minutes / tokens) | Total minutes | Total tokens | Rounds | Stop condition |
|---|---|---|---|---|---|
| B1 empty range (real repo, no reviewers) | 4 / 15 / 90000 | 0 | 0 | 0 | empty range (stopped, not Converged) |
| B2 Aborted path (scratch clone, range e5dde30^..e5dde30) | 1 / 15 / 90000 | 1.8 | 56761 | 1 | round cap (1 of 1) |
| B3 reset and NIT-only (scratch repo, feature 999-greet) | 4 / 15 / 90000 | 5.4 | 81290 | 4 | round cap (4 of 4) |
| S3 minutes cap (scratch clone of 31d7ff1, range 799eab0^..799eab0, started 13:07:12Z) | 4 / 1 / 90000 | 3.1 | 64756 | 1 | minutes cap (116 s of 60 s at the first check after reviewer A returned, with 27957 of 90000 tokens and round 1 of 4; the round was marked incomplete, not clean) |

Scratch tokens total 202807 (B2 56761, B3 81290, S3 64756).

## Summary

```
feature: 002-ai-workflow-enhancements
status: shipped
tickets: 9
tickets_closed: 9
context_clears: 3
converge_rounds: 2
converge_minutes: 12
converge_tokens: 225121
findings_blocker: 1
findings_should_fix: 14
findings_nit: 10
tickets_reopened: 0
```
