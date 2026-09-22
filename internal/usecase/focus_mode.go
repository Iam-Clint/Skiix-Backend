package usecase

import (
	"errors"
	"strings"

	"github.com/skiix-backend/internal/domain"
)

// Maximum number of categories a user can block.
// Prevents abuse and keeps the feature manageable.
const maxBlockedCategories = 20

// Maximum character length for a single category name.
const maxCategoryNameLength = 100

// FocusModeUsecase defines the business operations for Focus Mode.
// This interface is consumed by the HTTP handler layer.
type FocusModeUsecase interface {
	GetStatus(userID string) (*domain.FocusModeStatus, error)
	Toggle(userID string, isEnabled bool) error
	SetBlockedCategories(userID string, categories []string) error
}

// FocusModeUsecaseImpl is the concrete implementation of FocusModeUsecase.
// It depends on FocusModeRepository (injected via constructor) and contains
// all validation and business rules.
type FocusModeUsecaseImpl struct {
	focusRepo domain.FocusModeRepository
}

// NewFocusModeUsecase creates a new FocusModeUsecase with the given repository.
// This follows the constructor injection pattern used across the codebase.
func NewFocusModeUsecase(focusRepo domain.FocusModeRepository) FocusModeUsecase {
	return &FocusModeUsecaseImpl{
		focusRepo: focusRepo,
	}
}

// GetStatus returns the current focus mode status for a user.
// If the user has never configured focus mode, sensible defaults are returned:
//   - is_enabled: false
//   - blocked_categories: [] (empty array, not null)
//
// This ensures the frontend always receives a valid, parseable response.
func (u *FocusModeUsecaseImpl) GetStatus(userID string) (*domain.FocusModeStatus, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}

	// Fetch settings — may be nil if user never configured
	settings, err := u.focusRepo.GetSettings(userID)
	if err != nil {
		return nil, err
	}

	// Build response with defaults
	status := &domain.FocusModeStatus{
		IsEnabled:         false,
		BlockedCategories: make([]string, 0),
	}

	if settings != nil {
		status.IsEnabled = settings.IsEnabled
	}

	// Fetch blocked categories
	categories, err := u.focusRepo.GetBlockedCategories(userID)
	if err != nil {
		return nil, err
	}

	if categories != nil {
		status.BlockedCategories = categories
	}

	return status, nil
}

// Toggle enables or disables focus mode for a user.
// Uses UPSERT internally — works for both first-time setup and updates.
func (u *FocusModeUsecaseImpl) Toggle(userID string, isEnabled bool) error {
	if userID == "" {
		return errors.New("user id is required")
	}

	return u.focusRepo.UpsertSettings(userID, isEnabled)
}

// SetBlockedCategories validates and saves the list of blocked categories.
// Business rules enforced:
//  1. Maximum 20 categories per user
//  2. Each category name: 1-100 characters
//  3. Duplicate categories are auto-removed (deduplicated)
//  4. Category names are trimmed of leading/trailing whitespace
//  5. Empty array [] clears all categories (valid operation)
func (u *FocusModeUsecaseImpl) SetBlockedCategories(userID string, categories []string) error {
	if userID == "" {
		return errors.New("user id is required")
	}

	// Deduplicate and validate categories
	cleaned := u.deduplicateCategories(categories)

	// Validate count
	if len(cleaned) > maxBlockedCategories {
		return errors.New("too many categories (max 20)")
	}

	// Validate each category name
	for _, name := range cleaned {
		if name == "" {
			return errors.New("category name cannot be empty")
		}
		if len(name) > maxCategoryNameLength {
			return errors.New("category name too long (max 100 characters)")
		}
	}

	return u.focusRepo.SetBlockedCategories(userID, cleaned)
}

// deduplicateCategories removes duplicate entries from the categories slice.
// It trims whitespace from each name and uses a map to track seen entries.
// The order of first appearance is preserved.
func (u *FocusModeUsecaseImpl) deduplicateCategories(categories []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(categories))

	for _, category := range categories {
		trimmed := strings.TrimSpace(category)
		if trimmed == "" {
			continue
		}
		if !seen[trimmed] {
			seen[trimmed] = true
			result = append(result, trimmed)
		}
	}

	return result
}
