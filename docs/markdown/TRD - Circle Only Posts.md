# TRD - Circle Only Posts

> **Feature**: Circle Only Posts  
> **Backend Engineer**: Furqan (supported by Luthfi Alfari)  
> **Status**: In Progress  
> **Last Updated**: April 10, 2026  
> **Version**: 2.0

---

## 1. Overview

### Goal

Enable users to create and manage private groups called **"Circles"** (similar to Instagram's Close Friends), and share posts that are **exclusively visible** to members of a selected Circle. This feature enforces strict content privacy at the database query level, ensuring non-members can never access Circle-restricted posts.

### User Behavior

- Users can **create multiple Circles** (e.g., "Close Friends", "Study Group", "Family").
- Users can **add or remove members** from their Circles at any time.
- When creating a post, users can choose its **visibility**:
  - `public` — Visible to everyone on the platform.
  - `circle` — Visible only to members of the selected Circle.
- Circle posts appear in the **feed of Circle members only**.
- Circle members can view **who liked and commented** on Circle posts.
- Non-members receive **no indication** that a Circle post exists (zero leakage).

---

## 2. Database Schema

### Entity Relationship Overview

```text
┌──────────┐       ┌─────────────────┐       ┌──────────┐
│  users   │──1:N──│    circles      │──1:N──│  posts   │
│          │       │                 │       │          │
│          │──M:N──│ circle_members  │       │          │
└──────────┘       └─────────────────┘       └──────────┘
                                                  │
                                           ┌──────┴──────┐
                                           │             │
                                     ┌───────────┐ ┌──────────────┐
                                     │post_likes │ │post_comments │
                                     └───────────┘ └──────────────┘
```

---

### Table 1: `circles`

Stores user-created private groups. Each circle is **owned by one user** who has full administrative control.

| Column | Type | Constraint | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `id` | UUID | PRIMARY KEY | `gen_random_uuid()` | Unique identifier |
| `owner_id` | UUID | NOT NULL, FK → `users(id)` | — | The user who created and owns this Circle |
| `name` | VARCHAR(50) | NOT NULL | — | Display name of the Circle |
| `description` | VARCHAR(255) | — | `NULL` | Optional short description |
| `avatar_url` | TEXT | — | `NULL` | Optional Circle profile image URL |
| `member_count` | INTEGER | NOT NULL | `1` | Denormalized count for performance |
| `created_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | Last modification timestamp |

**Constraints:**
- `FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE`
- `CHECK (char_length(name) >= 1)` — Name cannot be empty.

---

### Table 2: `circle_members`

Junction table implementing the **many-to-many** relationship between users and Circles.

| Column | Type | Constraint | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `id` | UUID | PRIMARY KEY | `gen_random_uuid()` | Unique identifier |
| `circle_id` | UUID | NOT NULL, FK → `circles(id)` | — | The Circle being joined |
| `user_id` | UUID | NOT NULL, FK → `users(id)` | — | The member user |
| `role` | VARCHAR(20) | NOT NULL | `'member'` | Either `'owner'` or `'member'` |
| `joined_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | Timestamp of membership |

**Constraints:**
- `UNIQUE (circle_id, user_id)` — A user can only join a Circle once.
- `FOREIGN KEY (circle_id) REFERENCES circles(id) ON DELETE CASCADE`
- `FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`
- `CHECK (role IN ('owner', 'member'))` — Only valid roles.

---

### Table 3: `posts`

Core content table. Supports both public and Circle-restricted posts through the `visibility` column.

| Column | Type | Constraint | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `id` | UUID | PRIMARY KEY | `gen_random_uuid()` | Unique identifier |
| `author_id` | UUID | NOT NULL, FK → `users(id)` | — | The user who created the post |
| `content` | TEXT | — | `NULL` | Text content of the post |
| `media_url` | TEXT | — | `NULL` | URL of attached media (image/video) |
| `visibility` | VARCHAR(10) | NOT NULL | `'public'` | `'public'` or `'circle'` |
| `circle_id` | UUID | FK → `circles(id)` | `NULL` | Target Circle (required when visibility = `'circle'`) |
| `like_count` | INTEGER | NOT NULL | `0` | Denormalized count for performance |
| `comment_count` | INTEGER | NOT NULL | `0` | Denormalized count for performance |
| `created_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | Last modification timestamp |

**Constraints:**
- `FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE`
- `FOREIGN KEY (circle_id) REFERENCES circles(id) ON DELETE SET NULL`
- `CHECK (visibility IN ('public', 'circle'))` — Only valid visibilities.
- `CHECK ((visibility = 'circle' AND circle_id IS NOT NULL) OR (visibility = 'public' AND circle_id IS NULL))` — Enforces data integrity: circle posts MUST have a circle_id, public posts MUST NOT.

---

### Table 4: `post_likes`

Tracks which users liked which posts.

| Column | Type | Constraint | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `id` | UUID | PRIMARY KEY | `gen_random_uuid()` | Unique identifier |
| `post_id` | UUID | NOT NULL, FK → `posts(id)` | — | The liked post |
| `user_id` | UUID | NOT NULL, FK → `users(id)` | — | The user who liked it |
| `created_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | Timestamp of the like |

**Constraints:**
- `UNIQUE (post_id, user_id)` — A user can only like a post once.
- `FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE`
- `FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`

---

### Table 5: `post_comments`

Stores comments on posts.

| Column | Type | Constraint | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `id` | UUID | PRIMARY KEY | `gen_random_uuid()` | Unique identifier |
| `post_id` | UUID | NOT NULL, FK → `posts(id)` | — | The parent post |
| `user_id` | UUID | NOT NULL, FK → `users(id)` | — | The comment author |
| `content` | TEXT | NOT NULL | — | Comment text |
| `created_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | Last edit timestamp |

**Constraints:**
- `FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE`
- `FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`
- `CHECK (char_length(content) >= 1)` — Comment cannot be empty.

---

### Indexes

| Index Name | Table | Column(s) | Purpose |
| :--- | :--- | :--- | :--- |
| `idx_circles_owner` | `circles` | `owner_id` | Fast lookup of Circles owned by a user |
| `idx_circle_members_circle` | `circle_members` | `circle_id` | Fast member listing per Circle |
| `idx_circle_members_user` | `circle_members` | `user_id` | Fast lookup of all Circles a user belongs to |
| `idx_posts_author` | `posts` | `author_id` | Fast lookup of posts by author |
| `idx_posts_circle` | `posts` | `circle_id` | Fast lookup of posts within a Circle |
| `idx_posts_visibility` | `posts` | `visibility, created_at DESC` | Efficient feed query filtering |
| `idx_post_likes_post` | `post_likes` | `post_id` | Fast like count and listing |
| `idx_post_likes_user` | `post_likes` | `user_id` | Check if user already liked a post |
| `idx_post_comments_post` | `post_comments` | `post_id, created_at` | Chronological comment listing |

---

### SQL Migration Script

```sql
-- Migration: 003_circle_only_posts.sql
-- Description: Create Circle system, Posts, Likes, and Comments tables
-- Author: Furqan / Luthfi Alfari

-- ============================
-- 1. Circles
-- ============================
CREATE TABLE IF NOT EXISTS circles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            VARCHAR(50) NOT NULL,
    description     VARCHAR(255),
    avatar_url      TEXT,
    member_count    INTEGER NOT NULL DEFAULT 1,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_circle_name_not_empty CHECK (char_length(name) >= 1)
);

