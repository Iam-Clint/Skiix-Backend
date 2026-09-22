package usecase

import (
	"errors"
	"fmt"
	"time"

	"github.com/skiix-backend/internal/domain"
)

// FeedLimitUsecase defines business logic for the daily feed limit feature.
type FeedLimitUsecase interface {
	// GetSettings returns the user's current feed limit configuration.
	GetSettings(userID string) (*domain.FeedLimitSetting, error)
	// UpdateSettings creates or updates limit settings with full validation.
	UpdateSettings(userID string, req domain.UpdateFeedLimitRequest) (*domain.FeedLimitSetting, error)
	// DisableLimit turns off the feature while preserving settings.
	DisableLimit(userID string) error
	// GetTodayUsage returns full usage + progress + override status for today.
	GetTodayUsage(userID string) (*domain.FeedUsageResponse, error)
	// TrackConsumption records a consumption event and returns whether limit is reached.
	TrackConsumption(userID string, req domain.TrackConsumptionRequest) (*domain.TrackResponse, error)
	// ActivateOverride creates a snooze extension when the daily limit has been reached.
	ActivateOverride(userID string) (*domain.FeedLimitOverride, error)
	// GetUsageHistory returns the Nx summary of daily usage.
	GetUsageHistory(userID string, days int) (*domain.UsageHistoryResponse, error)
}

// feedLimitUsecase is the internal implementation.
type feedLimitUsecase struct {
	repo domain.FeedLimitRepository
}

// NewFeedLimitUsecase constructs a FeedLimitUsecase.
func NewFeedLimitUsecase(repo domain.FeedLimitRepository) FeedLimitUsecase {
	return &feedLimitUsecase{repo: repo}
}

// --- Validation constants ---
const (
	maxPostsPerDayMax          = 1000
	maxScrollMinutesMax        = 1440 // 24 hours
	maxOverridesPerDayMax      = 10
	minOverrideDurationMinutes = 5
	maxOverrideDurationMinutes = 60
	maxTrackPostViewCount      = 50
	maxTrackScrollTimeCount    = 30
	defaultHistoryDays         = 7
	maxHistoryDays             = 30
)

// GetSettings returns the user's feed limit configuration (or safe defaults).
func (u *feedLimitUsecase) GetSettings(userID string) (*domain.FeedLimitSetting, error) {
	return u.repo.GetSettings(userID)
}

