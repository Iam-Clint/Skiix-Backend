# TRD - Focus Mode Button

> **Feature**: Focus Mode Button  
> **Backend Engineer**: Luthfi Alfari & Upi  
> **Status**: In Progress  
> **Last Updated**: April 11, 2026  
> **Version**: 2.1

---

## 1. Overview

### Goal

Allows users to enter a **distraction-free mode** within the Skiix application. When Focus Mode is active, the feed content is filtered based on user-selected categories, ensuring they only see content that is relevant and aligned with their focus goals.

### User Behavior

- Users can **toggle** Focus Mode ON/OFF at any time.
- When active, the feed displays **restricted content** based on the chosen categories.
- Notifications and non-essential UI elements are hidden (e.g., the Reel tab icon).
- Content restriction is based on **specific categories** that can be customized.
- Users can **handpick** which content categories they want to filter out.

---

## 2. Database Schema

### Table: `focus_mode_settings`

Stores the focus mode configuration for each user. A user can only have one settings record.

| Column | Type | Constraint | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `id` | UUID | PRIMARY KEY | `gen_random_uuid()` | Unique identifier |
| `user_id` | UUID | NOT NULL, UNIQUE, FK → `users(id)` | — | Relation to the user |
| `is_enabled` | BOOLEAN | NOT NULL | `FALSE` | Focus mode status |
| `created_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | Last update timestamp |

**Constraints:**
- `UNIQUE (user_id)` — Ensures a 1:1 relationship per user.
- `FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`

---

### Table: `focus_mode_blocked_categories`

Stores the list of content categories blocked per user. Separated from the settings table to achieve database normalization, allowing N categories per user without using JSON or array columns.

| Column | Type | Constraint | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `id` | UUID | PRIMARY KEY | `gen_random_uuid()` | Unique identifier |
| `user_id` | UUID | NOT NULL, FK → `users(id)` | — | Relation to the user |
| `category_name` | VARCHAR(100) | NOT NULL | — | Name of the blocked category |
| `created_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | Creation timestamp |

**Constraints:**
- `UNIQUE (user_id, category_name)` — Prevents users from blocking the same category multiple times.
- `FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`

---

### Indexes

| Index Name | Table | Column(s) | Purpose |
| :--- | :--- | :--- | :--- |
| `idx_focus_settings_user` | `focus_mode_settings` | `user_id` | Fast lookup of settings by user |
| `idx_focus_blocked_user` | `focus_mode_blocked_categories` | `user_id` | Fast lookup of blocked categories by user |

---

### SQL Migration Script

```sql
-- Migration: 002_focus_mode.sql
-- Description: Create focus mode tables
-- Author: Luthfi Alfari

CREATE TABLE IF NOT EXISTS focus_mode_settings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_enabled  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_focus_mode_user UNIQUE (user_id)
);

CREATE TABLE IF NOT EXISTS focus_mode_blocked_categories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_name   VARCHAR(100) NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_user_category UNIQUE (user_id, category_name)
);

CREATE INDEX IF NOT EXISTS idx_focus_settings_user
    ON focus_mode_settings(user_id);

CREATE INDEX IF NOT EXISTS idx_focus_blocked_user
    ON focus_mode_blocked_categories(user_id);
```

### Entity Relationship Diagram

The following diagram shows the relationships between the focus mode tables and the existing `users` table, as well as the **future dependency** on the `posts` table for feed filtering.

```
┌─────────────────────┐
│       users          │
│─────────────────────│
│ id (PK)             │
│ email               │
│ password            │
│ provider            │
│ created_at          │
│ updated_at          │
└──────┬──────┬───────┘
       │      │
       │      │  1:1
       │      ▼
       │  ┌─────────────────────────┐
       │  │  focus_mode_settings     │
       │  │─────────────────────────│
       │  │ id (PK)                 │
       │  │ user_id (FK, UNIQUE)    │──→ users(id) ON DELETE CASCADE
       │  │ is_enabled              │
       │  │ created_at              │
       │  │ updated_at              │
       │  └─────────────────────────┘
       │
       │  1:N
       ▼
  ┌──────────────────────────────────┐
  │  focus_mode_blocked_categories    │
  │──────────────────────────────────│
  │ id (PK)                          │
  │ user_id (FK)                     │──→ users(id) ON DELETE CASCADE
  │ category_name                    │
  │ created_at                       │
  │ UNIQUE(user_id, category_name)   │
  └──────────────────────────────────┘

  ── Future Dependency (Phase 2) ─────────────────

  ┌─────────────────────┐       ┌──────────────────┐
  │      posts (TBD)     │       │ post_categories   │
  │─────────────────────│  1:N  │  (TBD)            │
  │ id (PK)             │──────→│ post_id (FK)      │
  │ user_id (FK)        │       │ category_name     │
  │ content             │       └──────────────────┘
  │ created_at          │              │
  └─────────────────────┘              │ matched against
                                       ▼
                          focus_mode_blocked_categories
                          (category_name)
```

