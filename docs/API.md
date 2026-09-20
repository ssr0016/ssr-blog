# API Documentation

Base URL: http://localhost:8080/api/v1

## Authentication Flow

### 1. Register

POST /auth/register

    {
      "email": "user@example.com",
      "name": "User Name",
      "password": "StrongPass123!"
    }

Response (201):

    {
      "id": 1,
      "email": "user@example.com",
      "name": "User Name",
      "role_id": 1,
      "created_at": "2026-01-01T00:00:00Z"
    }

### 2. Login

POST /auth/login

    {
      "email": "user@example.com",
      "password": "StrongPass123!"
    }

Response (200): Same as register. Sets app_session cookie.

### 3. Authenticated Requests

Include session cookie with each request:

    curl -b cookies.txt http://localhost:8080/api/v1/auth/me

### 4. Logout

POST /auth/logout

Response (200):

    {"message": "logged out"}

## Users

### List Users (paginated)

GET /users?page=1&limit=10&email=user

Response (200):

    {
      "data": [...],
      "meta": {
        "page": 1,
        "limit": 10,
        "total": 42,
        "total_pages": 5
      }
    }

### Get User

GET /users/:id

## Admin Endpoints

Requires role: admin.

### List Roles (paginated)

GET /admin/roles?page=1&limit=10

### Create Role

POST /admin/roles

    {
      "name": "manager",
      "description": "Manager role"
    }

### Assign Permission

POST /admin/roles/:id/permissions

    {
      "permission_id": 1
    }

### List Permissions (paginated)

GET /admin/permissions?page=1&limit=10&resource=users

### Change User Role

PUT /admin/users/:id/role

    {
      "role_id": 2
    }

## Error Responses

### 400 Bad Request

    {"error": "BAD_REQUEST", "message": "invalid request body"}

### 401 Unauthorized

    {"error": "UNAUTHORIZED", "message": "authentication required"}

### 403 Forbidden

    {"error": "FORBIDDEN", "message": "insufficient permissions"}

### 404 Not Found

    {"error": "NOT_FOUND", "message": "resource not found"}

### 409 Conflict

    {"error": "CONFLICT", "message": "email already taken"}

### 422 Validation Error

    {
      "error": "VALIDATION_ERROR",
      "message": "Key: 'RegisterRequest.Email' Error:Field validation..."
    }

### 429 Rate Limit

    {"error": "RATE_LIMIT_EXCEEDED", "message": "rate limit exceeded"}

## CSRF Protection

For POST/PUT/DELETE requests, include CSRF token:

1. GET /health - receives csrf_token cookie
2. Include header: X-CSRF-Token: <token>

Example:

    curl -c cookies.txt http://localhost:8080/health > /dev/null
    CSRF=$(grep csrf_token cookies.txt | awk '{print $NF}')

    curl -X POST http://localhost:8080/api/v1/auth/login \
      -H "Content-Type: application/json" \
      -H "X-CSRF-Token: $CSRF" \
      -b cookies.txt -c cookies.txt \
      -d '{"email": "admin@example.com", "password": "AdminPass123!"}'

## Complete Example

Full auth flow:

    # 1. Get CSRF token
    curl -c cookies.txt http://localhost:8080/health > /dev/null
    CSRF=$(grep csrf_token cookies.txt | awk '{print $NF}')

    # 2. Register
    curl -X POST http://localhost:8080/api/v1/auth/register \
      -H "Content-Type: application/json" \
      -H "X-CSRF-Token: $CSRF" \
      -b cookies.txt -c cookies.txt \
      -d '{"email": "user@example.com", "name": "User", "password": "StrongPass123!"}'

    # 3. Login
    curl -X POST http://localhost:8080/api/v1/auth/login \
      -H "Content-Type: application/json" \
      -H "X-CSRF-Token: $CSRF" \
      -b cookies.txt -c cookies.txt \
      -d '{"email": "user@example.com", "password": "StrongPass123!"}'

    # 4. Get current user
    curl -b cookies.txt http://localhost:8080/api/v1/auth/me

    # 5. List users (paginated)
    curl -b cookies.txt "http://localhost:8080/api/v1/users?page=1&limit=10"

---

## Email Verification

Registration automatically generates a verification token.

In dev: token is logged via slog:
  msg="verification token generated" user_id=X token=XYZ

### Verify Email

