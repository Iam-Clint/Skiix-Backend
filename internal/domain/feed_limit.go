package domain

import "time"

// =============================================
// Entities
// =============================================

// FeedLimitSetting stores a user's daily feed limit preferences.
// There is exactly one record per user (1:1 relationship).
type FeedLimitSetting struct {
	ID                      string    `json:"id"`
	UserID                  string    `json:"user_id"`
	IsEnabled               bool      `json:"is_enabled"`
	MaxPostsPerDay          *int      `json:"max_posts_per_day"`         // NULL = unlimited
	MaxScrollMinutes        *int      `json:"max_scroll_minutes"`        // NULL = unlimited
	MaxOverridesPerDay      int       `json:"max_overrides_per_day"`     // default 3
	OverrideDurationMinutes int       `json:"override_duration_minutes"` // default 15
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// FeedUsageDaily tracks how much feed the user has consumed today.
// Each row = one user, one calendar date.
type FeedUsageDaily struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	UsageDate      time.Time  `json:"usage_date"`
	PostsViewed    int        `json:"posts_viewed"`
	ScrollMinutes  int        `json:"scroll_minutes"`
	LimitReachedAt *time.Time `json:"limit_reached_at"`
	IsLimitReached bool       `json:"is_limit_reached"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// FeedLimitOverride records a single snooze/override activation by the user.
type FeedLimitOverride struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	OverrideDate time.Time `json:"override_date"`
	ActivatedAt  time.Time `json:"activated_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// =============================================
// Request/Response DTOs
// =============================================

// UpdateFeedLimitRequest holds the payload for PUT /feed-limit.
type UpdateFeedLimitRequest struct {
	IsEnabled               bool `json:"is_enabled"`
	MaxPostsPerDay          *int `json:"max_posts_per_day"`
	MaxScrollMinutes        *int `json:"max_scroll_minutes"`
	MaxOverridesPerDay      *int `json:"max_overrides_per_day"`
	OverrideDurationMinutes *int `json:"override_duration_minutes"`
}

// TrackConsumptionRequest holds the payload for POST /feed-limit/track.
// EventType is either "post_view" or "scroll_time".
// Count is number of posts viewed or minutes scrolled.
type TrackConsumptionRequest struct {
	EventType string `json:"event_type"`
	Count     int    `json:"count"`
}

// TrackResponse is the response after tracking consumption, shows quota status.
type TrackResponse struct {
	IsLimitReached         bool   `json:"is_limit_reached"`
	PostsViewed            int    `json:"posts_viewed"`
	ScrollMinutes          int    `json:"scroll_minutes"`
	PostsRemaining         *int   `json:"posts_remaining"`
	ScrollMinutesRemaining *int   `json:"scroll_minutes_remaining"`
	CanOverride            bool   `json:"can_override,omitempty"`
	OverridesRemaining     int    `json:"overrides_remaining,omitempty"`
	Message                string `json:"message,omitempty"`
}

// OverrideInfo holds the current snooze override status for the user.
type OverrideInfo struct {
	IsActive           bool       `json:"is_active"`
	ExpiresAt          *time.Time `json:"expires_at"`
	OverridesUsedToday int        `json:"overrides_used_today"`
	OverridesRemaining int        `json:"overrides_remaining"`
}

// FeedUsageResponse is the full daily usage status returned by GET /feed-limit/usage.
type FeedUsageResponse struct {
	Date           string            `json:"date"`
	PostsViewed    int               `json:"posts_viewed"`
	ScrollMinutes  int               `json:"scroll_minutes"`
	Limits         *FeedLimitSetting `json:"limits"`
	Progress       UsageProgress     `json:"progress"`
	IsLimitReached bool              `json:"is_limit_reached"`
	LimitReachedAt *time.Time        `json:"limit_reached_at"`
	Override       OverrideInfo      `json:"override"`
}

// UsageProgress shows percentage of limit consumed for display.
type UsageProgress struct {
	PostsPercentage  *float64 `json:"posts_percentage"`
	ScrollPercentage *float64 `json:"scroll_percentage"`
}

// DailyUsageHistoryItem is a single day's usage summary.
type DailyUsageHistoryItem struct {
	Date          string `json:"date"`
	PostsViewed   int    `json:"posts_viewed"`
	ScrollMinutes int    `json:"scroll_minutes"`
	LimitReached  bool   `json:"limit_reached"`
}

// UsageHistoryResponse is returned by GET /feed-limit/history.
type UsageHistoryResponse struct {
	History  []DailyUsageHistoryItem `json:"history"`
	Averages UsageAverages           `json:"averages"`
}

// UsageAverages holds aggregated statistics for the history period.
type UsageAverages struct {
	AvgPostsPerDay   float64 `json:"avg_posts_per_day"`
	AvgScrollMinutes float64 `json:"avg_scroll_minutes"`
	DaysLimitReached int     `json:"days_limit_reached"`
	TotalDays        int     `json:"total_days"`
}

// =============================================
// Repository Interface
// =============================================

// FeedLimitRepository defines the data access operations for the daily feed limit feature.
type FeedLimitRepository interface {
	// GetSettings retrieves the user's limit settings (returns defaults if not configured).
	GetSettings(userID string) (*FeedLimitSetting, error)
	// UpsertSettings creates or updates the user's limit settings.
	UpsertSettings(setting *FeedLimitSetting) error
	// DisableSettings sets is_enabled=false without deleting the configuration.
	DisableSettings(userID string) error

	// GetTodayUsage retrieves (or creates) the user's usage row for today.
	GetTodayUsage(userID string) (*FeedUsageDaily, error)
	// IncrementUsage atomically increments posts_viewed or scroll_minutes counter.
	IncrementUsage(userID string, postsViewed int, scrollMinutes int) (*FeedUsageDaily, error)
	// MarkLimitReached flags the usage row as limit reached with a timestamp.
	MarkLimitReached(userID string) error
	// GetUsageHistory returns daily usage rows for the past N days.
	GetUsageHistory(userID string, days int) ([]FeedUsageDaily, error)

	// CreateOverride inserts a new snooze override for the user.
	CreateOverride(userID string, expiresAt time.Time) (*FeedLimitOverride, error)
	// GetActiveOverride returns the currently active (non-expired) override, or nil.
	GetActiveOverride(userID string) (*FeedLimitOverride, error)
	// CountTodayOverrides counts how many overrides the user has activated today.
	CountTodayOverrides(userID string) (int, error)
}