**Relationship Summary:**

| From | To | Relationship | Constraint |
| :--- | :--- | :--- | :--- |
| `users` | `focus_mode_settings` | **1:1** | `UNIQUE(user_id)` ensures one setting per user |
| `users` | `focus_mode_blocked_categories` | **1:N** | A user can block up to 20 categories |
| `focus_mode_blocked_categories` | `posts` (Phase 2) | **Filter join** | `WHERE category_name NOT IN (blocked_list)` |

---

### Design Decisions

| Decision | Rationale |
| :--- | :--- |
| **Separated from `users` table** | Promotes Single Responsibility — settings can evolve without altering the core user identity table. |
| **Separate table for categories** | Proper normalization. Avoids complex querying and indexing issues associated with JSON/array columns. |
| **UUID for Primary Keys** | Maintains consistency with the existing `users` table design in the codebase. |
| **`ON DELETE CASCADE`** | Guarantees that if a user is deleted, all their focus settings are automatically cleaned up. |
| **`UPSERT` pattern** | Prevents race conditions during concurrent toggle attempts (last-write-wins). |

---

## 3. API Design

### Base URL

```text
/api/v1/focus-mode
```

*Note: All endpoints require **JWT Authentication** via headers.*
```text
Authorization: Bearer <JWT_TOKEN>
```

---

### Endpoint 1: Get Focus Mode Status

`GET /api/v1/focus-mode`

**Description**: Retrieves the current focus mode status and the list of blocked categories for the authenticated user.

**Response 200 (Success):**
```json
{
    "is_enabled": true,
    "blocked_categories": ["entertainment", "news", "memes"]
}
```

**Response 200 (User has never set it up - defaults):**
```json
{
    "is_enabled": false,
    "blocked_categories": []
}
```

**Response 401 (Unauthorized):**
```json
{
    "error": "unauthorized"
}
```

---

### Endpoint 2: Toggle Focus Mode

`PUT /api/v1/focus-mode`

**Description**: Enables or disables focus mode.

**Request Body:**
```json
{
    "is_enabled": true
}
```

**Response 200 (Success):**
```json
{
    "message": "focus mode updated",
    "is_enabled": true
}
```

**Response 400 (Invalid Request):**
```json
{
    "error": "is_enabled must be a boolean"
}
```

---

### Endpoint 3: Set Blocked Categories

`PUT /api/v1/focus-mode/categories`

**Description**: Configures the list of content categories to be blocked when focus mode is active. This action **replaces** the entire list (it does not append).

**Request Body:**
```json
{
    "categories": ["entertainment", "news", "memes"]
}
```

**Validation Rules:**
- `categories` array must only contain strings.
- Each category name: min 1 character, max 100 characters.
- Maximum limit of 20 categories per user.
- Empty array `[]` will remove all blocked categories (reset).
- Duplicate categories in the payload must be auto-deduplicated.

**Response 200 (Success):**
```json
{
    "message": "blocked categories updated",
    "blocked_categories": ["entertainment", "news", "memes"]
}
```

**Response 400 (Validation Error Examples):**
```json
{
    "error": "too many categories (max 20)"
}
```
```json
{
    "error": "category name too long (max 100 characters)"
}
```

---

## 4. System Flow

### Flow 1: Toggle Focus Mode

1. **User (Mobile App)** sends `PUT /api/v1/focus-mode { "is_enabled": true }`
2. **JWT Middleware** extracts the `user_id` from the token.
3. **Handler** parses and validates the JSON request body.
4. **Usecase** handles the business logic validation.
5. **Repository** executes an UPSERT on `focus_mode_settings` (`INSERT ... ON CONFLICT (user_id) DO UPDATE`).
6. **PostgreSQL** confirms atomic row insertion/update.
7. **API Response** returns HTTP 200.

### Flow 2: Set Blocked Categories

1. **User (Mobile App)** sends `PUT /api/v1/focus-mode/categories { "categories": ["news", "memes"] }`
2. **Usecase** validates the array (max 20 items, max 100 chars each) and removes any duplicates.
3. **Repository** executes a database transaction:
   - `DELETE FROM focus_mode_blocked_categories WHERE user_id = $1`
   - `INSERT INTO focus_mode_blocked_categories (user_id, category_name) VALUES ...`