GET /api/v1/auth/verify-email?token=XXX

Response (200):
  {"message": "email verified successfully"}

Errors:
- 400 - invalid or expired token
- 400 - token already used

## Password Reset

### Request Reset

POST /api/v1/auth/forgot-password

Request:
  {"email": "user@example.com"}

Response (200):
  {"message": "if the email exists, a reset link has been sent"}

Note: Always returns 200 (security - don't reveal if email exists).

### Reset Password

POST /api/v1/auth/reset-password

Request:
  {
    "token": "reset-token-from-email",
    "new_password": "NewStrongPass123x"
  }

Response (200):
  {"message": "password reset successful"}

Errors:
- 400 - invalid or expired token
- 400 - token already used

## Account Lockout

After 5 failed login attempts, account is locked for 15 minutes.

### Login When Locked

POST /api/v1/auth/login

Response (403):
  {"error": "FORBIDDEN", "message": "account is locked, try again later"}

### Auto-Unlock

After 15 minutes, account auto-unlocks.
Successful login resets the counter.

### Check Lock Status

SQL:
  SELECT email, failed_login_attempts, locked_until 
  FROM users WHERE email = 'user@example.com';

## Complete Auth Flow

Full example with email verify + password reset:

1. Register
   POST /api/v1/auth/register
   Body: {"email":"user@example.com","name":"User","password":"StrongPass123"}

2. Check token in logs (dev) or email (prod)
   Look for: "verification token generated"

3. Verify email
   GET /api/v1/auth/verify-email?token=XXX

4. Login
   POST /api/v1/auth/login
   Body: {"email":"user@example.com","password":"StrongPass123"}

5. Forgot password (if needed)
   POST /api/v1/auth/forgot-password
   Body: {"email":"user@example.com"}

6. Reset password (get token from logs/email)
   POST /api/v1/auth/reset-password
   Body: {"token":"XXX","new_password":"NewStrongPass123x"}

## Pagination

All list endpoints support pagination:

| Param | Default | Max |
|---|---|---|
| page | 1 | - |
| limit | 20 | 100 |

Examples:
  GET /api/v1/users?page=2&limit=10
  GET /api/v1/admin/roles?page=1&limit=5
  GET /api/v1/admin/permissions?resource=users&page=1&limit=10

Response:
  {
    "data": [...],
    "meta": {
      "page": 2,
      "limit": 10,
      "total": 42,
      "total_pages": 5
    }
  }

## RBAC (Admin Endpoints)

Requires role: admin.

### List Roles
  GET /api/v1/admin/roles?page=1&limit=10

### Create Role
  POST /api/v1/admin/roles
  Body: {"name":"manager","description":"Manager role"}

### Assign Permission to Role
  POST /api/v1/admin/roles/:id/permissions
  Body: {"permission_id": 1}

### Revoke Permission
  DELETE /api/v1/admin/roles/:id/permissions/:pid

### List Permissions
  GET /api/v1/admin/permissions?page=1&limit=10&resource=users

### Create Permission
  POST /api/v1/admin/permissions
  Body: {"name":"posts:read","resource":"posts","action":"read"}

### Change User Role
  PUT /api/v1/admin/users/:id/role
  Body: {"role_id": 2}

## Posts (Public)

No login, CSRF token or rate limit is needed. Only published posts that are not deleted are visible here.

The examples assume a published post with the slug hello-world exists. Create and publish one with the admin endpoints below.

### List Published Posts (paginated)

GET /posts?page=1&limit=20

    curl "http://localhost:8080/api/v1/posts?page=1&limit=20"

Response (200):

    {
      "data": [
        {
          "title": "Hello World",
          "slug": "hello-world",
          "excerpt": "First post",
          "cover_image_url": "https://example.com/cover.png",
          "published_at": "2026-01-01T10:00:00Z"
        }
      ],
      "meta": {
        "page": 1,
        "limit": 20,
        "total": 1,
        "total_pages": 1
      }
    }

- Newest published first (published_at, then id, descending).
- Items are summaries: title, slug, excerpt, cover_image_url, published_at. There is no content; fetch a single post for that.
- Default limit is 20, max is 100. Out-of-range page and limit values are clamped, not rejected.
- No posts (or a page past the end) returns 200 with "data": [] and the real total.

### Get Post by Slug

GET /posts/:slug

    curl http://localhost:8080/api/v1/posts/hello-world

Response (200):

    {
      "id": 1,
      "title": "Hello World",
      "slug": "hello-world",
      "content": "# Hello\n\nMy first post.",
      "excerpt": "First post",
      "cover_image_url": "https://example.com/cover.png",
      "status": "published",
      "published_at": "2026-01-01T10:00:00Z",
      "created_at": "2026-01-01T09:00:00Z",
      "updated_at": "2026-01-01T10:00:00Z"
    }

Response (404):

    {"error": "NOT_FOUND", "message": "post not found"}

A draft, a deleted post and a slug that never existed all return the exact same 404 body, so a caller cannot tell them apart.

Content is raw markdown, stored exactly as written. It is not rendered or sanitized and may contain HTML. Clients must render it safely (sanitize the HTML) before showing it.

## Admin: Posts

Requires role: admin (session cookie from login). POST, PUT and DELETE also need the CSRF header (see CSRF Protection above). Without a session the response is 401; a logged-in non-admin gets 403.

Setup for the examples below (login as admin, keep the cookies and CSRF token):

    curl -c cookies.txt http://localhost:8080/health > /dev/null
    CSRF=$(grep csrf_token cookies.txt | awk '{print $NF}')

    curl -X POST http://localhost:8080/api/v1/auth/login \
      -H "Content-Type: application/json" \
      -H "X-CSRF-Token: $CSRF" \
      -b cookies.txt -c cookies.txt \
      -d '{"email": "admin@example.com", "password": "AdminPass123!"}'

The examples use post id 1; use the id returned by Create Post instead.

Rules that apply to all admin post endpoints:

- The slug is generated by the server from the title (lowercase, accents removed, words joined by "-"). It never changes after creation. A "slug" field in a request body is ignored.
- If a slug is already taken, the server appends a number: hello-world, hello-world-2, hello-world-3. A deleted post keeps its slug reserved, so a slug is never reused. Because of this retry, post ids can have gaps.
- status is "draft" or "published". published_at is set by the server when a post is first published, kept while it stays published, and cleared (null) when it goes back to draft. Publishing again later sets a new time.
- POST, PUT and DELETE share one rate limit per client: 30 requests per minute, burst of 10. Over the limit returns 429. Reads are not limited.
- Success writes are recorded in the audit log (actor, action, post id, slug). Post content is never logged.

Fields:

| Field | Create | Update | Rule |
|---|---|---|---|
| title | required | required | not blank, max 200 chars, no NUL bytes |
| content | required | required | not blank, max 100000 chars, no NUL bytes, raw markdown |
| excerpt | optional | optional | max 500 chars, no NUL bytes |
| cover_image_url | optional | optional | http(s) URL, max 2048 chars |
| status | optional (default draft) | required | draft or published |

### Create Post

POST /admin/posts

    curl -X POST http://localhost:8080/api/v1/admin/posts \
      -H "Content-Type: application/json" \
      -H "X-CSRF-Token: $CSRF" \
      -b cookies.txt \
      -d '{
        "title": "Hello World",
        "content": "# Hello\n\nMy first post.",
        "excerpt": "First post",
        "cover_image_url": "https://example.com/cover.png"
      }'