// UpdateSettings validates the request and upserts the settings.
//
// Business rules:
//   - When is_enabled=true, at least one limit (posts or scroll) must be set.
//   - max_posts_per_day must be 1–1000 if provided.
//   - max_scroll_minutes must be 1–1440 if provided.
//   - max_overrides_per_day must be 0–10 if provided.
//   - override_duration_minutes must be 5–60 if provided.
func (u *feedLimitUsecase) UpdateSettings(userID string, req domain.UpdateFeedLimitRequest) (*domain.FeedLimitSetting, error) {
	// Validate max_posts_per_day
	if req.MaxPostsPerDay != nil {
		if *req.MaxPostsPerDay <= 0 || *req.MaxPostsPerDay > maxPostsPerDayMax {
			return nil, fmt.Errorf("max_posts_per_day must be a positive number (max %d)", maxPostsPerDayMax)
		}
	}
	// Validate max_scroll_minutes
	if req.MaxScrollMinutes != nil {
		if *req.MaxScrollMinutes <= 0 || *req.MaxScrollMinutes > maxScrollMinutesMax {
			return nil, fmt.Errorf("max_scroll_minutes must be between 1 and %d", maxScrollMinutesMax)
		}
	}
	// Validate max_overrides_per_day
	if req.MaxOverridesPerDay != nil {
		if *req.MaxOverridesPerDay < 0 || *req.MaxOverridesPerDay > maxOverridesPerDayMax {
			return nil, fmt.Errorf("max_overrides_per_day must be between 0 and %d", maxOverridesPerDayMax)
		}
	}
	// Validate override_duration_minutes
	if req.OverrideDurationMinutes != nil {
		if *req.OverrideDurationMinutes < minOverrideDurationMinutes || *req.OverrideDurationMinutes > maxOverrideDurationMinutes {
			return nil, fmt.Errorf("override_duration_minutes must be between %d and %d", minOverrideDurationMinutes, maxOverrideDurationMinutes)
		}
	}
	// When enabling, at least one limit must be set
	if req.IsEnabled {
		if req.MaxPostsPerDay == nil && req.MaxScrollMinutes == nil {
			return nil, errors.New("at least one limit (posts or scroll time) must be set when enabling")
		}
	}

	// Merge with existing settings
	existing, err := u.repo.GetSettings(userID)
	if err != nil {
		return nil, err
	}

	existing.IsEnabled = req.IsEnabled
	if req.MaxPostsPerDay != nil {
		existing.MaxPostsPerDay = req.MaxPostsPerDay
	}
	if req.MaxScrollMinutes != nil {
		existing.MaxScrollMinutes = req.MaxScrollMinutes
	}
	if req.MaxOverridesPerDay != nil {
		existing.MaxOverridesPerDay = *req.MaxOverridesPerDay
	}
	if req.OverrideDurationMinutes != nil {
		existing.OverrideDurationMinutes = *req.OverrideDurationMinutes
	}
	existing.UserID = userID

	if err := u.repo.UpsertSettings(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// DisableLimit turns off the feed limit feature while preserving settings.
func (u *feedLimitUsecase) DisableLimit(userID string) error {
	return u.repo.DisableSettings(userID)
}

// GetTodayUsage computes the full usage response including progress and override status.
func (u *feedLimitUsecase) GetTodayUsage(userID string) (*domain.FeedUsageResponse, error) {
	settings, err := u.repo.GetSettings(userID)
	if err != nil {
		return nil, err
	}

	usage, err := u.repo.GetTodayUsage(userID)
	if err != nil {
		return nil, err
	}

	// Calculate progress percentages
	progress := domain.UsageProgress{}
	if settings.MaxPostsPerDay != nil && *settings.MaxPostsPerDay > 0 {
		pct := float64(usage.PostsViewed) / float64(*settings.MaxPostsPerDay) * 100
		progress.PostsPercentage = &pct
	}
	if settings.MaxScrollMinutes != nil && *settings.MaxScrollMinutes > 0 {
		pct := float64(usage.ScrollMinutes) / float64(*settings.MaxScrollMinutes) * 100
		progress.ScrollPercentage = &pct
	}

	// Override status
	activeOverride, err := u.repo.GetActiveOverride(userID)
	if err != nil {
		return nil, err
	}
	overridesUsed, err := u.repo.CountTodayOverrides(userID)
	if err != nil {
		return nil, err
	}
	overrideInfo := domain.OverrideInfo{
		IsActive:           activeOverride != nil,
		OverridesUsedToday: overridesUsed,
		OverridesRemaining: settings.MaxOverridesPerDay - overridesUsed,
	}
	if activeOverride != nil {
		overrideInfo.ExpiresAt = &activeOverride.ExpiresAt
	}

	return &domain.FeedUsageResponse{
		Date:           usage.UsageDate.Format("2006-01-02"),
		PostsViewed:    usage.PostsViewed,
		ScrollMinutes:  usage.ScrollMinutes,
		Limits:         settings,
		Progress:       progress,
		IsLimitReached: usage.IsLimitReached,
		LimitReachedAt: usage.LimitReachedAt,
		Override:       overrideInfo,
	}, nil
}

// TrackConsumption records feed usage and checks if the daily limit has been hit.
//
// Business rules:
//   - EventType must be "post_view" or "scroll_time"
//   - Count must be positive; max 50 for post_view and max 30 for scroll_time
//   - If limit is already reached and no active override exists, reject with limit info
func (u *feedLimitUsecase) TrackConsumption(userID string, req domain.TrackConsumptionRequest) (*domain.TrackResponse, error) {
	// Validate event type
	if req.EventType != "post_view" && req.EventType != "scroll_time" {
		return nil, errors.New("invalid event_type (must be 'post_view' or 'scroll_time')")
	}
	// Validate count
	if req.Count <= 0 {
		return nil, errors.New("count must be a positive number")
	}
	if req.EventType == "post_view" && req.Count > maxTrackPostViewCount {
		return nil, fmt.Errorf("count must not exceed %d for post_view", maxTrackPostViewCount)
	}
	if req.EventType == "scroll_time" && req.Count > maxTrackScrollTimeCount {
		return nil, fmt.Errorf("count must not exceed %d for scroll_time", maxTrackScrollTimeCount)
	}

	settings, err := u.repo.GetSettings(userID)
	if err != nil {
		return nil, err
	}

	// Determine increments
	postsViewed := 0
	scrollMinutes := 0
	if req.EventType == "post_view" {
		postsViewed = req.Count
	} else {
		scrollMinutes = req.Count
	}

	// Atomically increment usage
	usage, err := u.repo.IncrementUsage(userID, postsViewed, scrollMinutes)
	if err != nil {
		return nil, err
	}

	// If feature not enabled, just return current counters — no limit enforced
	if !settings.IsEnabled {
		return &domain.TrackResponse{
			IsLimitReached: false,
			PostsViewed:    usage.PostsViewed,
			ScrollMinutes:  usage.ScrollMinutes,
		}, nil
	}

	// Check if limit is reached after increment
	limitReached := false
	if settings.MaxPostsPerDay != nil && usage.PostsViewed >= *settings.MaxPostsPerDay {
		limitReached = true
	}
	if settings.MaxScrollMinutes != nil && usage.ScrollMinutes >= *settings.MaxScrollMinutes {
		limitReached = true
	}

	// Mark reached in the database if newly hit
	if limitReached && !usage.IsLimitReached {
		_ = u.repo.MarkLimitReached(userID)
	}

	// Build response with remaining quota
	resp := &domain.TrackResponse{
		IsLimitReached: limitReached,
		PostsViewed:    usage.PostsViewed,
		ScrollMinutes:  usage.ScrollMinutes,
	}

	// Calculate remaining
	if settings.MaxPostsPerDay != nil {
		remaining := *settings.MaxPostsPerDay - usage.PostsViewed
		if remaining < 0 {
			remaining = 0
		}
		resp.PostsRemaining = &remaining
	}
	if settings.MaxScrollMinutes != nil {
		remaining := *settings.MaxScrollMinutes - usage.ScrollMinutes
		if remaining < 0 {
			remaining = 0
		}
		resp.ScrollMinutesRemaining = &remaining
	}

	if limitReached {
		overridesUsed, _ := u.repo.CountTodayOverrides(userID)
		overridesRemaining := settings.MaxOverridesPerDay - overridesUsed
		resp.CanOverride = overridesRemaining > 0
		resp.OverridesRemaining = overridesRemaining
		resp.Message = "You've reached your daily feed limit. Take a break! 🧘"
	}

	return resp, nil
}

// ActivateOverride grants the user a temporary snooze when their limit has been reached.
//
// Business rules:
//   - Limit must have actually been reached (is_limit_reached=true)
//   - No override may currently be active
//   - The user must have overrides remaining for today
func (u *feedLimitUsecase) ActivateOverride(userID string) (*domain.FeedLimitOverride, error) {
	settings, err := u.repo.GetSettings(userID)
	if err != nil {
		return nil, err
	}

	usage, err := u.repo.GetTodayUsage(userID)
	if err != nil {
		return nil, err
	}

	if !usage.IsLimitReached {
		return nil, errors.New("limit has not been reached yet")
	}

	activeOverride, err := u.repo.GetActiveOverride(userID)
	if err != nil {
		return nil, err
	}
	if activeOverride != nil {
		return nil, errors.New("an override is already active")
	}

	overridesUsed, err := u.repo.CountTodayOverrides(userID)
	if err != nil {
		return nil, err
	}
	if overridesUsed >= settings.MaxOverridesPerDay {
		return nil, errors.New("no overrides remaining today")
	}

	expiresAt := time.Now().Add(time.Duration(settings.OverrideDurationMinutes) * time.Minute)
	return u.repo.CreateOverride(userID, expiresAt)
}

// GetUsageHistory returns daily usage summaries for the past N days, including averages.
func (u *feedLimitUsecase) GetUsageHistory(userID string, days int) (*domain.UsageHistoryResponse, error) {
	if days <= 0 {
		days = defaultHistoryDays
	}
	if days > maxHistoryDays {
		return nil, fmt.Errorf("days must be between 1 and %d", maxHistoryDays)
	}

	usages, err := u.repo.GetUsageHistory(userID, days)
	if err != nil {
		return nil, err
	}

	history := make([]domain.DailyUsageHistoryItem, 0, len(usages))
	totalPosts := 0
	totalScroll := 0
	daysLimitReached := 0

	for _, u := range usages {
		history = append(history, domain.DailyUsageHistoryItem{
			Date:          u.UsageDate.Format("2006-01-02"),
			PostsViewed:   u.PostsViewed,
			ScrollMinutes: u.ScrollMinutes,
			LimitReached:  u.IsLimitReached,
		})
		totalPosts += u.PostsViewed
		totalScroll += u.ScrollMinutes
		if u.IsLimitReached {
			daysLimitReached++
		}
	}

	n := len(history)
	avgPosts := 0.0
	avgScroll := 0.0
	if n > 0 {
		avgPosts = float64(totalPosts) / float64(n)
		avgScroll = float64(totalScroll) / float64(n)
	}

	return &domain.UsageHistoryResponse{
		History: history,
		Averages: domain.UsageAverages{
			AvgPostsPerDay:   avgPosts,
			AvgScrollMinutes: avgScroll,
			DaysLimitReached: daysLimitReached,
			TotalDays:        n,
		},
	}, nil
}
