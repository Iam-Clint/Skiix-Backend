# Skiix Backend

---

## 🚀 Quick Start 

### Prerequisites
- Go 1.26.1+

```bash
cp .env.example .env
```

**Supabase (IPv4):** the direct `db.*.supabase.co` host is often IPv6-only. In the [Supabase Dashboard](https://supabase.com/dashboard) → your project → **Connect** → **Session** (pooler), copy the connection string, put it in `.env` as `DATABASE_URL=...` (and add `?sslmode=require` if the URI has no query). That overrides the `DB_*` settings.

**Local PostgreSQL (optional):** `docker compose up -d` from this folder, or point `DB_*` at your instance and leave `DATABASE_URL` empty.

```bash
swag init -g cmd/skiix/main.go
go mod tidy

go run ./cmd/skiix
```

**Done!** API is ready at `http://localhost:8080`

---

## 📖 Access Documentation

- **Swagger UI**: http://localhost:8080/docs
- **API Base URL**: http://localhost:8080

---

## 📁 Project Structure

```
skiix-backend/
├── cmd/
│   └── skiix/
│       └── main.go                      # Application entry point
├── internal/                            # Private application packages
│   ├── domain/                          # Core business entities & interfaces
│   ├── usecase/                         # Business logic & use cases
│   ├── repository/                      # Data access layer
│   ├── infrastructure/                  # JWT, Bcrypt, DB connection
│   └── delivery/http/                   # HTTP handlers & routes
├── pkg/
│   └── migration/                       # Database migrations
├── docs/                                # Swagger API documentation (auto-generated)
├── docker-compose.yml                   # Docker Compose configuration
├── go.mod                               # Go module definition
└── go.sum                               # Dependency checksums
```

---

## 🎯 Default Credentials

**Database:**
- User: `postgres`
- Password: `postgres_password`
- Database: `skiix_db`