Response (201):

    {
      "id": 1,
      "title": "Hello World",
      "slug": "hello-world",
      "content": "# Hello\n\nMy first post.",
      "excerpt": "First post",
      "cover_image_url": "https://example.com/cover.png",
      "status": "draft",
      "published_at": null,
      "created_at": "2026-01-01T09:00:00Z",
      "updated_at": "2026-01-01T09:00:00Z"
    }

Send "status": "published" to publish right away; published_at is then set to the current time.

Response (422): invalid or missing fields, or a title with no letters or digits for a slug:

    {
      "error": "VALIDATION_ERROR",
      "message": "Key: 'CreatePostRequest.Title' Error:Field validation for 'Title' failed on the 'required' tag"
    }

### List Posts (paginated)

GET /admin/posts?status=draft&page=1&limit=20

    curl -b cookies.txt "http://localhost:8080/api/v1/admin/posts?status=draft&page=1&limit=20"

Response (200):

    {
      "data": [
        {
          "id": 1,
          "title": "Hello World",
          "slug": "hello-world",
          "excerpt": "First post",
          "cover_image_url": "https://example.com/cover.png",
          "status": "draft",
          "published_at": null,
          "created_at": "2026-01-01T09:00:00Z",
          "updated_at": "2026-01-01T09:00:00Z"
        }
      ],
      "meta": {
        "page": 1,
        "limit": 20,
        "total": 1,
        "total_pages": 1
      }
    }

