# Spec: Password History Check

**Date:** 2026-09-20
**Author:** ssr0016
**Status:** Draft

---

## Problem

Users can currently set a new password (via change-password or reset-password) that is identical to one of their recent previous passwords, weakening the security benefit of forced or voluntary password rotation.

## Solution

Track a history of each user's last 5 password hashes and reject any new password that matches one of them, on both password change and password reset flows.

## Behaviors

1. When a user successfully sets a new password (change-password or reset-password flow), the system stores the resulting password hash in that user's password history.
2. Before accepting a new password, the system checks it against the user's last 5 stored password hashes (including the current active password).
3. If the new password matches any of the last 5 password hashes, the request is rejected with a clear validation error and the password is not changed.
4. The system retains only the 5 most recent password hashes per user; older entries beyond the 5th are discarded when a new one is added.
5. Password history checks apply uniformly to both the authenticated change-password flow and the token-based password-reset flow.

## Edge Cases

- User has fewer than 5 passwords in history (e.g., brand-new account) -> check only against the entries that exist; new password is accepted if it doesn't match any of them.
- User tries to reset password to their current active password -> rejected, since the current password is included in the history check.
- Password history table has stale entries from before this feature existed -> only hashes recorded after this feature ships are guaranteed present; users with no prior history simply pass the check trivially.
- Two different users happen to pick the same password -> no cross-user comparison; history check is scoped per user only.
- User's account is deleted and later a new account is created with the same email -> new account starts with empty password history (no carryover).

## Non-Goals

- Not enforcing password complexity/strength rules (handled separately, if at all).
- Not adding a configurable history depth (fixed at last 5 passwords for this iteration).
- Not retroactively backfilling history for passwords set before this feature ships.
- Not exposing password history (hashes or metadata) via any API endpoint.

## Success Criteria

- [ ] User cannot change their password to any of their last 5 passwords (change-password flow).
- [ ] User cannot reset their password to any of their last 5 passwords (password-reset flow).
- [ ] A rejected reuse attempt returns a clear error and does not alter the user's current password or history.
- [ ] Only the 5 most recent password hashes are retained per user.
- [ ] Existing users with no history are not blocked from setting a new password.

## Open Questions

- [ ] Should the error message reveal *that* the password was reused (potential info disclosure) or use a generic "password does not meet requirements" message?
- [ ] Does "last 5 passwords" include the password being replaced (current active), or only the 4 prior to it plus current = 5 total? (Spec assumes current + 4 prior = 5.)