4. **PostgreSQL** ensures categories are fully replaced atomically.
5. **API Response** returns HTTP 200.

### Flow 3: Feed Filtering (Phase 2 Component)

1. **User (Mobile App)** requests feed via `GET /api/v1/feed`.
2. **Feed Usecase** queries **FocusModeRepository** to check if `is_enabled == true`.
3. If true, fetches the user's blocked categories.
4. Appends a category filter to the Feed SQL Database Query (e.g., `WHERE category NOT IN (blocked_list)`).
5. Returns the filtered posts to the user.

### Feed Integration Strategy (Phase 2 — Category Tagging)

> **Amanda's Question**: "How does each post get tagged with a category so the system can identify which feed items to block?"

**Answer**: Every post in the system will be associated with one or more content categories. There are two possible approaches:

| Approach | Implementation | Pros | Cons |
| :--- | :--- | :--- | :--- |
| **Option A: Column on `posts`** | `posts.category VARCHAR(100)` | Simple, fast queries | Limited to 1 category per post |
| **Option B: Junction table** | `post_categories(post_id, category_name)` | Multi-tag support, flexible | Slightly more complex queries |

**Recommended: Option B** (junction table) — because real-world content often belongs to multiple categories (e.g., a meme about sports news = "memes" + "news" + "sports").

**Feed Filtering SQL (Phase 2 Example):**

```sql
-- Get feed for user, excluding posts that match ANY blocked category
SELECT p.*
FROM posts p
WHERE p.id NOT IN (
    SELECT pc.post_id
    FROM post_categories pc
    INNER JOIN focus_mode_blocked_categories fmbc
        ON pc.category_name = fmbc.category_name
    WHERE fmbc.user_id = $1
)
AND (SELECT is_enabled FROM focus_mode_settings WHERE user_id = $1) = TRUE
ORDER BY p.created_at DESC
LIMIT 20 OFFSET $2;
```

**Key behaviors:**
- If `is_enabled = FALSE` → the `WHERE` condition skips the subquery entirely, returning the full feed.
- If a post has **any** category matching a blocked category, the entire post is excluded.
- Posts without categories are **never filtered** (they always appear in the feed).

---

## 5. Architecture (Clean Architecture Layers)

### Domain Layer — `internal/domain/focus_mode.go`

```go
// Entities
type FocusModeSetting struct {
    ID        string
    UserID    string
    IsEnabled bool
    CreatedAt time.Time
    UpdatedAt time.Time
}

type FocusBlockedCategory struct {
    ID           string
    UserID       string
    CategoryName string
    CreatedAt    time.Time
}

// Response DTO
type FocusModeStatus struct {
    IsEnabled         bool     `json:"is_enabled"`
    BlockedCategories []string `json:"blocked_categories"`
}

// Repository Interface
type FocusModeRepository interface {
    GetSettings(userID string) (*FocusModeSetting, error)
    UpsertSettings(userID string, isEnabled bool) error
    GetBlockedCategories(userID string) ([]string, error)
    SetBlockedCategories(userID string, categories []string) error
}

// Usecase Interface
type FocusModeUsecase interface {
    GetStatus(userID string) (*FocusModeStatus, error)
    Toggle(userID string, enable bool) error
    SetBlockedCategories(userID string, categories []string) error
}
```

### File Structure Changes

| Action | File Path | Description |
| :--- | :--- | :--- |
| **[NEW]** | `internal/domain/focus_mode.go` | Entities & interfaces |
| **[NEW]** | `internal/repository/focus_mode.go` | PostgreSQL data access implementation |
| **[NEW]** | `internal/usecase/focus_mode.go` | Business logic implementation |
| **[NEW]** | `internal/usecase/focus_mode_test.go` | Comprehensive Unit tests |
| **[NEW]** | `internal/delivery/http/focus_mode_handler.go` | API handlers + Swagger annotations |
| **[NEW]** | `pkg/migration/002_focus_mode.sql` | Database migration scripts |
| **[MOD]** | `cmd/skiix/main.go` | Dependency injection wiring |
| **[MOD]** | `internal/delivery/http/router.go` | Register new endpoints |

---

## 6. Error Handling

