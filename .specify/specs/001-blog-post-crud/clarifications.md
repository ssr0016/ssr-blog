# Clarifications: Blog Post CRUD

**Date:** 2026-09-20
**Spec:** 001-blog-post-crud

Each item below was an open question in the first draft of spec.md. The
decision is recorded here so the plan can proceed without ambiguity.

## Blockers

1. **Slug collision.** Numeric suffix with DB-level unique index and
   service-level retry on unique-violation. Rejected as too admin-hostile.
   The alternative (reject with conflict) is not used.

2. **Slug on title edit.** Immutable. No redirects. Rationale: simpler,
   matches the intent that shared links keep working. Admins who need a
   different slug must delete and recreate.

3. **Delete semantics.** Soft delete (`deleted_at TIMESTAMPTZ NULL`).
   Rationale: prevents accidental data loss, keeps the slug reserved so
   old links never resolve to a different post, and leaves room for a
   future restore ticket. Cost is one column and a filter on every query.

## Open questions

4. **Published date on unpublish.** Cleared on return to draft, reset on
   re-publish. Rationale: keeps the meaning of `published_at` simple
   ("when it last became published").

5. **Editor role.** Admin only. The seeded `editor` role is not granted
   write access in this ticket. Rationale: separate concern, separate
   ticket if needed.

6. **Author attribution.** None in MVP. Posts do not record who wrote
   them. Rationale: single-admin site for now.

7. **Field limits.**
   - Title: 200
   - Slug: 200 (derived from title)
   - Content: 100,000
   - Excerpt: 500
   - Cover image URL: 2048

8. **Non-Latin titles.** Rejected with a validation error when the
   generated slug would be empty. Rationale: no transliteration scheme
   in MVP; can be added later without breaking the API.

9. **Public list contents.** Summaries only (title, slug, excerpt,
   cover image URL, published date). No full content in list responses.

10. **Admin backdating.** System sets `published_at` only. No
    admin-supplied or backdated values.