- Lists posts of every status, newest created first. Deleted posts are not listed.
- Optional status filter: draft or published. Leave it out to get all. Any other value returns 422.
- Items are summaries (no content). Default limit is 20, max is 100; out-of-range page and limit are clamped.

Response (422):

    {"error": "VALIDATION_ERROR", "message": "status must be draft or published"}

### Get Post

GET /admin/posts/:id

    curl -b cookies.txt http://localhost:8080/api/v1/admin/posts/1

Response (200): Same shape as Create Post, including drafts.

Response (400): the id is not a positive integer:

    {"error": "BAD_REQUEST", "message": "invalid post id"}

Response (404): unknown or deleted id:

    {"error": "NOT_FOUND", "message": "post not found"}

### Update Post (edit, publish, unpublish)

PUT /admin/posts/:id

This is a full replace of title, content, excerpt, cover_image_url and status. There is no partial update: an optional field you leave out is reset to empty. The slug is not changed.

    curl -X PUT http://localhost:8080/api/v1/admin/posts/1 \
      -H "Content-Type: application/json" \
      -H "X-CSRF-Token: $CSRF" \
      -b cookies.txt \
      -d '{
        "title": "Hello World",
        "content": "# Hello\n\nMy first post, now live.",
        "excerpt": "First post",
        "cover_image_url": "https://example.com/cover.png",
        "status": "published"
      }'

Response (200):

    {
      "id": 1,
      "title": "Hello World",
      "slug": "hello-world",
      "content": "# Hello\n\nMy first post, now live.",
      "excerpt": "First post",
      "cover_image_url": "https://example.com/cover.png",
      "status": "published",
      "published_at": "2026-01-01T10:00:00Z",
      "created_at": "2026-01-01T09:00:00Z",
      "updated_at": "2026-01-01T10:00:00Z"
    }

- draft to published: published_at is set to now.
- published to published (a normal edit): published_at is kept, however many other fields change.
- any status to draft: published_at becomes null and the post disappears from the public endpoints.

Unpublish (back to draft):

    curl -X PUT http://localhost:8080/api/v1/admin/posts/1 \
      -H "Content-Type: application/json" \
      -H "X-CSRF-Token: $CSRF" \
      -b cookies.txt \
      -d '{
        "title": "Hello World",
        "content": "# Hello\n\nMy first post, now live.",
        "excerpt": "First post",
        "cover_image_url": "https://example.com/cover.png",
        "status": "draft"
      }'

Response (200): Same shape, with "status": "draft" and "published_at": null.

Errors: 400 (bad id or malformed JSON), 404 (unknown or deleted id), 422 (validation, including a missing status).

### Delete Post

DELETE /admin/posts/:id

    curl -X DELETE http://localhost:8080/api/v1/admin/posts/1 \
      -H "X-CSRF-Token: $CSRF" \
      -b cookies.txt

Response (204): empty body.

This is a soft delete. The post disappears from every public and admin read, but the row is kept and its slug stays reserved, so a new post with the same title gets "-2". Deleting an unknown or already deleted post returns 404:

    {"error": "NOT_FOUND", "message": "post not found"}

### Errors for Post Endpoints

Error bodies always look like {"error": "CODE", "message": "..."}; see Error Responses above.

| Endpoint | Possible errors |
|---|---|
| GET /posts | 500 |
| GET /posts/:slug | 404, 500 |
| POST /admin/posts | 400, 401, 403, 422, 429, 500 |
| GET /admin/posts | 401, 403, 422, 500 |
| GET /admin/posts/:id | 400, 401, 403, 404, 500 |
| PUT /admin/posts/:id | 400, 401, 403, 404, 422, 429, 500 |
| DELETE /admin/posts/:id | 400, 401, 403, 404, 429, 500 |

A missing or wrong CSRF header on POST, PUT or DELETE returns 403 FORBIDDEN before anything else runs, with the message "CSRF token invalid or missing". A 500 returns a generic INTERNAL_ERROR message with no internal details. Post endpoints never return 409: a slug collision is resolved by adding a number.
