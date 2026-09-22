# Skiix Backend — Engineering Progress Report

> **Author:** Luthfi Alfaridz (Xinn) — Backend Engineer  
> **Date:** April 18, 2026  
> **Version:** 1.0  
> **Repository:** [github.com/xinnxz/skiix-backend](https://github.com/xinnxz/skiix-backend)  

---

## 1. Executive Summary

This document provides a comprehensive overview of all backend engineering work completed for the **Skiix** social media application. Over the course of the development period, **three major features** were designed, implemented, tested, and deployed to the remote repository — all following the **Clean Architecture** pattern and professional software engineering practices.

| Phase | Feature | Status | Branch |
|-------|---------|--------|--------|
| 1 | Focus Mode Button | ✅ Complete | `feature/posts-module` |
| 2 | Circle Only Posts | ✅ Complete | `feature/posts-module` |
| 3 | Daily Feed Limit | ✅ Complete | `feature/daily-feed-limit` |

**Total deliverables:**
- **9 database tables** created (including constraints, indexes, and foreign keys)
- **20+ REST API endpoints** (all JWT-secured)
- **17+ unit tests** covering all business logic
- **Swagger/OpenAPI documentation** auto-generated for all endpoints

---

## 2. System Architecture

The Skiix backend is built with **Go (Golang)** using the **Gin-Gonic** web framework and **PostgreSQL** as the primary database. The codebase strictly follows the **Clean Architecture** pattern to ensure separation of concerns, testability, and maintainability.

```
┌─────────────────────────────────────────────────────────┐
│                    HTTP Layer (Gin)                       │
│         Handlers, Middleware, Routes, Swagger             │
├─────────────────────────────────────────────────────────┤
│                   Usecase Layer                           │
│           Business Logic, Validation, DTOs                │
├─────────────────────────────────────────────────────────┤
│                  Repository Layer                         │
│         PostgreSQL Queries, Data Access                   │
├─────────────────────────────────────────────────────────┤
│                   Domain Layer                            │
│        Entities, Interfaces, Request/Response             │
└─────────────────────────────────────────────────────────┘
```

### Project Structure

```
skiix-backend/
├── cmd/skiix/
│   └── main.go                          # Application entry point & DI wiring
├── internal/
│   ├── domain/                          # Entities & interfaces
│   │   ├── auth.go                      # User, JWT entities
│   │   ├── focus_mode.go                # Focus mode entities
│   │   └── feed_limit.go               # Feed limit entities & DTOs
│   ├── repository/                      # Data access layer
│   │   ├── user.go                      # User CRUD operations
│   │   ├── focus_mode.go                # Focus mode queries
│   │   └── feed_limit.go               # Feed limit atomic queries
│   ├── usecase/                         # Business logic layer
│   │   ├── auth.go                      # Authentication logic
│   │   ├── focus_mode.go                # Focus mode logic
│   │   ├── feed_limit.go               # Feed limit logic
│   │   ├── auth_test.go                 # Auth unit tests
│   │   └── feed_limit_test.go          # Feed limit unit tests
│   ├── delivery/http/                   # HTTP handlers
│   │   ├── auth_handler.go              # Auth endpoints
│   │   ├── oauth_handler.go             # Google OAuth endpoints
│   │   ├── focus_mode_handler.go        # Focus mode endpoints
│   │   ├── feed_limit_handler.go        # Feed limit endpoints
│   │   ├── middleware.go                # JWT middleware
│   │   └── routes.go                    # Route registration
│   └── infrastructure/                  # External services
│       ├── database.go                  # PostgreSQL connection
│       ├── jwt.go                       # JWT token service
│       └── password.go                  # BCrypt hashing
├── pkg/migration/
│   └── migration.go                     # Database migrations (all tables)
├── docs/                                # Swagger & documentation
│   ├── swagger.json                     # OpenAPI spec (JSON)
│   ├── swagger.yaml                     # OpenAPI spec (YAML)
│   └── docs.go                          # Swagger UI serve code
└── .env.example                         # Environment template
```

---

## 3. Feature Implementation Details

---

### 3.1 Phase 1: Focus Mode Button

**TRD Reference:** `docs/markdown/TRD - Focus Mode Button.md`

#### Objective
Enable users to activate a "Focus Mode" that blocks specific content categories from appearing in their feed, promoting healthier social media consumption.

#### Database Schema

| Table | Type | Description |
|-------|------|-------------|
| `focus_mode_settings` | 1:1 with users | Stores `is_enabled` toggle state per user |
| `focus_mode_blocked_categories` | 1:N with users | Each row represents one blocked category |

**Key constraints:**
- `UNIQUE(user_id)` on settings — ensures exactly one config per user
- `UNIQUE(user_id, category)` on blocked categories — prevents duplicates
- `ON DELETE CASCADE` — cleanup when user is deleted

#### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/focus-mode` | Get focus mode status + blocked categories |
| `PUT` | `/api/v1/focus-mode` | Toggle focus mode ON/OFF |
| `PUT` | `/api/v1/focus-mode/categories` | Update blocked categories list |

#### Implementation Notes
- All endpoints require JWT authentication
- Categories are bulk-replaced (DELETE all → INSERT new) for simplicity
- Settings are created lazily on first `GET` request (default: disabled)

---

### 3.2 Phase 2: Circle Only Posts (Close Friends)

**TRD Reference:** `docs/markdown/TRD - Circle Only Posts.md`

#### Objective
Allow users to create private "circles" (similar to Instagram Close Friends) and share posts visible only to circle members.

#### Database Schema

| Table | Type | Description |
|-------|------|-------------|
| `circles` | 1:N with users | Circle metadata (name, owner, member count) |
| `circle_members` | M:N junction | Links users to circles they belong to |
| `posts` (modified) | Extended | Added `visibility` enum + `circle_id` FK |

**Key constraints:**
- `CHECK (member_count <= 150)` — hard cap on circle size
- `UNIQUE(circle_id, user_id)` on members — prevents duplicate membership
- `visibility` column: `'public'` (default) or `'circle'`
- `circle_id` is `NULL` when visibility is `'public'`

#### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/circles` | Create a new circle |
| `GET` | `/api/v1/circles` | List user's circles |
| `PUT` | `/api/v1/circles/:id` | Update circle name |
| `DELETE` | `/api/v1/circles/:id` | Delete a circle |
| `POST` | `/api/v1/circles/:id/members` | Add member to circle |
| `DELETE` | `/api/v1/circles/:id/members/:userId` | Remove member |

#### Implementation Notes
- **Ownership enforcement:** Only the circle creator can modify/delete it
- **Atomic member_count:** `member_count` is updated within DB transactions to prevent race conditions
- **Feed privacy filter:** Circle-only posts are excluded from the public feed query and only visible to members

---

### 3.3 Phase 3: Daily Feed Limit (Screen Time Controller)

**TRD Reference:** `docs/markdown/TRD - Daily Feed Limit.md`

#### Objective
Help users manage their screen time by setting daily limits on feed consumption (posts viewed and scroll time), with a "Snooze" mechanism for temporary extensions.

#### Database Schema

| Table | Type | Description |
|-------|------|-------------|
| `feed_limit_settings` | 1:1 with users | Max posts/day, max scroll minutes, override config |
| `feed_usage_daily` | 1:N with users | One row per user per calendar day |
| `feed_limit_overrides` | 1:N with users | Audit log of snooze activations |

**Key constraints:**
- `UNIQUE(user_id)` on settings
- `UNIQUE(user_id, usage_date)` on daily usage — one row per day
- `CHECK (max_posts_per_day >= 1)` — minimum 1 post
- `CHECK (max_scroll_minutes >= 1)` — minimum 1 minute
- `CHECK (max_overrides_per_day BETWEEN 0 AND 10)` — bounded snooze count
- Indexes on `(user_id, usage_date)` and `(user_id, override_date)` for fast lookups

#### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/feed-limit` | Get current limit settings |
| `PUT` | `/api/v1/feed-limit` | Create/update limit settings |
| `DELETE` | `/api/v1/feed-limit` | Disable limits (preserves config) |
| `GET` | `/api/v1/feed-limit/usage` | Get today's consumption status |
| `POST` | `/api/v1/feed-limit/track` | Report consumption event |
| `POST` | `/api/v1/feed-limit/override` | Activate snooze extension |
| `GET` | `/api/v1/feed-limit/history` | Get usage history (7-30 days) |

#### Implementation Notes

**Atomic Tracking (Race-Condition Free):**
```sql
INSERT INTO feed_usage_daily (id, user_id, usage_date, posts_viewed, scroll_minutes)
VALUES ($1, $2, CURRENT_DATE, $3, $4)
ON CONFLICT (user_id, usage_date)
DO UPDATE SET
    posts_viewed = feed_usage_daily.posts_viewed + EXCLUDED.posts_viewed,
    scroll_minutes = feed_usage_daily.scroll_minutes + EXCLUDED.scroll_minutes,
    updated_at = NOW()
RETURNING *;
```
This ensures concurrent tracking requests from the mobile client never cause data inconsistency.

**Date-Based Auto Reset:**
Each day automatically gets a fresh usage row. There is no "reset" job needed — the `UNIQUE(user_id, usage_date)` constraint combined with `ON CONFLICT` handles it naturally.

**Snooze System Flow:**
1. Client calls `POST /track` → response includes `is_limit_reached: true`
2. Client shows "Limit Reached" UI with "Snooze for 15 min?" button
3. Client calls `POST /override` → backend validates:
   - Is limit actually reached? ✅
   - Has user exceeded max daily overrides? ✅
   - Is there an active (non-expired) override already? ✅
4. If all checks pass → creates override with `expires_at = now + 15 min`

---

## 4. Testing Strategy

### Unit Tests (Usecase Layer)

All business logic is covered by unit tests using the **testify** framework with **mock repositories**. This ensures tests run fast (no database dependency) and validate pure logic.

| Test Category | Count | Description |
|--------------|-------|-------------|
| Auth Tests | 10+ | Registration, login, token validation |
| Feed Limit Tests | 17+ | Settings, tracking, override, history |

**Key test scenarios covered:**
- ✅ Enable/disable settings with valid/invalid ranges
- ✅ Track post views and scroll time (atomic increment)
- ✅ Limit detection when either metric exceeds threshold
- ✅ Override activation (success, daily cap exceeded, no limit reached)
- ✅ Override expiration handling
- ✅ Usage history aggregation with averages
- ✅ Edge cases: disabled limits, zero counts, NULL thresholds

### Build Verification
```bash
go build ./...    # ✅ Compiles without errors
go test ./...     # ✅ All tests pass
```

---

## 5. API Documentation

**Swagger UI** is auto-generated using `swag` annotations on every handler function and is available at:

```
http://localhost:8080/swagger/index.html
```

The following specification files are committed to the repository:
- `docs/swagger.json` — Machine-readable OpenAPI 2.0 spec
- `docs/swagger.yaml` — Human-readable YAML version
- `docs/docs.go` — Go source for serving Swagger UI

These files can be imported directly into **Postman**, **Insomnia**, or any OpenAPI-compatible tool for testing.

---

## 6. Git History & Branch Strategy

### Branch Structure

| Branch | Purpose | Status |
|--------|---------|--------|
| `main` | Production-ready code | Base (Auth + Focus Mode) |
| `feature/posts-module` | Circle Only Posts | ✅ Ready for PR |
| `feature/daily-feed-limit` | Daily Feed Limit + Swagger | ✅ Ready for PR |

### Commit Convention
All commits follow the **Conventional Commits** standard:
- `feat(scope):` — New feature implementation
- `docs:` — Documentation changes
- `test:` — Test additions
- `refactor:` — Code restructuring

---

## 7. Environment Setup

To run the backend locally:

```bash
# 1. Clone the repository
git clone https://github.com/xinnxz/skiix-backend.git
cd skiix-backend

# 2. Configure environment
cp .env.example .env
# Edit .env with your PostgreSQL credentials

# 3. Start the server
go run cmd/skiix/main.go

# 4. Open Swagger UI
# http://localhost:8080/swagger/index.html
```

### Required Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | Database username | `postgres` |
| `DB_PASSWORD` | Database password | `your_password` |
| `DB_NAME` | Database name | `skiix_db` |
| `DB_SSLMODE` | SSL mode | `disable` |
| `JWT_SECRET` | JWT signing key | `your-secret-key` |
| `BASE_URL` | Server base URL | `http://localhost:8080` |

---

## 8. Summary & Next Steps

### Completed ✅
- [x] Focus Mode Button — Full stack (DB → API)
- [x] Circle Only Posts — Full stack (DB → API)
- [x] Daily Feed Limit — Full stack (DB → API)
- [x] Unit testing — 17+ test cases on business logic
- [x] Swagger documentation — All 20+ endpoints documented
- [x] Git workflow — Feature branches pushed to remote

### Recommended Next Steps
- [ ] **Code Review & PR Merge** — Merge feature branches into `main`
- [ ] **Frontend Integration** — Mobile team consumes the API using Swagger spec
- [ ] **CI/CD Pipeline** — Automate testing and deployment
- [ ] **Load Testing** — Validate atomic tracking under concurrent load
- [ ] **Monitoring & Logging** — Add structured logging for production

---

*End of Report*