| Scenario | HTTP Code | Error Message |
| :--- | :--- | :--- |
| Invalid / Missing Token | `401` | `"unauthorized"` |
| Expired Token | `401` | `"token expired"` |
| Invalid JSON body | `400` | `"invalid request body"` |
| `is_enabled` type mismatch | `400` | `"is_enabled must be a boolean"` |
| `categories` type mismatch | `400` | `"categories must be an array of strings"` |
| Category name exceeds bounds | `400` | `"category name too long (max 100 characters)"` |
| Over 20 categories | `400` | `"too many categories (max 20)"` |
| Empty category name sting | `400` | `"category name cannot be empty"` |
| Database transaction failure | `500` | `"internal server error"` |

---

## 7. Edge Cases

| Edge Case | Expected System Behavior |
| :--- | :--- |
| User has never updated Focus settings | Return the predefined defaults: `{ is_enabled: false, blocked_categories: [] }`. |
| Toggle ON without categories | Focus mode becomes active, but no content is filtered (feed displays as normal). |
| Categories assigned but toggled OFF | Categories are saved reliably but have zero impact on feed querying until activated. |
| Duplicate categories in payload | The UseCase automatically removes duplicates; saves only unique entries. |
| Empty categories array `[]` sent | Safely deletes all existing blocked categories (effectively acts as a reset). |
| User account is deleted | PostgreSQL `CASCADE` immediately wipes all related focus configurations and categories. |
| Concurrent toggle requests (Race condition) | Database `ON CONFLICT DO UPDATE` reliably ensures the very last write operation wins. |
| Case sensitivity on categories | Saved exactly as-is (case-sensitive). "News" and "news" are technically distinct. |

---

## 8. Dependencies & Prerequisites

| Dependency | Status | Notes |
| :--- | :--- | :--- |
| `users` table | ✅ Ready | Exists in DB (Required for Foreign Keys) |
| JWT Middleware | ✅ Ready | Exists mapping for extracting `user_id` |
| Feed/Posts Module | ⚠️ Pending | Core feed architecture must be established to apply the filtering queries |

### Implementation Phases

| Phase | Scope | Blocker/Dependency | Status |
| :--- | :--- | :--- | :--- |
| **Phase 1** | **Setting CRUD operations**: API routes for toggling focus mode and managing block lists. | None (Only needs Users and Auth) | ✅ **Ready for Development** |
| **Phase 2** | **Feed Filtering logic**: Integrating Focus Mode DB checks into the Feed timeline queries. | Waiting on the Feed/Posts module creation. | ⏳ Pending Dependencies |

---

## 9. Acceptance Criteria

### Phase 1 (CRUD Settings)
- [ ] Users can successfully toggle focus mode ON/OFF via `PUT /api/v1/focus-mode`.
- [ ] Users can assign blocked categories via `PUT /api/v1/focus-mode/categories`.
- [ ] `GET /api/v1/focus-mode` returns accurate configurations.
- [ ] First-time requests properly default to off / empty array without fatal exceptions.
- [ ] Edge validations (max 20 tags, max 100 chars, empty arrays) behave correctly.
- [ ] All three endpoints are guarded by the JWT authentication layer.
- [ ] Business logic (UseCase) boasts ≥ 90% unit test coverage.
- [ ] Fully documented via Swagger UI.
- [ ] Migration file created and validated locally.

### Phase 2 (Feed Integration)
- [ ] Feed query pipeline successfully reads blocked categories.
- [ ] Standard feed is completely unaffected when Focus Mode is OFF.
- [ ] Database query latency penalty introduced by filtering remains under 50ms.

---

## 10. Estimated Timeline

| Task Area | Estimated Time | Primary Owner |
| :--- | :--- | :--- |
| Database migration & Domain entities | 1 - 2 hours | Luthfi |
| Repository implementation (PostgreSQL) | 2 - 3 hours | Luthfi |
| Usecase logic & Comprehensive Unit Tests | 3 - 4 hours | Luthfi |
| HTTP Handlers & Swagger integration | 2 - 3 hours | Luthfi |
| E2E/Integration testing flows | 1 - 2 hours | Luthfi |
| Code review & refactoring | 1 - 2 hours | Luthfi + Reviewer |
| **Total Allocation (Phase 1)** | **~2-3 Days** | |

---

## 11. Changelog

| Date | Version | Editor | Notes |
| :--- | :--- | :--- | :--- |
| — | 1.0 | Amanda | Initial framework planning |
| April 10, 2026 | 2.0 | Luthfi | Expanded technical specifications covering DB schemas, Restful endpoints, sequence flows, bounds limits, and Phase mapping. |
| April 11, 2026 | 2.1 | Luthfi | Added ER diagram for table relationships, Feed Integration Strategy explaining category tagging on posts, and filtering SQL example. Based on Amanda's feedback. |
