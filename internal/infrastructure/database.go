package infrastructure

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// BuildPostgresDSN returns a lib/pq connection string. Prefer URL form with
// query parameters so special characters in passwords are encoded for Supabase, etc.
func BuildPostgresDSN(config DatabaseConfig) string {
	host := strings.TrimSpace(config.Host)
	port := strings.TrimSpace(config.Port)
	if port == "" {
		port = "5432"
	}
	ssl := strings.TrimSpace(config.SSLMode)
	if ssl == "" {
		if host == "localhost" || host == "127.0.0.1" {
			ssl = "disable"
		} else {
			ssl = "require"
		}
	}

	dbname := strings.TrimSpace(config.DBName)
	if dbname == "" {
		dbname = "postgres"
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(config.User, config.Password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + dbname,
	}
	q := url.Values{}
	q.Set("sslmode", ssl)
	// avoid hanging on bad networks
	q.Set("connect_timeout", "15")
	// Direct db.*.supabase.co often resolves to IPv6 first; many LANs have no IPv6
	// route ("no route to host"). If DNS returns an A record, force that address.
	if v4, ok := firstIPv4ForHost(host); ok {
		q.Set("hostaddr", v4)
		log.Printf("database: using IPv4 hostaddr=%s for host %s (avoids broken IPv6 paths)", v4, host)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

// firstIPv4ForHost returns the first A record, if any. Many Supabase direct hosts
// have both A and AAAA; lib/pq may pick IPv6 and fail on networks without IPv6.
func firstIPv4ForHost(host string) (string, bool) {
	host = strings.TrimSpace(host)
	if host == "" || host == "localhost" || host == "127.0.0.1" {
		return "", false
	}
	if net.ParseIP(host) != nil {
		return "", false
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return "", false
	}
	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			return v4.String(), true
		}
	}
	return "", false
}

// NewPostgresConnectionFromURL uses a full connection string (e.g. from Supabase
// "Connection pooling" / session mode) and applies pool settings. The password must
// already be URL-encoded in the string.
func NewPostgresConnectionFromURL(dsnIn string) (*sql.DB, error) {
	dsnIn = strings.TrimSpace(dsnIn)
	if dsnIn == "" {
		return nil, fmt.Errorf("empty DATABASE_URL")
	}

	// lib/pq accepts both postgres: and postgresql: schemes; normalize for parsing.
	canon := dsnIn
	if strings.HasPrefix(canon, "postgresql://") {
		canon = "postgres://" + strings.TrimPrefix(canon, "postgresql://")
	}

	u, err := url.Parse(canon)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	username := ""
	if u.User != nil {
		username = u.User.Username()
	}
	sslInURL := u.Query().Get("sslmode")
	if sslInURL == "" {
		sslInURL = "(default)"
	}
	log.Printf(
		"database: connecting via DATABASE_URL (host=%s path=%s user=%s sslmode_in_url=%s)",
		u.Hostname(), u.Path, username, sslInURL,
	)

	q := u.Query()
	if q.Get("sslmode") == "" && u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" {
		q.Set("sslmode", "require")
	}
	if q.Get("connect_timeout") == "" {
		q.Set("connect_timeout", "15")
	}
	if q.Get("hostaddr") == "" {
		if v4, ok := firstIPv4ForHost(u.Hostname()); ok {
			q.Set("hostaddr", v4)
			log.Printf("database: using IPv4 hostaddr=%s for %s (from DATABASE_URL)", v4, u.Hostname())
		}
	}
	u.RawQuery = q.Encode()
	dsn := u.String()

	return openPooledWithPing(dsn)
}

func openPooledWithPing(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		wrapped := fmt.Errorf("failed to connect to database: %w", err)
		if strings.Contains(err.Error(), "no route to host") {
			return nil, fmt.Errorf(
				"%w — try DATABASE_URL=Session pooler (Supabase → Connect) or a different network; direct db host often uses IPv6",
				wrapped,
			)
		}
		return nil, wrapped
	}
	log.Println("database: connection verified (ping OK)")
	return db, nil
}

func NewPostgresConnection(config DatabaseConfig) (*sql.DB, error) {
	dsn := BuildPostgresDSN(config)

	// Do not log password; connection details are enough for support/debug.
	ssl := strings.TrimSpace(config.SSLMode)
	if ssl == "" {
		if strings.TrimSpace(config.Host) == "localhost" || strings.TrimSpace(config.Host) == "127.0.0.1" {
			ssl = "disable"
		} else {
			ssl = "require"
		}
	}
	log.Printf(
		"database: connecting (host=%s port=%s user=%s dbname=%s sslmode=%s)",
		config.Host, config.Port, config.User, config.DBName, ssl,
	)

	return openPooledWithPing(dsn)
}

func CloseDatabase(db *sql.DB) error {
	return db.Close()
}
