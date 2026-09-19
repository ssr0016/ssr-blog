# Plan: [Feature Name]

**Spec:** ./spec.md
**Date:** YYYY-MM-DD

---

## Files to Touch

| File | Change | Risk |
|---|---|---|
| internal/service/xxx.go | Add method | Low |
| internal/repository/xxx.go | Add query | Medium |
| internal/handler/xxx.go | Add endpoint | Low |
| internal/database/migrations/NNNN.sql | New table | High |

## Order

1. Migration (DB first)
2. Repository (data access)
3. Service (business logic)
4. Handler (HTTP)
5. Swagger regen

## Risks

- [Risk 1] -> [mitigation]
- [Risk 2] -> [mitigation]

## Test Strategy

- Unit: [what]
- Integration: [what]
- Manual: [what]

## Rollback

[Paano i-rollback kung may problema?]
