# Spec: Blog Post CRUD

**Date:** 2026-09-20
**Author:** ssr0016
**Status:** Clarified

---

## Problem

The blog has authentication and roles but no content. There is nothing an admin can publish and nothing a visitor can read.

## Solution

Let admins create, edit, and delete blog posts, with drafts kept private until published. Let anyone read published posts without logging in. Each post gets a unique, human-readable slug generated from its title, so posts have stable, shareable links.

## Behaviors

A post has: title, slug, content (markdown), excerpt, cover image URL, status (`draft` or `published`), and published date.

**Admin: writing**

1. An admin can create a post. Title and content are required. Excerpt and cover image URL are optional. Status defaults to `draft`.
2. The slug is generated from the title. The admin does not supply it.
3. Slugs are unique across all posts, drafts and published. If the generated slug is already taken, the system appends a numeric suffix (e.g., `-2`, `-3`) and retries until unique. The post is not rejected for a slug collision. Uniqueness is enforced at the database level with a unique index, and the service retries on unique-violation errors to handle concurrent creation safely.
4. A slug never changes after creation, even if the title is edited, so links already shared keep working.
5. An admin can edit a post's title, content, excerpt, cover image URL, and status.
6. The published date is set by the system the first time a post becomes `published`. Editing a published post does not change it. Returning a post to `draft` clears it. Re-publishing sets it again to the current time.
7. An admin can delete a post. Deletion is soft: the post is marked deleted and hidden from all admin and public reads. Its slug stays reserved so it is never reused by a future post. A future ticket may add restore.
8. An admin can view any non-deleted post regardless of status, and can list all non-deleted posts, including drafts, filtered by status.

**Public: reading**

9. Anyone, logged in or not, can list published posts. Newest published first, paginated using the project's standard format (data + meta, default limit 20, max 100).
10. Anyone can read a single published post by its slug and receives all its fields.
11. The public list shows post summaries (title, slug, excerpt, cover image URL, published date) and does not include full content.
12. Drafts are invisible to the public. Requesting a draft's slug gives the same "not found" response as a slug that does not exist, so a draft's existence is not revealed.

**Access control**

13. Create, edit, and delete are restricted to admins. Unauthenticated callers are told authentication is required. Authenticated non-admins are told they are forbidden. In both cases nothing is changed. The `editor` role does not have write access.

**Content**

14. Content is stored and returned exactly as written (markdown source). The system does not render or transform it.
15. The excerpt is whatever the admin writes. It is not auto-generated. If omitted, it is empty.

## Field limits

- Title: 200 characters
- Slug: derived, max 200 characters (matches title)
- Content: 100,000 characters
- Excerpt: 500 characters
- Cover image URL: 2048 characters

Limits are enforced by request validation at the handler layer.

## Edge Cases

- Two posts with the same title -> the second gets a different, unique slug. Both are created.
- Two admins create posts with the same title at the same moment -> both succeed with different slugs. The database unique index rejects the losing insert and the service retries with the next suffix. There are never duplicate slugs and neither request fails on a slug conflict.
- Title is edited after publishing -> slug is unchanged. Old links still resolve.
- Title has no letters or numbers (e.g., `"!!!"`) -> creation is rejected with a validation error. Nothing is saved.
- Title has accents or mixed case (e.g., `"Café Au Lait"`) -> slug is lowercase and URL-safe (e.g., `cafe-au-lait`).
- Title is non-Latin (e.g., Japanese or Arabic) and produces an empty slug -> creation is rejected with a validation error. Nothing is saved.
- Title, or content missing or blank -> validation error. Nothing is saved.
- Title, content, excerpt, or cover image URL over the maximum length -> validation error. Nothing is saved.
- Cover image URL is not a valid `http`/`https` URL -> validation error. An omitted or empty URL is allowed.
- Status is anything other than `draft` or `published` -> validation error.
- Publish an already-published post -> succeeds. Published date is unchanged.
- Move a published post back to draft -> disappears from public list and public read (not found). Published date is cleared.
- Re-publish a post that was reverted to draft -> published date is set to the time of re-publishing.
- Public request for a draft's slug -> not found, identical to a nonexistent slug.
- Public request for a slug that does not exist -> not found.
- Public request for a soft-deleted post's slug -> not found.
- Admin edits or deletes a post ID that does not exist -> not found.
- Admin views or lists a soft-deleted post -> not found / excluded from list.
- Admin deletes a published post -> no longer in the public list. Its public link returns not found.
- Admin deletes a post, then creates a new post whose title would produce the same slug -> the new post gets a suffixed slug; the deleted post's slug is never reused.
- Public list has no published posts -> empty data with valid pagination meta. Not an error.
- Public list page beyond the last page -> empty data with valid pagination meta.
- Pagination limit above 100 or below 1 -> handled the same way as other paginated endpoints in the project.
- Logged-in non-admin (e.g., `editor` or `user` role) attempts create/edit/delete -> forbidden. Nothing changes.
- Unauthenticated caller attempts create/edit/delete -> authentication required. Nothing changes.
- Malicious markdown or HTML in content (e.g., a script tag) -> stored as written. The system does not render it, so clients must render it safely.

## Non-Goals

- Comments, likes, or any reader interaction.
- Categories, tags, or search.
- Multiple authors, author profiles, or author bylines.
- Scheduled publishing (a future published date). Publishing happens when the admin sets status to `published`.
- Admin-supplied or backdated published dates. The system always sets it.
- Revision history or undo of edits and deletes.
- Restoring soft-deleted posts (a future ticket may add this).
- Image upload or hosting. The cover image is a URL only.
- Server-side markdown rendering or HTML sanitization.
- Editing a slug by hand.
- Slug redirects after a title edit (slugs are immutable).
- Bulk operations (bulk publish or delete).
- Access for the `editor` role or any role other than `admin`.
- RSS/Atom feeds, sitemaps, view counts, or analytics.

## Success Criteria

- [ ] An admin can create, edit, list, view, and soft-delete posts, including drafts.
- [ ] A new post gets a slug from its title. Slugs are unique, including under simultaneous creation.
- [ ] A slug does not change when the title is edited.
- [ ] Published date is set on first publish, kept on later edits, and cleared on return to draft.
- [ ] Deleted posts are hidden from all reads and their slugs are never reused.
- [ ] Anonymous visitors can list and read published posts.
- [ ] Anonymous visitors cannot tell that a draft exists.
- [ ] Non-admins and anonymous callers cannot create, edit, or delete. Nothing changes when they try.
- [ ] Every validation edge case above returns a clear error and saves nothing.
- [ ] Public list uses the standard pagination format and limits.
- [ ] Tests cover admin-only access and each validation and not-found path.
- [ ] Repository behavior, including slug uniqueness under concurrent creation and soft-delete exclusion, is covered by integration tests against real Postgres.
- [ ] Swagger regenerated and `docs/API.md` updated, per the project's API-change rules.
- [ ] `make lint` and `make test` pass with no warnings or failures.

## Open Questions

Resolved in `.specify/specs/001-blog-post-crud/clarifications.md`. No open questions remain for planning.