-- ============================
-- 2. Circle Members (Junction)
-- ============================
CREATE TABLE IF NOT EXISTS circle_members (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    circle_id   UUID NOT NULL REFERENCES circles(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        VARCHAR(20) NOT NULL DEFAULT 'member',
    joined_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_circle_member UNIQUE (circle_id, user_id),
    CONSTRAINT chk_member_role CHECK (role IN ('owner', 'member'))
);

-- ============================
-- 3. Posts
-- ============================
CREATE TABLE IF NOT EXISTS posts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content         TEXT,
    media_url       TEXT,
    visibility      VARCHAR(10) NOT NULL DEFAULT 'public',
    circle_id       UUID REFERENCES circles(id) ON DELETE SET NULL,
    like_count      INTEGER NOT NULL DEFAULT 0,
    comment_count   INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_visibility CHECK (visibility IN ('public', 'circle')),
    CONSTRAINT chk_circle_visibility_integrity CHECK (
        (visibility = 'circle' AND circle_id IS NOT NULL) OR
        (visibility = 'public' AND circle_id IS NULL)
    )
);

-- ============================
-- 4. Post Likes
-- ============================
CREATE TABLE IF NOT EXISTS post_likes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id     UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_post_like UNIQUE (post_id, user_id)
);

-- ============================
-- 5. Post Comments
-- ============================
CREATE TABLE IF NOT EXISTS post_comments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id     UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_comment_not_empty CHECK (char_length(content) >= 1)
);

