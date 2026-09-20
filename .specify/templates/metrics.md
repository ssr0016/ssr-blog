# Metrics: [Feature Name]

Descriptive only. Never blocks a ticket, review, or ship. Updated when each event happens.
A value is a number (0 is a real value) or "not tracked". "pending" is used only for an event
that has not happened yet. Never blank.

**Feature:** [NNN-feature-name]
**Status:** [In progress | Converged | Not Converged | Aborted | Shipped | Abandoned]
**Started:** YYYY-MM-DD

---

## Per-feature counts

| Metric | Value |
|---|---|
| Tickets | 0 |
| Tickets closed | 0 |
| Context clears | 0 |
| Converge rounds | 0 |
| Tickets reopened after done | 0 |

### Context clears

| # | Ticket | Reason |
|---|---|---|
| (none yet) | - | - |

### Findings per round

| Round | BLOCKER | SHOULD-FIX | NIT |
|---|---|---|---|
| (no converge round run yet) | 0 | 0 | 0 |

### Finding resolutions

| Fixed | Deferred | Rejected |
|---|---|---|
| 0 | 0 | 0 |

## Per-ticket

| Ticket | Wall-clock minutes | Tokens | 100K crossed |
|---|---|---|---|
| T1 | pending | pending | pending |

## Converge

| Run | Caps in effect (rounds / minutes / tokens) | Total minutes | Total tokens | Rounds | Stop condition |
|---|---|---|---|---|---|
| (no converge run yet) | 4 / 15 / 90000 | 0 | 0 | 0 | none |

## Summary

```
feature: [NNN-feature-name]
status: in progress
tickets: 0
tickets_closed: 0
context_clears: 0
converge_rounds: 0
converge_minutes: 0
converge_tokens: 0
findings_blocker: 0
findings_should_fix: 0
findings_nit: 0
tickets_reopened: 0
```
