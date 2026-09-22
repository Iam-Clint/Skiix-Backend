package domain

import "time"

// FocusModeSetting represents a user's focus mode configuration.
// Each user has at most one FocusModeSetting record (1:1 relationship).
type FocusModeSetting struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	IsEnabled bool      `json:"is_enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FocusBlockedCategory represents a content category that a user
// has chosen to block when Focus Mode is active.
type FocusBlockedCategory struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	CategoryName string    `json:"category_name"`
	CreatedAt    time.Time `json:"created_at"`
}

// FocusModeStatus is the response DTO combining settings and blocked categories.
type FocusModeStatus struct {
	IsEnabled         bool     `json:"is_enabled"`
	BlockedCategories []string `json:"blocked_categories"`
}

// FocusModeRepository defines the data access contract for focus mode.
// The repository layer is responsible ONLY for database operations,
// without any business logic or validation.
type FocusModeRepository interface {
	// GetSettings retrieves the focus mode settings for a user.
	// Returns nil (not error) if no settings exist yet.
	GetSettings(userID string) (*FocusModeSetting, error)

	// UpsertSettings creates or updates the focus mode toggle for a user.
	// Uses PostgreSQL ON CONFLICT to handle concurrent requests safely.
	UpsertSettings(userID string, isEnabled bool) error

	// GetBlockedCategories returns the list of blocked category names for a user.
	// Returns an empty slice (not nil) if no categories are blocked.
	GetBlockedCategories(userID string) ([]string, error)

	// SetBlockedCategories replaces all blocked categories for a user.
	// This is an atomic operation: DELETE all existing + INSERT new ones
	// inside a single database transaction.
	SetBlockedCategories(userID string, categories []string) error
}
