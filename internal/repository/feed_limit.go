package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skiix-backend/internal/domain"
)

// PostgresFeedLimitRepository implements domain.FeedLimitRepository using PostgreSQL.
type PostgresFeedLimitRepository struct {
	db *sql.DB
}

// NewPostgresFeedLimitRepository creates a new PostgresFeedLimitRepository.
func NewPostgresFeedLimitRepository(db *sql.DB) *PostgresFeedLimitRepository {
	return &PostgresFeedLimitRepository{db: db}
}

// GetSettings retrieves the user's feed limit settings.
// If the user has never configured limits, it returns a safe default (disabled).
func (r *PostgresFeedLimitRepository) GetSettings(userID string) (*domain.FeedLimitSetting, error) {
	query := `
		SELECT id, user_id, is_enabled, max_posts_per_day, max_scroll_minutes,
		       max_overrides_per_day, override_duration_minutes, created_at, updated_at
		FROM feed_limit_settings
		WHERE user_id = $1
	`
	var s domain.FeedLimitSetting
	err := r.db.QueryRow(query, userID).Scan(
		&s.ID, &s.UserID, &s.IsEnabled,
		&s.MaxPostsPerDay, &s.MaxScrollMinutes,
		&s.MaxOverridesPerDay, &s.OverrideDurationMinutes,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		// Return safe defaults — feature is opt-in so still disabled
		return &domain.FeedLimitSetting{
			UserID:                  userID,
			IsEnabled:               false,
			MaxOverridesPerDay:      3,
			OverrideDurationMinutes: 15,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feed limit settings: %w", err)
	}
	return &s, nil
}

// UpsertSettings creates or updates the user's feed limit settings.
func (r *PostgresFeedLimitRepository) UpsertSettings(s *domain.FeedLimitSetting) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	query := `
		INSERT INTO feed_limit_settings
			(id, user_id, is_enabled, max_posts_per_day, max_scroll_minutes,
			 max_overrides_per_day, override_duration_minutes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			is_enabled                = EXCLUDED.is_enabled,
			max_posts_per_day         = EXCLUDED.max_posts_per_day,
			max_scroll_minutes        = EXCLUDED.max_scroll_minutes,
			max_overrides_per_day     = EXCLUDED.max_overrides_per_day,
			override_duration_minutes = EXCLUDED.override_duration_minutes,
			updated_at                = NOW()
	`
	_, err := r.db.Exec(query,
		s.ID, s.UserID, s.IsEnabled,
		s.MaxPostsPerDay, s.MaxScrollMinutes,
		s.MaxOverridesPerDay, s.OverrideDurationMinutes,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert feed limit settings: %w", err)
	}
	return nil
}

// DisableSettings sets is_enabled=false without deleting the user's configuration.
func (r *PostgresFeedLimitRepository) DisableSettings(userID string) error {
	query := `UPDATE feed_limit_settings SET is_enabled = FALSE, updated_at = NOW() WHERE user_id = $1`
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to disable feed limit: %w", err)
	}
	return nil
}

// GetTodayUsage returns the user's usage row for today, creating one if it doesn't exist.
// This implements the "date-based auto reset" — each new day creates a fresh row.
func (r *PostgresFeedLimitRepository) GetTodayUsage(userID string) (*domain.FeedUsageDaily, error) {
	// Try to get existing row for today
	query := `
		SELECT id, user_id, usage_date, posts_viewed, scroll_minutes,
		       limit_reached_at, is_limit_reached, created_at, updated_at
		FROM feed_usage_daily
		WHERE user_id = $1 AND usage_date = CURRENT_DATE
	`
	var u domain.FeedUsageDaily
	err := r.db.QueryRow(query, userID).Scan(
		&u.ID, &u.UserID, &u.UsageDate,
		&u.PostsViewed, &u.ScrollMinutes,
		&u.LimitReachedAt, &u.IsLimitReached,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		// Create a new usage row for today
		insertQuery := `
			INSERT INTO feed_usage_daily (id, user_id, usage_date, posts_viewed, scroll_minutes, is_limit_reached, created_at, updated_at)
			VALUES ($1, $2, CURRENT_DATE, 0, 0, FALSE, NOW(), NOW())
			ON CONFLICT (user_id, usage_date) DO NOTHING
			RETURNING id, user_id, usage_date, posts_viewed, scroll_minutes, limit_reached_at, is_limit_reached, created_at, updated_at
		`
		newID := uuid.New().String()
		err2 := r.db.QueryRow(insertQuery, newID, userID).Scan(
			&u.ID, &u.UserID, &u.UsageDate,
			&u.PostsViewed, &u.ScrollMinutes,
			&u.LimitReachedAt, &u.IsLimitReached,
			&u.CreatedAt, &u.UpdatedAt,
		)
		if err2 == sql.ErrNoRows {
			// Row was inserted by concurrent request, fetch it again
			return r.GetTodayUsage(userID)
		}
		if err2 != nil {
			return nil, fmt.Errorf("failed to create today's usage: %w", err2)
		}
		return &u, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get today's usage: %w", err)
	}
	return &u, nil
}