-- ============================
-- 6. Indexes
-- ============================
CREATE INDEX IF NOT EXISTS idx_circles_owner ON circles(owner_id);
CREATE INDEX IF NOT EXISTS idx_circle_members_circle ON circle_members(circle_id);
CREATE INDEX IF NOT EXISTS idx_circle_members_user ON circle_members(user_id);
CREATE INDEX IF NOT EXISTS idx_posts_author ON posts(author_id);
CREATE INDEX IF NOT EXISTS idx_posts_circle ON posts(circle_id);
CREATE INDEX IF NOT EXISTS idx_posts_visibility ON posts(visibility, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_post_likes_post ON post_likes(post_id);
CREATE INDEX IF NOT EXISTS idx_post_likes_user ON post_likes(user_id);
CREATE INDEX IF NOT EXISTS idx_post_comments_post ON post_comments(post_id, created_at);
```

### Design Decisions

| Decision | Rationale |
| :--- | :--- |
| **Denormalized `member_count`, `like_count`, `comment_count`** | Avoids expensive `COUNT(*)` aggregate queries on every feed/list request. Updated via application-level increment/decrement inside transactions. |
| **`ON DELETE SET NULL` for `posts.circle_id`** | If a Circle is deleted, posts are preserved but become inaccessible (orphaned). An alternative is `CASCADE` to delete all posts, but preserving content is safer. Application layer should handle cleanup. |
| **`CHECK` constraint on visibility + circle_id** | Enforces data integrity at the database level — impossible to have a `circle` post without a `circle_id`, or a `public` post with one. |
| **Separate `post_likes` and `post_comments` tables** | Normalized design allows efficient querying, indexing, and pagination. Avoids JSON arrays which are difficult to query and index. |
| **`role` column in `circle_members`** | Enables future permission granularity (e.g., `admin`, `moderator`) without schema changes. |
| **Composite index `(visibility, created_at DESC)`** | Optimized for the most common query pattern: fetching recent public posts for the global feed. |

---

## 3. API Design

### Base URL

```text
/api/v1
```

*Note: All endpoints require **JWT Authentication** via header:*
```text
Authorization: Bearer <JWT_TOKEN>
```

---

### Module A: Circle Management

---

#### A1. Create a Circle

`POST /api/v1/circles`

**Description**: Creates a new Circle. The authenticated user automatically becomes the owner and first member.

**Request Body:**
```json
{
    "name": "Close Friends",
    "description": "My inner circle"
}
```

| Field | Type | Required | Validation |
| :--- | :--- | :--- | :--- |
| `name` | string | ✅ Yes | Min 1, Max 50 characters |
| `description` | string | ❌ No | Max 255 characters |

**Response 201 (Created):**
```json
{
    "id": "a1b2c3d4-...",
    "name": "Close Friends",
    "description": "My inner circle",
    "member_count": 1,
    "role": "owner",
    "created_at": "2026-04-10T12:00:00Z"
}
```

**Response 400**: `"circle name is required"` / `"circle name too long (max 50 characters)"`  
**Response 400**: `"maximum circles limit reached (max 20)"`  
**Response 401**: `"unauthorized"`

---

#### A2. List My Circles

`GET /api/v1/circles`

**Description**: Returns all Circles the authenticated user belongs to (both owned and joined).

**Response 200:**
```json
{
    "circles": [
        {
            "id": "a1b2c3d4-...",
            "name": "Close Friends",
            "description": "My inner circle",
            "avatar_url": null,
            "member_count": 5,
            "role": "owner",
            "created_at": "2026-04-10T12:00:00Z"
        },
        {
            "id": "e5f6g7h8-...",
            "name": "Study Group",
            "description": null,
            "avatar_url": null,
            "member_count": 12,
            "role": "member",
            "created_at": "2026-04-09T08:00:00Z"
        }
    ],
    "total": 2
}
```

---

#### A3. Get Circle Details

`GET /api/v1/circles/:circleId`

**Description**: Returns detailed information about a specific Circle, including the member list. Only accessible to Circle members.

**Response 200:**
```json
{
    "id": "a1b2c3d4-...",
    "name": "Close Friends",
    "description": "My inner circle",
    "avatar_url": null,
    "owner": {
        "id": "user-uuid-...",
        "username": "luthfi",
        "avatar_url": "https://..."
    },
    "member_count": 5,
    "members": [
        {
            "id": "user-uuid-...",
            "username": "furqan",
            "avatar_url": "https://...",
            "role": "member",
            "joined_at": "2026-04-10T12:00:00Z"
        }
    ],
    "created_at": "2026-04-10T12:00:00Z"
}
```

**Response 403**: `"you are not a member of this circle"`  
**Response 404**: `"circle not found"`

---

#### A4. Update Circle

`PUT /api/v1/circles/:circleId`

**Description**: Updates Circle details. Only the **Circle owner** can perform this action.

**Request Body:**
```json
{
    "name": "Best Friends",
    "description": "Updated description"
}
```

**Response 200**: Updated circle object.  
**Response 403**: `"only the circle owner can update this circle"`  
**Response 404**: `"circle not found"`

---

#### A5. Delete Circle

`DELETE /api/v1/circles/:circleId`

**Description**: Permanently deletes a Circle and removes all memberships. Posts created within this Circle will have their `circle_id` set to `NULL` (orphaned). Only the **Circle owner** can perform this action.

**Response 200:**
```json
{
    "message": "circle deleted successfully"
}
```

**Response 403**: `"only the circle owner can delete this circle"`  
**Response 404**: `"circle not found"`

---

#### A6. Add Members to Circle

`POST /api/v1/circles/:circleId/members`

**Description**: Adds one or more users to a Circle. Only the **Circle owner** can add members.

**Request Body:**
```json
{
    "user_ids": ["user-uuid-1", "user-uuid-2"]
}
```

| Field | Type | Required | Validation |
| :--- | :--- | :--- | :--- |
| `user_ids` | string[] | ✅ Yes | Min 1, Max 50 users per request |

**Response 200:**
```json
{
    "message": "members added successfully",
    "added_count": 2,
    "member_count": 7
}
```

**Response 400**: `"user_ids is required"` / `"too many users (max 50 per request)"`  
**Response 400**: `"circle member limit reached (max 150)"`  
**Response 403**: `"only the circle owner can add members"`  
**Response 404**: `"circle not found"` / `"one or more users not found"`

---

#### A7. Remove Member from Circle

`DELETE /api/v1/circles/:circleId/members/:userId`

**Description**: Removes a member from a Circle. The Circle **owner** can remove anyone. A **member** can only remove themselves (leave the Circle).

**Response 200:**
```json
{
    "message": "member removed successfully",
    "member_count": 6
}
```

**Response 400**: `"circle owner cannot be removed (delete the circle instead)"`  
**Response 403**: `"you can only remove yourself from this circle"`  
**Response 404**: `"circle not found"` / `"user is not a member of this circle"`

---

### Module B: Circle Posts

---

#### B1. Create a Post

`POST /api/v1/posts`

**Description**: Creates a new post. If `visibility` is set to `"circle"`, a valid `circle_id` must be provided and the author must be a member of that Circle.

**Request Body (Public Post):**
```json
{
    "content": "Hello, world!",
    "media_url": "https://cdn.skiix.com/image.jpg",
    "visibility": "public"
}
```

**Request Body (Circle Post):**
```json
{
    "content": "This is only for my close friends!",
    "media_url": null,
    "visibility": "circle",
    "circle_id": "a1b2c3d4-..."
}
```

| Field | Type | Required | Validation |
| :--- | :--- | :--- | :--- |
| `content` | string | ❌ Conditional | Required if `media_url` is null. Max 5000 chars. |
| `media_url` | string | ❌ Conditional | Required if `content` is null. Must be a valid URL. |
| `visibility` | string | ✅ Yes | `"public"` or `"circle"` |
| `circle_id` | string | ❌ Conditional | Required when `visibility` = `"circle"`. Must be a valid Circle UUID. |

**Response 201 (Created):**
```json
{
    "id": "post-uuid-...",
    "author": {
        "id": "user-uuid-...",
        "username": "luthfi"
    },
    "content": "This is only for my close friends!",
    "media_url": null,
    "visibility": "circle",
    "circle": {
        "id": "a1b2c3d4-...",
        "name": "Close Friends"
    },
    "like_count": 0,
    "comment_count": 0,
    "created_at": "2026-04-10T12:00:00Z"
}
```

**Response 400**: `"content or media_url is required"` / `"circle_id is required for circle posts"`  
**Response 403**: `"you are not a member of this circle"`  
**Response 404**: `"circle not found"`

---

#### B2. Get Circle Feed

`GET /api/v1/circles/:circleId/posts?page=1&limit=20`

**Description**: Returns a paginated list of posts within a specific Circle. Only accessible to Circle members. Results are ordered by `created_at DESC` (newest first).

**Query Parameters:**

| Parameter | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `page` | integer | `1` | Page number |
| `limit` | integer | `20` | Posts per page (max 50) |

**Response 200:**
```json
{
    "posts": [
        {
            "id": "post-uuid-...",
            "author": {
                "id": "user-uuid-...",
                "username": "luthfi",
                "avatar_url": "https://..."
            },
            "content": "Circle-only content here",
            "media_url": null,
            "like_count": 3,
            "comment_count": 1,
            "is_liked": true,
            "created_at": "2026-04-10T12:00:00Z"
        }
    ],
    "pagination": {
        "page": 1,
        "limit": 20,
        "total_posts": 45,
        "total_pages": 3,
        "has_next": true
    }
}
```

**Response 403**: `"you are not a member of this circle"`  
**Response 404**: `"circle not found"`

---

#### B3. Delete a Post

`DELETE /api/v1/posts/:postId`

**Description**: Permanently deletes a post and all associated likes/comments. The **post author** can delete their own posts. The **Circle owner** can delete any post within their Circle.

**Response 200:**
```json
{
    "message": "post deleted successfully"
}
```

**Response 403**: `"you do not have permission to delete this post"`  
**Response 404**: `"post not found"`

---

### Module C: Post Interactions

---

#### C1. Like a Post

`POST /api/v1/posts/:postId/like`

**Description**: Likes a post. For Circle posts, the user must be a Circle member. Idempotent — liking an already-liked post returns success without duplicating.

**Response 200:**
```json
{
    "message": "post liked",
    "like_count": 4
}
```

**Response 403**: `"you are not a member of this circle"`  
**Response 404**: `"post not found"`

---

#### C2. Unlike a Post

`DELETE /api/v1/posts/:postId/like`

**Description**: Removes a like from a post.

**Response 200:**
```json
{
    "message": "post unliked",
    "like_count": 3
}
```

---

#### C3. Get Post Likes

`GET /api/v1/posts/:postId/likes?page=1&limit=20`

**Description**: Returns a paginated list of users who liked the post. For Circle posts, only Circle members can view this list.

**Response 200:**
```json
{
    "likes": [
        {
            "user": {
                "id": "user-uuid-...",
                "username": "furqan",
                "avatar_url": "https://..."
            },
            "created_at": "2026-04-10T13:00:00Z"
        }
    ],
    "total": 4
}
```

---

#### C4. Create a Comment

`POST /api/v1/posts/:postId/comments`

**Description**: Adds a comment to a post. For Circle posts, the user must be a Circle member.

**Request Body:**
```json
{
    "content": "Great post!"
}
```

| Field | Type | Required | Validation |
| :--- | :--- | :--- | :--- |
| `content` | string | ✅ Yes | Min 1, Max 2000 characters |

**Response 201:**
```json
{
    "id": "comment-uuid-...",
    "user": {
        "id": "user-uuid-...",
        "username": "furqan"
    },
    "content": "Great post!",
    "created_at": "2026-04-10T13:30:00Z"
}
```

---

#### C5. Get Post Comments

`GET /api/v1/posts/:postId/comments?page=1&limit=20`

**Description**: Returns a paginated list of comments on a post, ordered by `created_at ASC` (oldest first). For Circle posts, only Circle members can view comments.

**Response 200:**
```json
{
    "comments": [
        {
            "id": "comment-uuid-...",
            "user": {
                "id": "user-uuid-...",
                "username": "furqan",
                "avatar_url": "https://..."
            },
            "content": "Great post!",
            "created_at": "2026-04-10T13:30:00Z"
        }
    ],
    "pagination": {
        "page": 1,
        "limit": 20,
        "total_comments": 8,
        "total_pages": 1,
        "has_next": false
    }
}
```

---

#### C6. Delete a Comment

`DELETE /api/v1/posts/:postId/comments/:commentId`

**Description**: Deletes a comment. The **comment author** can delete their own comments. The **post author** and **Circle owner** can also delete any comment.

**Response 200:**
```json
{
    "message": "comment deleted successfully"
}
```

**Response 403**: `"you do not have permission to delete this comment"`

---

## 4. System Flow

### Flow 1: Create a Circle and Add Members

1. **User** sends `POST /api/v1/circles { "name": "Close Friends" }`.
2. **Handler** validates name length (1-50 chars) and checks circle count limit (max 20 per user).
3. **Repository** executes inside a **database transaction**:
   - `INSERT INTO circles (owner_id, name) VALUES ($1, $2) RETURNING id`
   - `INSERT INTO circle_members (circle_id, user_id, role) VALUES ($1, $2, 'owner')`
4. **Response**: Returns created Circle with `member_count: 1`.
5. **User** sends `POST /api/v1/circles/:id/members { "user_ids": ["uuid-1", "uuid-2"] }`.
6. **Usecase** validates: is the requester the owner? Are all user_ids valid? Would adding exceed 150 member limit?
7. **Repository** executes batch insert and increments `member_count`.

### Flow 2: Create a Circle-Only Post

1. **User** sends `POST /api/v1/posts { "content": "...", "visibility": "circle", "circle_id": "..." }`.
2. **Handler** validates required fields based on visibility.
3. **Usecase** performs authorization check:
   - Query `circle_members` to verify the author is a member of the target Circle.
   - If not a member → return `403 Forbidden`.
4. **Repository** inserts the post with `visibility = 'circle'` and the `circle_id`.
5. **Response**: Returns the created post.

### Flow 3: Privacy-Enforced Feed Query

1. **User** requests `GET /api/v1/circles/:circleId/posts`.
2. **Usecase** performs membership check:
   - `SELECT 1 FROM circle_members WHERE circle_id = $1 AND user_id = $2`
   - If no row → return `403 Forbidden`.
3. **Repository** queries posts:
   ```sql
   SELECT p.*, u.username, u.avatar_url,
          EXISTS(SELECT 1 FROM post_likes WHERE post_id = p.id AND user_id = $2) AS is_liked
   FROM posts p
   JOIN users u ON p.author_id = u.id
   WHERE p.circle_id = $1
   ORDER BY p.created_at DESC
   LIMIT $3 OFFSET $4
   ```
4. **Response**: Returns paginated posts with `is_liked` flag.

### Flow 4: Like/Comment with Privacy Check

1. **User** sends `POST /api/v1/posts/:postId/like`.
2. **Usecase** queries the post to determine its visibility.
3. If `visibility = 'circle'`:
   - Verify the user is a member of `post.circle_id`.
   - If not → return `403 Forbidden`.
4. **Repository** executes inside a **transaction**:
   - `INSERT INTO post_likes (post_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
   - `UPDATE posts SET like_count = like_count + 1 WHERE id = $1` (only if insert was successful)
5. **Response**: Returns updated `like_count`.

---

## 5. Authorization Matrix

| Action | Owner | Member | Non-Member |
| :--- | :--- | :--- | :--- |
| View Circle details | ✅ | ✅ | ❌ |
| Update Circle name/description | ✅ | ❌ | ❌ |
| Delete Circle | ✅ | ❌ | ❌ |
| Add members | ✅ | ❌ | ❌ |
| Remove other members | ✅ | ❌ | ❌ |
| Remove self (leave) | ❌ (must delete circle) | ✅ | ❌ |
| Create Circle post | ✅ | ✅ | ❌ |
| View Circle posts | ✅ | ✅ | ❌ |
| Delete own post | ✅ | ✅ | ❌ |
| Delete any post in Circle | ✅ | ❌ | ❌ |
| Like/Unlike Circle post | ✅ | ✅ | ❌ |
| View likes on Circle post | ✅ | ✅ | ❌ |
| Comment on Circle post | ✅ | ✅ | ❌ |
| Delete own comment | ✅ | ✅ | ❌ |
| Delete any comment in Circle | ✅ | ❌ | ❌ |

---

## 6. Architecture (Clean Architecture Layers)

### Domain Layer — `internal/domain/`

```go
// circle.go
type Circle struct {
    ID          string    `json:"id"`
    OwnerID     string    `json:"owner_id"`
    Name        string    `json:"name"`
    Description *string   `json:"description"`
    AvatarURL   *string   `json:"avatar_url"`
    MemberCount int       `json:"member_count"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type CircleMember struct {
    ID       string    `json:"id"`
    CircleID string    `json:"circle_id"`
    UserID   string    `json:"user_id"`
    Role     string    `json:"role"`
    JoinedAt time.Time `json:"joined_at"`
}

type CircleRepository interface {
    Create(circle *Circle) error
    GetByID(id string) (*Circle, error)
    GetByUser(userID string) ([]Circle, error)
    Update(circle *Circle) error
    Delete(id string) error
    AddMembers(circleID string, userIDs []string) error
    RemoveMember(circleID string, userID string) error
    IsMember(circleID string, userID string) (bool, error)
    GetMembers(circleID string) ([]CircleMember, error)
    CountByOwner(ownerID string) (int, error)
}

// post.go
type Post struct {
    ID           string    `json:"id"`
    AuthorID     string    `json:"author_id"`
    Content      *string   `json:"content"`
    MediaURL     *string   `json:"media_url"`
    Visibility   string    `json:"visibility"`
    CircleID     *string   `json:"circle_id"`
    LikeCount    int       `json:"like_count"`
    CommentCount int       `json:"comment_count"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

type PostRepository interface {
    Create(post *Post) error
    GetByID(id string) (*Post, error)
    GetByCircle(circleID string, page, limit int) ([]Post, int, error)
    Delete(id string) error
    Like(postID, userID string) error
    Unlike(postID, userID string) error
    IsLiked(postID, userID string) (bool, error)
    GetLikes(postID string, page, limit int) ([]User, int, error)
    CreateComment(comment *PostComment) error
    GetComments(postID string, page, limit int) ([]PostComment, int, error)
    DeleteComment(commentID string) error
}
```

### File Structure Changes

| Action | File Path | Description |
| :--- | :--- | :--- |
| **[NEW]** | `internal/domain/circle.go` | Circle entity, CircleMember, repository interface |
| **[NEW]** | `internal/domain/post.go` | Post entity, PostComment, PostLike, repository interface |
| **[NEW]** | `internal/repository/circle.go` | PostgreSQL Circle data access |
| **[NEW]** | `internal/repository/post.go` | PostgreSQL Post data access |
| **[NEW]** | `internal/usecase/circle.go` | Circle business logic |
| **[NEW]** | `internal/usecase/post.go` | Post business logic (with privacy checks) |
| **[NEW]** | `internal/usecase/circle_test.go` | Circle unit tests |
| **[NEW]** | `internal/usecase/post_test.go` | Post unit tests |
| **[NEW]** | `internal/delivery/http/circle_handler.go` | Circle API handlers + Swagger |
| **[NEW]** | `internal/delivery/http/post_handler.go` | Post API handlers + Swagger |
| **[NEW]** | `pkg/migration/003_circle_only_posts.sql` | Database migration |
| **[MOD]** | `cmd/skiix/main.go` | Wire new dependencies |
| **[MOD]** | `internal/delivery/http/router.go` | Register new routes |

---

## 7. Error Handling

| Scenario | HTTP Code | Error Message |
| :--- | :--- | :--- |
| Invalid / Missing Token | `401` | `"unauthorized"` |
| Expired Token | `401` | `"token expired"` |
| User is not a Circle member | `403` | `"you are not a member of this circle"` |
| Non-owner attempts admin action | `403` | `"only the circle owner can perform this action"` |
| No permission to delete post | `403` | `"you do not have permission to delete this post"` |
| Circle not found | `404` | `"circle not found"` |
| Post not found | `404` | `"post not found"` |
| User not found | `404` | `"one or more users not found"` |
| Circle name empty / too long | `400` | `"circle name is required"` / `"circle name too long (max 50)"` |
| Over 20 circles per user | `400` | `"maximum circles limit reached (max 20)"` |
| Over 150 members per circle | `400` | `"circle member limit reached (max 150)"` |
| Post has no content or media | `400` | `"content or media_url is required"` |
| Circle post missing circle_id | `400` | `"circle_id is required for circle posts"` |
| Comment empty | `400` | `"comment content is required"` |
| Invalid pagination parameters | `400` | `"invalid page or limit parameter"` |
| Database failure | `500` | `"internal server error"` |

---

## 8. Edge Cases

| Edge Case | Expected System Behavior |
| :--- | :--- |
| Circle owner tries to remove themselves | Rejected. Owner must **delete the Circle** instead. Prevents orphaned Circles. |
| Circle is deleted while posts exist | `circle_id` on posts is set to `NULL` (`ON DELETE SET NULL`). Posts become orphaned and inaccessible via Circle feed. Application-layer cleanup job should handle these. |
| User creates a Circle post but is removed from Circle afterwards | The post **remains visible** to current Circle members. Author can still delete it, but cannot view the Circle feed anymore. |
| User tries to like a Circle post they can't access | `403 Forbidden`. Authorization check happens before any write operation. |
| Duplicate like request (idempotent) | Returns success with current `like_count`. `ON CONFLICT DO NOTHING` prevents duplicates. |
| Post with both `content` and `media_url` as null | Rejected: `400` — at least one must be provided. |
| Post with `visibility: "circle"` but no `circle_id` | Rejected: `400` — database `CHECK` constraint also enforces this as a safety net. |
| User is added to a Circle they're already in | `ON CONFLICT DO NOTHING` prevents duplicate membership. Returns success without error. |
| Very large Circle (150 members) | Member list endpoint is paginated. `member_count` is denormalized to avoid `COUNT(*)`. |
| Concurrent like/unlike (race condition) | Transaction-level isolation + `ON CONFLICT` handles this. `like_count` is updated atomically. |

---

## 9. Rate Limits & Constraints

| Resource | Limit | Rationale |
| :--- | :--- | :--- |
| Circles per user | Max **20** | Prevents abuse and keeps UX manageable |
| Members per Circle | Max **150** | Inspired by Dunbar's number. Keeps Circles intimate. |
| Members added per request | Max **50** | Prevents timeout on large batch inserts |
| Post content length | Max **5,000** chars | Encourages concise content |
| Comment content length | Max **2,000** chars | Standard for social platforms |
| Posts per page (pagination) | Max **50** | Prevents excessive data transfer |

---

## 10. Dependencies & Prerequisites

| Dependency | Status | Notes |
| :--- | :--- | :--- |
| `users` table | ✅ Ready | Required for all foreign key references |
| JWT Middleware | ✅ Ready | Authentication for all endpoints |
| Posts module (basic) | 🔴 Must be built | Part of this TRD — `posts` table is defined here |
| Media upload service | ⚠️ Pending | `media_url` assumes external upload; S3/Cloud Storage TBD |
| Push notifications | ⚠️ Pending | For future: notify members when new Circle post is created |

### Implementation Phases

| Phase | Scope | Dependency | Status |
| :--- | :--- | :--- | :--- |
| **Phase 1** | Circle CRUD — Create, list, update, delete Circles. Add/remove members. | Only Users + Auth | ✅ **Ready for Development** |
| **Phase 2** | Posts CRUD — Create public and Circle posts. Delete posts. | Phase 1 complete | ⏳ Sequential |
| **Phase 3** | Interactions — Likes, comments, pagination, feed integration. | Phase 2 complete | ⏳ Sequential |
| **Phase 4** | Feed Integration — Circle posts appear in member timelines. | Feed module design | ⏳ Future |

---

## 11. Acceptance Criteria

### Phase 1 (Circle Management)
- [ ] Users can create Circles with name and optional description.
- [ ] Circle creator is automatically added as owner and first member.
- [ ] Users can list all Circles they belong to (owned + joined).
- [ ] Only Circle owners can update Circle details.
- [ ] Only Circle owners can delete Circles.
- [ ] Only Circle owners can add members (batch, up to 50 per request).
- [ ] Circle owners can remove any member. Members can only remove themselves.
- [ ] Maximum 20 Circles per user. Maximum 150 members per Circle.
- [ ] Non-members receive `403` on all Circle endpoints.
- [ ] Unit test coverage ≥ 90% on Usecase layer.

### Phase 2 (Posts)
- [ ] Users can create public posts (`visibility: "public"`).
- [ ] Users can create Circle-restricted posts (`visibility: "circle"`, `circle_id` required).
- [ ] Circle posts are only creatable by Circle members.
- [ ] Circle feed endpoint returns paginated posts, accessible only to members.
- [ ] Database `CHECK` constraint enforces visibility/circle_id integrity.
- [ ] Post authors and Circle owners can delete posts.

### Phase 3 (Interactions)
- [ ] Users can like/unlike posts. Likes are idempotent.
- [ ] Users can comment on posts. Comments support CRUD.
- [ ] For Circle posts, all interactions require Circle membership.
- [ ] `like_count` and `comment_count` are accurate and updated atomically.
- [ ] Likes and comments lists are paginated.
- [ ] Authorization matrix is fully enforced.

---

## 12. Estimated Timeline

| Task Area | Estimated Time | Primary Owner |
| :--- | :--- | :--- |
| Database migration (5 tables, indexes, constraints) | 2 - 3 hours | Furqan / Luthfi |
| Domain entities & interfaces | 2 - 3 hours | Furqan / Luthfi |
| Circle Repository (PostgreSQL) | 3 - 4 hours | Furqan |
| Post Repository (PostgreSQL) | 3 - 4 hours | Furqan |
| Circle Usecase + Unit Tests | 4 - 5 hours | Furqan |
| Post Usecase + Unit Tests (with privacy logic) | 5 - 6 hours | Furqan |
| Circle HTTP Handlers + Swagger | 3 - 4 hours | Furqan / Luthfi |
| Post HTTP Handlers + Swagger | 3 - 4 hours | Furqan / Luthfi |
| Integration & E2E Testing | 3 - 4 hours | Furqan + Luthfi |
| Code Review & Refactoring | 2 - 3 hours | Team |
| **Total Allocation (All Phases)** | **~6-8 Working Days** | |

---

## 13. Changelog

| Date | Version | Editor | Notes |
| :--- | :--- | :--- | :--- |
| — | 1.0 | Furqan | Initial feature description and high-level behavior |
| April 10, 2026 | 2.0 | Luthfi | Complete technical specification: 5-table DB schema with constraints and indexes, 14 RESTful API endpoints across 3 modules, authorization matrix, 4 system flows, phased implementation plan, comprehensive error handling and edge cases, rate limits, and acceptance criteria. |
