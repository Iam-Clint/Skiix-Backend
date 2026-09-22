# TRD - Daily Feed Limit

> **Feature**: Daily Feed Limit  
> **Backend Engineer**: Furqan (supported by Luthfi Alfari)  
> **Status**: In Progress  
> **Last Updated**: April 10, 2026  
> **Version**: 2.0

---

## 1. Overview

### Goal

Enable users to set **daily consumption limits** on their feed to promote healthy app usage habits. This feature functions as an in-app **screen time controller**, allowing users to define boundaries on how many posts they view or how long they scroll per day. When a limit is reached, the user is notified and the feed is gated.

### User Behavior

- Users can set a **maximum number of posts** they want to view per day.
- Users can set a **maximum scroll duration** (in minutes) per day.
- Either limit, or both, can be configured independently.
- Limits **reset automatically** at midnight (based on user's timezone or UTC).
- Users receive a **notification/alert** when they reach their daily limit.
- After reaching the limit, the feed displays a **"limit reached" screen** instead of new posts.
- Users can **override** the limit (snooze for 15 minutes) a limited number of times per day.
- Users can **disable** the feature entirely at any time.
- The feature is **opt-in** — disabled by default for all users.

---

## 2. Database Schema

### Entity Relationship Overview

```text
┌──────────┐       ┌──────────────────────┐
│  users   │──1:1──│ feed_limit_settings   │
│          │       └──────────────────────┘
│          │
│          │──1:N──┌──────────────────────┐
│          │       │ feed_usage_daily      │
│          │       └──────────────────────┘
│          │
│          │──1:N──┌──────────────────────┐
│          │       │ feed_limit_overrides  │
└──────────┘       └──────────────────────┘
```

---

### Table 1: `feed_limit_settings`

Stores each user's daily feed limit preferences. One record per user.

| Column | Type | Constraint | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `id` | UUID | PRIMARY KEY | `gen_random_uuid()` | Unique identifier |
| `user_id` | UUID | NOT NULL, UNIQUE, FK → `users(id)` | — | Relation to the user |
| `is_enabled` | BOOLEAN | NOT NULL | `FALSE` | Whether the feed limit is active |
| `max_posts_per_day` | INTEGER | — | `NULL` | Maximum posts viewable per day (NULL = unlimited) |
| `max_scroll_minutes` | INTEGER | — | `NULL` | Maximum scroll time in minutes per day (NULL = unlimited) |
| `max_overrides_per_day` | INTEGER | NOT NULL | `3` | How many times user can snooze the limit |
| `override_duration_minutes` | INTEGER | NOT NULL | `15` | Duration of each override in minutes |
| `created_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | Last modification timestamp |

**Constraints:**
- `UNIQUE (user_id)` — One settings record per user.
- `FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`
- `CHECK (max_posts_per_day IS NULL OR max_posts_per_day > 0)` — Must be positive if set.
- `CHECK (max_scroll_minutes IS NULL OR max_scroll_minutes > 0)` — Must be positive if set.
- `CHECK (max_overrides_per_day >= 0 AND max_overrides_per_day <= 10)` — Reasonable override limit.

---

### Table 2: `feed_usage_daily`

Tracks daily feed consumption per user. A new record is created for each user for each day they use the feed.

| Column | Type | Constraint | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `id` | UUID | PRIMARY KEY | `gen_random_uuid()` | Unique identifier |
| `user_id` | UUID | NOT NULL, FK → `users(id)` | — | Relation to the user |
| `usage_date` | DATE | NOT NULL | `CURRENT_DATE` | The calendar date of this usage record |
| `posts_viewed` | INTEGER | NOT NULL | `0` | Number of posts viewed today |
| `scroll_minutes` | INTEGER | NOT NULL | `0` | Total scroll time in minutes today |
| `limit_reached_at` | TIMESTAMPTZ | — | `NULL` | Timestamp when the limit was first hit |
| `is_limit_reached` | BOOLEAN | NOT NULL | `FALSE` | Quick flag for limit status |
| `created_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | Record creation timestamp |
| `updated_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | Last update timestamp |

**Constraints:**
- `UNIQUE (user_id, usage_date)` — One record per user per day.
- `FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`
- `CHECK (posts_viewed >= 0)` — Cannot be negative.
- `CHECK (scroll_minutes >= 0)` — Cannot be negative.

---

### Table 3: `feed_limit_overrides`

Logs each time a user overrides (snoozes) their daily limit. Used to enforce the maximum override count.

| Column | Type | Constraint | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `id` | UUID | PRIMARY KEY | `gen_random_uuid()` | Unique identifier |
| `user_id` | UUID | NOT NULL, FK → `users(id)` | — | Relation to the user |
| `override_date` | DATE | NOT NULL | `CURRENT_DATE` | The date of the override |
| `activated_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | When the override started |
| `expires_at` | TIMESTAMPTZ | NOT NULL | — | When the override expires |
| `created_at` | TIMESTAMPTZ | NOT NULL | `NOW()` | Record creation timestamp |

**Constraints:**
- `FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`

---

### Indexes

| Index Name | Table | Column(s) | Purpose |
| :--- | :--- | :--- | :--- |
| `idx_feed_limit_user` | `feed_limit_settings` | `user_id` | Fast settings lookup |
| `idx_feed_usage_user_date` | `feed_usage_daily` | `user_id, usage_date` | Fast daily usage lookup |
| `idx_feed_override_user_date` | `feed_limit_overrides` | `user_id, override_date` | Count today's overrides |

---

### SQL Migration Script

```sql
-- Migration: 004_daily_feed_limit.sql
-- Description: Create Daily Feed Limit tables
-- Author: Furqan / Luthfi Alfari

-- ============================
-- 1. Feed Limit Settings
-- ============================
CREATE TABLE IF NOT EXISTS feed_limit_settings (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_enabled                  BOOLEAN NOT NULL DEFAULT FALSE,
    max_posts_per_day           INTEGER,
    max_scroll_minutes          INTEGER,
    max_overrides_per_day       INTEGER NOT NULL DEFAULT 3,
    override_duration_minutes   INTEGER NOT NULL DEFAULT 15,
    created_at                  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_feed_limit_user UNIQUE (user_id),
    CONSTRAINT chk_max_posts CHECK (max_posts_per_day IS NULL OR max_posts_per_day > 0),
    CONSTRAINT chk_max_scroll CHECK (max_scroll_minutes IS NULL OR max_scroll_minutes > 0),
    CONSTRAINT chk_max_overrides CHECK (max_overrides_per_day >= 0 AND max_overrides_per_day <= 10)
);

-- ============================
-- 2. Feed Usage Daily
-- ============================
CREATE TABLE IF NOT EXISTS feed_usage_daily (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    usage_date          DATE NOT NULL DEFAULT CURRENT_DATE,
    posts_viewed        INTEGER NOT NULL DEFAULT 0,
    scroll_minutes      INTEGER NOT NULL DEFAULT 0,
    limit_reached_at    TIMESTAMP WITH TIME ZONE,
    is_limit_reached    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_user_usage_date UNIQUE (user_id, usage_date),
    CONSTRAINT chk_posts_viewed CHECK (posts_viewed >= 0),
    CONSTRAINT chk_scroll_minutes CHECK (scroll_minutes >= 0)
);

-- ============================
-- 3. Feed Limit Overrides
-- ============================
CREATE TABLE IF NOT EXISTS feed_limit_overrides (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    override_date   DATE NOT NULL DEFAULT CURRENT_DATE,
    activated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ============================
-- 4. Indexes
-- ============================
CREATE INDEX IF NOT EXISTS idx_feed_limit_user ON feed_limit_settings(user_id);
CREATE INDEX IF NOT EXISTS idx_feed_usage_user_date ON feed_usage_daily(user_id, usage_date);
CREATE INDEX IF NOT EXISTS idx_feed_override_user_date ON feed_limit_overrides(user_id, override_date);
```

### Design Decisions

| Decision | Rationale |
| :--- | :--- |
| **Separate `feed_usage_daily` table** | Each row = one day's usage for one user. Clean time-series data, easy to query historical usage trends, and avoids resetting columns at midnight. |
| **`feed_limit_overrides` as audit log** | Tracks every snooze event individually. Enables analytics (how often do users override?) and prevents manipulation. |
| **`NULL` for unlimited** | `max_posts_per_day = NULL` means no post limit. This is more semantic than using `-1` or `0` as magic values. |
| **`is_limit_reached` boolean flag** | Denormalized quick-check flag. Avoids recalculating limit status on every feed request. |
| **Date-based partitioning ready** | `usage_date` and `override_date` columns enable future table partitioning by date for performance at scale. |
| **Override system (snooze)** | Rather than a hard block, users get flexibility to extend. Respects user autonomy while still promoting awareness. Defaults: 3 overrides × 15 minutes each. |

---

## 3. API Design

### Base URL

```text
/api/v1/feed-limit
```

*Note: All endpoints require **JWT Authentication** via header:*
```text
Authorization: Bearer <JWT_TOKEN>
```

---

### Endpoint 1: Get Feed Limit Settings

`GET /api/v1/feed-limit`

**Description**: Retrieves the current feed limit configuration for the authenticated user.

**Response 200 (User has configured limits):**
```json
{
    "is_enabled": true,
    "max_posts_per_day": 50,
    "max_scroll_minutes": 30,
    "max_overrides_per_day": 3,
    "override_duration_minutes": 15
}
```

**Response 200 (User has never configured — defaults):**
```json
{
    "is_enabled": false,
    "max_posts_per_day": null,
    "max_scroll_minutes": null,
    "max_overrides_per_day": 3,
    "override_duration_minutes": 15
}
```

---

### Endpoint 2: Update Feed Limit Settings

`PUT /api/v1/feed-limit`

**Description**: Creates or updates the user's daily feed limit settings.

**Request Body:**
```json
{
    "is_enabled": true,
    "max_posts_per_day": 50,
    "max_scroll_minutes": 30,
    "max_overrides_per_day": 3,
    "override_duration_minutes": 15
}
```

| Field | Type | Required | Validation |
| :--- | :--- | :--- | :--- |
| `is_enabled` | boolean | ✅ Yes | — |
| `max_posts_per_day` | integer or null | ❌ No | Must be > 0 if provided. Max: 1000. |
| `max_scroll_minutes` | integer or null | ❌ No | Must be > 0 if provided. Max: 1440 (24h). |
| `max_overrides_per_day` | integer | ❌ No | 0-10. Default: 3. |
| `override_duration_minutes` | integer | ❌ No | 5-60. Default: 15. |

**Validation Rule:** When `is_enabled` is `true`, at least one of `max_posts_per_day` or `max_scroll_minutes` must be provided (non-null). Enabling limits without specifying any limit value is rejected.

**Response 200 (Success):**
```json
{
    "message": "feed limit settings updated",
    "is_enabled": true,
    "max_posts_per_day": 50,
    "max_scroll_minutes": 30
}
```

**Response 400:** `"at least one limit (posts or scroll time) must be set when enabling"`  
**Response 400:** `"max_posts_per_day must be a positive number (max 1000)"`  
**Response 400:** `"max_scroll_minutes must be between 1 and 1440"`

---

### Endpoint 3: Get Today's Usage

`GET /api/v1/feed-limit/usage`

**Description**: Retrieves the user's feed consumption stats for today, compared against their configured limits.

**Response 200:**
```json
{
    "date": "2026-04-10",
    "posts_viewed": 35,
    "scroll_minutes": 22,
    "limits": {
        "max_posts_per_day": 50,
        "max_scroll_minutes": 30,
        "is_enabled": true
    },
    "progress": {
        "posts_percentage": 70,
        "scroll_percentage": 73
    },
    "is_limit_reached": false,
    "limit_reached_at": null,
    "override": {
        "is_active": false,
        "expires_at": null,
        "overrides_used_today": 1,
        "overrides_remaining": 2
    }
}
```

---

### Endpoint 4: Track Feed Consumption

`POST /api/v1/feed-limit/track`

**Description**: Called by the frontend to report feed consumption events. The frontend sends periodic heartbeats (e.g., every 30 seconds while scrolling, or on each post view). This endpoint also returns the current limit status so the frontend knows when to show the limit screen.

**Request Body:**
```json
{
    "event_type": "post_view",
    "count": 5
}
```

| Field | Type | Required | Validation |
| :--- | :--- | :--- | :--- |
| `event_type` | string | ✅ Yes | `"post_view"` or `"scroll_time"` |
| `count` | integer | ✅ Yes | For `post_view`: number of posts (1-50). For `scroll_time`: minutes (1-30). |

**Response 200 (Under limit):**
```json
{
    "is_limit_reached": false,
    "posts_viewed": 40,
    "scroll_minutes": 22,
    "posts_remaining": 10,
    "scroll_minutes_remaining": 8
}
```

**Response 200 (Limit reached):**
```json
{
    "is_limit_reached": true,
    "posts_viewed": 52,
    "scroll_minutes": 30,
    "posts_remaining": 0,
    "scroll_minutes_remaining": 0,
    "can_override": true,
    "overrides_remaining": 2,
    "message": "You've reached your daily feed limit. Take a break! 🧘"
}
```

**Response 400:** `"invalid event_type (must be 'post_view' or 'scroll_time')"`  
**Response 400:** `"count must be a positive number"`

---

### Endpoint 5: Activate Override (Snooze)

`POST /api/v1/feed-limit/override`

**Description**: Activates a temporary override (snooze) when the user has reached their limit. Extends feed access for the configured `override_duration_minutes`. Limited to `max_overrides_per_day` per day.

**Response 200 (Override activated):**
```json
{
    "message": "override activated",
    "expires_at": "2026-04-10T15:15:00Z",
    "overrides_remaining": 1
}
```

**Response 400:** `"no overrides remaining today"`  
**Response 400:** `"limit has not been reached yet"`  
**Response 400:** `"an override is already active"`

---

### Endpoint 6: Disable Feed Limit

`DELETE /api/v1/feed-limit`

**Description**: Disables the feed limit feature entirely. Settings are preserved but `is_enabled` is set to `false`.

**Response 200:**
```json
{
    "message": "feed limit disabled",
    "is_enabled": false
}
```

---

### Endpoint 7: Get Usage History

`GET /api/v1/feed-limit/history?days=7`

**Description**: Returns feed consumption history for the past N days. Useful for the frontend to display usage trends/charts.

**Query Parameters:**

| Parameter | Type | Default | Validation |
| :--- | :--- | :--- | :--- |
| `days` | integer | `7` | 1-30 |

**Response 200:**
```json
{
    "history": [
        {
            "date": "2026-04-10",
            "posts_viewed": 35,
            "scroll_minutes": 22,
            "limit_reached": false
        },
        {
            "date": "2026-04-09",
            "posts_viewed": 55,
            "scroll_minutes": 31,
            "limit_reached": true
        },
        {
            "date": "2026-04-08",
            "posts_viewed": 12,
            "scroll_minutes": 8,
            "limit_reached": false
        }
    ],
    "averages": {
        "avg_posts_per_day": 34,
        "avg_scroll_minutes": 20,
        "days_limit_reached": 1,
        "total_days": 3
    }
}
```

---

## 4. System Flow

### Flow 1: Configure Feed Limit

1. **User** sends `PUT /api/v1/feed-limit { "is_enabled": true, "max_posts_per_day": 50 }`.
2. **Handler** validates input: `is_enabled` is boolean, at least one limit is set.
3. **Usecase** checks: are limits within valid ranges?
4. **Repository** executes UPSERT on `feed_limit_settings`.
5. **Response**: Returns updated settings.

### Flow 2: Track Feed Consumption (Core Loop)

1. **Frontend** periodically sends `POST /api/v1/feed-limit/track { "event_type": "post_view", "count": 1 }` as user scrolls.
2. **Usecase** checks if feed limit is enabled for this user.
3. If not enabled → increment counters silently, return `is_limit_reached: false`.
4. If enabled:
   - **Repository** executes inside a **transaction**:
     - `INSERT INTO feed_usage_daily (user_id, usage_date, posts_viewed) VALUES ($1, CURRENT_DATE, $3) ON CONFLICT (user_id, usage_date) DO UPDATE SET posts_viewed = feed_usage_daily.posts_viewed + $3`
   - Fetch updated usage + user's limits.
   - Compare: `posts_viewed >= max_posts_per_day` OR `scroll_minutes >= max_scroll_minutes`?
   - If limit reached:
     - Set `is_limit_reached = true`, `limit_reached_at = NOW()` on `feed_usage_daily`.
     - Check if active override exists (`expires_at > NOW()`).
     - If override active → allow continued access.
     - If no override → return `is_limit_reached: true` with override options.
5. **Frontend** receives response and displays limit screen or continues feed.

### Flow 3: Override (Snooze) the Limit

1. **User** taps "Extend 15 minutes" on the limit screen.
2. **Frontend** sends `POST /api/v1/feed-limit/override`.
3. **Usecase** validates:
   - Is limit actually reached? If not → reject.
   - Is there already an active override? If yes → reject.
   - Count today's overrides: `SELECT COUNT(*) FROM feed_limit_overrides WHERE user_id = $1 AND override_date = CURRENT_DATE`. If count >= `max_overrides_per_day` → reject.
4. **Repository** inserts override record with `expires_at = NOW() + override_duration_minutes`.
5. **Response**: Returns expiration time and remaining overrides.
6. **Frontend** starts a countdown timer and resumes feed.
7. Subsequent `track` requests check if override has expired. If expired → gate the feed again.

### Flow 4: Daily Reset (Automatic)

```text
Option A: Application-level (recommended for MVP)
  - Every track request checks usage_date.
  - If usage_date < CURRENT_DATE, a new row is created automatically.
  - Old data is naturally partitioned by date.

Option B: Database-level (for production scale)
  - PostgreSQL cron job (pg_cron) or external scheduler.
  - No rows need resetting — new date = new row.
  - Historical data preserved for analytics.
```

The design uses **Option A by default** — no reset job needed. Each day's usage is a separate row, so "resetting" is simply creating a new row for the new date.

---

## 5. Architecture (Clean Architecture Layers)

### Domain Layer — `internal/domain/feed_limit.go`

```go
type FeedLimitSetting struct {
    ID                      string    `json:"id"`
    UserID                  string    `json:"user_id"`
    IsEnabled               bool      `json:"is_enabled"`
    MaxPostsPerDay          *int      `json:"max_posts_per_day"`
    MaxScrollMinutes        *int      `json:"max_scroll_minutes"`
    MaxOverridesPerDay      int       `json:"max_overrides_per_day"`
    OverrideDurationMinutes int       `json:"override_duration_minutes"`
    CreatedAt               time.Time `json:"created_at"`
    UpdatedAt               time.Time `json:"updated_at"`
}

type FeedUsageDaily struct {
    ID             string     `json:"id"`
    UserID         string     `json:"user_id"`
    UsageDate      time.Time  `json:"usage_date"`
    PostsViewed    int        `json:"posts_viewed"`
    ScrollMinutes  int        `json:"scroll_minutes"`
    LimitReachedAt *time.Time `json:"limit_reached_at"`
    IsLimitReached bool       `json:"is_limit_reached"`
}

type FeedLimitOverride struct {
    ID           string    `json:"id"`
    UserID       string    `json:"user_id"`
    OverrideDate time.Time `json:"override_date"`
    ActivatedAt  time.Time `json:"activated_at"`
    ExpiresAt    time.Time `json:"expires_at"`
}

type FeedLimitRepository interface {
    GetSettings(userID string) (*FeedLimitSetting, error)
    UpsertSettings(setting *FeedLimitSetting) error
    DisableSettings(userID string) error
    GetTodayUsage(userID string) (*FeedUsageDaily, error)
    IncrementUsage(userID string, postsViewed int, scrollMinutes int) (*FeedUsageDaily, error)
    MarkLimitReached(userID string) error
    GetUsageHistory(userID string, days int) ([]FeedUsageDaily, error)
    CreateOverride(userID string, expiresAt time.Time) (*FeedLimitOverride, error)
    GetActiveOverride(userID string) (*FeedLimitOverride, error)
    CountTodayOverrides(userID string) (int, error)
}

type FeedLimitUsecase interface {
    GetSettings(userID string) (*FeedLimitSetting, error)
    UpdateSettings(userID string, setting *FeedLimitSetting) error
    DisableLimit(userID string) error
    GetTodayUsage(userID string) (*FeedUsageResponse, error)
    TrackConsumption(userID string, eventType string, count int) (*TrackResponse, error)
    ActivateOverride(userID string) (*FeedLimitOverride, error)
    GetUsageHistory(userID string, days int) (*UsageHistoryResponse, error)
}
```

### File Structure Changes

| Action | File Path | Description |
| :--- | :--- | :--- |
| **[NEW]** | `internal/domain/feed_limit.go` | Entities, DTOs & interfaces |
| **[NEW]** | `internal/repository/feed_limit.go` | PostgreSQL data access |
| **[NEW]** | `internal/usecase/feed_limit.go` | Business logic (limit checking, override logic) |
| **[NEW]** | `internal/usecase/feed_limit_test.go` | Comprehensive unit tests |
| **[NEW]** | `internal/delivery/http/feed_limit_handler.go` | API handlers + Swagger annotations |
| **[NEW]** | `pkg/migration/004_daily_feed_limit.sql` | Database migration |
| **[MOD]** | `cmd/skiix/main.go` | Dependency injection wiring |
| **[MOD]** | `internal/delivery/http/router.go` | Register new endpoints |

---

## 6. Error Handling

| Scenario | HTTP Code | Error Message |
| :--- | :--- | :--- |
| Invalid / Missing Token | `401` | `"unauthorized"` |
| Expired Token | `401` | `"token expired"` |
| Invalid JSON body | `400` | `"invalid request body"` |
| `is_enabled` is not boolean | `400` | `"is_enabled must be a boolean"` |
| Enabling without any limit set | `400` | `"at least one limit (posts or scroll time) must be set when enabling"` |
| `max_posts_per_day` invalid | `400` | `"max_posts_per_day must be a positive number (max 1000)"` |
| `max_scroll_minutes` invalid | `400` | `"max_scroll_minutes must be between 1 and 1440"` |
| `max_overrides_per_day` out of range | `400` | `"max_overrides_per_day must be between 0 and 10"` |
| Invalid `event_type` | `400` | `"invalid event_type (must be 'post_view' or 'scroll_time')"` |
| Negative `count` value | `400` | `"count must be a positive number"` |
| Override when limit not reached | `400` | `"limit has not been reached yet"` |
| Override already active | `400` | `"an override is already active"` |
| No overrides remaining | `400` | `"no overrides remaining today"` |
| Invalid `days` parameter | `400` | `"days must be between 1 and 30"` |
| Database failure | `500` | `"internal server error"` |

---

## 7. Edge Cases

| Edge Case | Expected System Behavior |
| :--- | :--- |
| User has never configured feed limits | Return defaults: `{ is_enabled: false, max_posts_per_day: null, max_scroll_minutes: null }`. Track requests still increment usage silently for analytics. |
| User enables limit with only `max_posts_per_day` | Only post-based limit is enforced. Scroll time is unlimited (`null`). |
| User enables limit with only `max_scroll_minutes` | Only time-based limit is enforced. Post count is unlimited (`null`). |
| Limit already reached, user sends another track | No further increment. Return `is_limit_reached: true` with override options. |
| Override expires mid-session | Next `track` call detects expired override and re-gates the feed. Frontend should poll periodically. |
| User changes limits after reaching today's limit | New limits applied immediately. If new limit > current usage → `is_limit_reached` resets to `false`. |
| User disables limit mid-day | Feed is immediately ungated. Usage data for today is preserved. |
| Midnight rollover during active session | Next `track` call creates a new `feed_usage_daily` row for the new date. Counters start from zero. Override from previous day is ignored (different `override_date`). |
| Concurrent track requests (race condition) | `ON CONFLICT DO UPDATE SET posts_viewed = posts_viewed + $1` handles this atomically in PostgreSQL. |
| User deletes account | All settings, usage, and override records cascade-deleted. |
| `count` value is excessively large (e.g., 9999) | Validated server-side: max 50 for `post_view`, max 30 for `scroll_time`. Prevents abuse. |
| Frontend fails to send track requests | Usage will be under-reported. This is acceptable — limits are advisory, not security-critical. |

---

## 8. Frontend Integration Notes

| Concern | Implementation Guidance |
| :--- | :--- |
| **Post View Tracking** | Frontend should call `POST /track` with `event_type: "post_view"` each time a post enters the viewport (or batch every 5 posts). |
| **Scroll Time Tracking** | Frontend should send heartbeats every 30 seconds while the user is actively scrolling: `event_type: "scroll_time", count: 1` (1 minute per 2 heartbeats). |
| **Limit Screen** | When `is_limit_reached: true` is received, the frontend should display a "Take a Break" interstitial screen with usage stats and a "Snooze" button. |
| **Override Countdown** | After activating an override, the frontend should show a countdown timer. When it expires, re-check the limit status. |
| **Usage Dashboard** | Use `GET /history?days=7` to render a weekly usage chart (bar chart or line graph). |

---

## 9. Dependencies & Prerequisites

| Dependency | Status | Notes |
| :--- | :--- | :--- |
| `users` table | ✅ Ready | Required for all foreign key references |
| JWT Middleware | ✅ Ready | Authentication for all endpoints |
| Feed/Posts module | ⚠️ Partially needed | Track endpoint works independently, but displaying "limit reached" screen requires feed integration |
| Push notification service | ⚠️ Pending | For sending "limit reached" push notifications (future enhancement) |
| Timezone support | ⚠️ Consideration | Currently uses UTC. Future: user-specific timezone for midnight reset. |

### Implementation Phases

| Phase | Scope | Dependency | Status |
| :--- | :--- | :--- | :--- |
| **Phase 1** | Settings CRUD — Enable/disable limits, configure thresholds. | Only Users + Auth | ✅ **Ready for Development** |
| **Phase 2** | Usage Tracking — Track post views and scroll time, detect limit reached. | Phase 1 complete | ⏳ Sequential |
| **Phase 3** | Override System — Snooze functionality with daily limits. | Phase 2 complete | ⏳ Sequential |
| **Phase 4** | Analytics — Usage history, trends, averages. | Phase 3 complete | ⏳ Sequential |
| **Phase 5** | Push Notifications — Alert user when approaching/reaching limit. | Notification service | ⏳ Future |

---

## 10. Acceptance Criteria

### Phase 1 (Settings CRUD)
- [ ] Users can configure `max_posts_per_day` and/or `max_scroll_minutes` via `PUT /api/v1/feed-limit`.
- [ ] Enabling without specifying at least one limit is rejected with `400`.
- [ ] `GET /api/v1/feed-limit` returns current settings or defaults for new users.
- [ ] `DELETE /api/v1/feed-limit` disables the feature while preserving settings.
- [ ] All inputs are validated (positive numbers, valid ranges).
- [ ] All endpoints are guarded by JWT authentication.
- [ ] Unit test coverage ≥ 90% on Usecase layer.

### Phase 2 (Usage Tracking)
- [ ] `POST /api/v1/feed-limit/track` correctly increments `posts_viewed` or `scroll_minutes`.
- [ ] A new usage row is created per user per day automatically.
- [ ] `is_limit_reached` is correctly determined by comparing usage vs limits.
- [ ] Response includes remaining quota for the frontend to display progress.
- [ ] Concurrent track requests handled atomically by the database.

### Phase 3 (Override System)
- [ ] `POST /api/v1/feed-limit/override` creates a time-limited override.
- [ ] Overrides are capped at `max_overrides_per_day`.
- [ ] Expired overrides are correctly detected on subsequent track calls.
- [ ] Cannot activate override if limit hasn't been reached.
- [ ] Cannot activate override if one is already active.

### Phase 4 (Analytics)
- [ ] `GET /api/v1/feed-limit/history?days=7` returns accurate daily summaries.
- [ ] Response includes calculated averages.
- [ ] Historical data persists indefinitely for trend analysis.

---

## 11. Estimated Timeline

| Task Area | Estimated Time | Primary Owner |
| :--- | :--- | :--- |
| Database migration (3 tables, indexes, constraints) | 1 - 2 hours | Furqan / Luthfi |
| Domain entities & interfaces | 1 - 2 hours | Furqan / Luthfi |
| Repository layer (PostgreSQL) | 3 - 4 hours | Furqan |
| Usecase layer (limit logic, override logic) + Unit Tests | 4 - 6 hours | Furqan |
| HTTP Handlers + Swagger annotations | 3 - 4 hours | Furqan / Luthfi |
| Frontend Integration Testing | 2 - 3 hours | Furqan + FE Dev (Michael) |
| Usage History & Analytics endpoint | 2 - 3 hours | Furqan / Luthfi |
| Code Review & Refactoring | 1 - 2 hours | Team |
| **Total Allocation (All Phases)** | **~5-7 Working Days** | |

---

## 12. Changelog

| Date | Version | Editor | Notes |
| :--- | :--- | :--- | :--- |
| — | 1.0 | Furqan | Initial feature description and high-level behavior |
| April 10, 2026 | 2.0 | Luthfi | Complete technical specification: 3-table DB schema with constraints, 7 RESTful API endpoints, override/snooze system, 4 system flows, frontend integration guidance, phased implementation plan, comprehensive error handling and edge cases. |
