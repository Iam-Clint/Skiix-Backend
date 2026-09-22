package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skiix-backend/internal/domain"
)

// PostgresFocusModeRepository implements domain.FocusModeRepository
// using raw SQL queries with parameterized inputs to prevent SQL injection.
type PostgresFocusModeRepository struct {
	db *sql.DB
}

// NewPostgresFocusModeRepository creates a new repository instance.
// Accepts *sql.DB (the shared database connection pool) and returns
// the domain interface, maintaining dependency inversion.
func NewPostgresFocusModeRepository(db *sql.DB) domain.FocusModeRepository {
	return &PostgresFocusModeRepository{
		db: db,
	}
}

// GetSettings retrieves focus mode settings for a specific user.
// If no record exists (new user), returns nil without error so the
// usecase layer can return sensible defaults.
func (r *PostgresFocusModeRepository) GetSettings(userID string) (*domain.FocusModeSetting, error) {
	query := `
		SELECT id, user_id, is_enabled, created_at, updated_at
		FROM focus_mode_settings
		WHERE user_id = $1
	`

	setting := &domain.FocusModeSetting{}
	err := r.db.QueryRow(query, userID).Scan(
		&setting.ID,
		&setting.UserID,
		&setting.IsEnabled,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get focus mode settings: %w", err)
	}

	return setting, nil
}

// UpsertSettings creates or updates the focus mode toggle for a user.
// Uses PostgreSQL's ON CONFLICT (UPSERT) pattern to handle the case
// where settings don't exist yet OR need to be updated.
// This approach is safe against race conditions — the last write wins.
func (r *PostgresFocusModeRepository) UpsertSettings(userID string, isEnabled bool) error {
	query := `
		INSERT INTO focus_mode_settings (id, user_id, is_enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			is_enabled = EXCLUDED.is_enabled,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	_, err := r.db.Exec(query, uuid.New().String(), userID, isEnabled, now, now)
	if err != nil {
		return fmt.Errorf("failed to upsert focus mode settings: %w", err)
	}

	return nil
}

// GetBlockedCategories returns all category names blocked by a user.
// Always returns a non-nil slice — if no categories exist, returns [].
func (r *PostgresFocusModeRepository) GetBlockedCategories(userID string) ([]string, error) {
	query := `
		SELECT category_name
		FROM focus_mode_blocked_categories
		WHERE user_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get blocked categories: %w", err)
	}
	defer rows.Close()

	categories := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating categories: %w", err)
	}

	return categories, nil
}

// SetBlockedCategories replaces ALL blocked categories for a user atomically.
// This uses a database transaction to ensure consistency:
//  1. DELETE all existing categories for the user
//  2. INSERT the new list of categories
//
// If any step fails, the entire operation is rolled back.
// An empty categories slice effectively clears all blocked categories.
func (r *PostgresFocusModeRepository) SetBlockedCategories(userID string, categories []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Step 1: Remove all existing blocked categories for this user
	deleteQuery := `DELETE FROM focus_mode_blocked_categories WHERE user_id = $1`
	if _, err := tx.Exec(deleteQuery, userID); err != nil {
		return fmt.Errorf("failed to delete existing categories: %w", err)
	}

	// Step 2: Insert each new category
	if len(categories) > 0 {
		insertQuery := `
			INSERT INTO focus_mode_blocked_categories (id, user_id, category_name, created_at)
			VALUES ($1, $2, $3, $4)
		`

		now := time.Now()
		for _, category := range categories {
			_, err := tx.Exec(insertQuery, uuid.New().String(), userID, category, now)
			if err != nil {
				return fmt.Errorf("failed to insert category '%s': %w", category, err)
			}
		}
	}

	// Commit the transaction — both steps succeed or neither does
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