// IncrementUsage atomically increments posts_viewed or scroll_minutes for today.
// Uses ON CONFLICT DO UPDATE so concurrent requests are safe.
func (r *PostgresFeedLimitRepository) IncrementUsage(userID string, postsViewed int, scrollMinutes int) (*domain.FeedUsageDaily, error) {
	// Ensure today's row exists first
	newID := uuid.New().String()
	upsertQuery := `
		INSERT INTO feed_usage_daily (id, user_id, usage_date, posts_viewed, scroll_minutes, is_limit_reached, created_at, updated_at)
		VALUES ($1, $2, CURRENT_DATE, $3, $4, FALSE, NOW(), NOW())
		ON CONFLICT (user_id, usage_date) DO UPDATE SET
			posts_viewed   = feed_usage_daily.posts_viewed   + $3,
			scroll_minutes = feed_usage_daily.scroll_minutes + $4,
			updated_at     = NOW()
		RETURNING id, user_id, usage_date, posts_viewed, scroll_minutes, limit_reached_at, is_limit_reached, created_at, updated_at
	`
	var u domain.FeedUsageDaily
	err := r.db.QueryRow(upsertQuery, newID, userID, postsViewed, scrollMinutes).Scan(
		&u.ID, &u.UserID, &u.UsageDate,
		&u.PostsViewed, &u.ScrollMinutes,
		&u.LimitReachedAt, &u.IsLimitReached,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to increment usage: %w", err)
	}
	return &u, nil
}

// MarkLimitReached updates the today's row to flag limit as reached with a timestamp.
func (r *PostgresFeedLimitRepository) MarkLimitReached(userID string) error {
	query := `
		UPDATE feed_usage_daily
		SET is_limit_reached = TRUE, limit_reached_at = NOW(), updated_at = NOW()
		WHERE user_id = $1 AND usage_date = CURRENT_DATE AND is_limit_reached = FALSE
	`
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to mark limit reached: %w", err)
	}
	return nil
}

// GetUsageHistory returns daily usage summaries for the past N days.
func (r *PostgresFeedLimitRepository) GetUsageHistory(userID string, days int) ([]domain.FeedUsageDaily, error) {
	query := `
		SELECT id, user_id, usage_date, posts_viewed, scroll_minutes,
		       limit_reached_at, is_limit_reached, created_at, updated_at
		FROM feed_usage_daily
		WHERE user_id = $1 AND usage_date >= CURRENT_DATE - INTERVAL '1 day' * $2
		ORDER BY usage_date DESC
	`
	rows, err := r.db.Query(query, userID, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage history: %w", err)
	}
	defer rows.Close()

	var result []domain.FeedUsageDaily
	for rows.Next() {
		var u domain.FeedUsageDaily
		if err := rows.Scan(
			&u.ID, &u.UserID, &u.UsageDate,
			&u.PostsViewed, &u.ScrollMinutes,
			&u.LimitReachedAt, &u.IsLimitReached,
			&u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

// CreateOverride inserts a new snooze override for the user.
func (r *PostgresFeedLimitRepository) CreateOverride(userID string, expiresAt time.Time) (*domain.FeedLimitOverride, error) {
	id := uuid.New().String()
	query := `
		INSERT INTO feed_limit_overrides (id, user_id, override_date, activated_at, expires_at, created_at)
		VALUES ($1, $2, CURRENT_DATE, NOW(), $3, NOW())
		RETURNING id, user_id, override_date, activated_at, expires_at, created_at
	`
	var o domain.FeedLimitOverride
	err := r.db.QueryRow(query, id, userID, expiresAt).Scan(
		&o.ID, &o.UserID, &o.OverrideDate, &o.ActivatedAt, &o.ExpiresAt, &o.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create override: %w", err)
	}
	return &o, nil
}

// GetActiveOverride returns the latest non-expired override for the user, or nil if none.
func (r *PostgresFeedLimitRepository) GetActiveOverride(userID string) (*domain.FeedLimitOverride, error) {
	query := `
		SELECT id, user_id, override_date, activated_at, expires_at, created_at
		FROM feed_limit_overrides
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY expires_at DESC
		LIMIT 1
	`
	var o domain.FeedLimitOverride
	err := r.db.QueryRow(query, userID).Scan(
		&o.ID, &o.UserID, &o.OverrideDate, &o.ActivatedAt, &o.ExpiresAt, &o.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No active override — not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active override: %w", err)
	}
	return &o, nil
}

// CountTodayOverrides counts how many snooze overrides the user has activated today.
func (r *PostgresFeedLimitRepository) CountTodayOverrides(userID string) (int, error) {
	query := `SELECT COUNT(*) FROM feed_limit_overrides WHERE user_id = $1 AND override_date = CURRENT_DATE`
	var count int
	if err := r.db.QueryRow(query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count today's overrides: %w", err)
	}
	return count, nil
}
