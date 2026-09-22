package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/skiix-backend/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock ---

type MockFeedLimitRepository struct{ mock.Mock }

func (m *MockFeedLimitRepository) GetSettings(userID string) (*domain.FeedLimitSetting, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.FeedLimitSetting), args.Error(1)
}
func (m *MockFeedLimitRepository) UpsertSettings(s *domain.FeedLimitSetting) error {
	args := m.Called(s)
	return args.Error(0)
}
func (m *MockFeedLimitRepository) DisableSettings(userID string) error {
	args := m.Called(userID)
	return args.Error(0)
}
func (m *MockFeedLimitRepository) GetTodayUsage(userID string) (*domain.FeedUsageDaily, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.FeedUsageDaily), args.Error(1)
}
func (m *MockFeedLimitRepository) IncrementUsage(userID string, postsViewed, scrollMinutes int) (*domain.FeedUsageDaily, error) {
	args := m.Called(userID, postsViewed, scrollMinutes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.FeedUsageDaily), args.Error(1)
}
func (m *MockFeedLimitRepository) MarkLimitReached(userID string) error {
	args := m.Called(userID)
	return args.Error(0)
}
func (m *MockFeedLimitRepository) GetUsageHistory(userID string, days int) ([]domain.FeedUsageDaily, error) {
	args := m.Called(userID, days)
	return args.Get(0).([]domain.FeedUsageDaily), args.Error(1)
}
func (m *MockFeedLimitRepository) CreateOverride(userID string, expiresAt time.Time) (*domain.FeedLimitOverride, error) {
	args := m.Called(userID, expiresAt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.FeedLimitOverride), args.Error(1)
}
func (m *MockFeedLimitRepository) GetActiveOverride(userID string) (*domain.FeedLimitOverride, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.FeedLimitOverride), args.Error(1)
}
func (m *MockFeedLimitRepository) CountTodayOverrides(userID string) (int, error) {
	args := m.Called(userID)
	return args.Int(0), args.Error(1)
}

// --- Settings Tests ---

func TestUpdateSettings_Success_EnableWithPosts(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	max := 50
	existing := &domain.FeedLimitSetting{UserID: "u1", MaxOverridesPerDay: 3, OverrideDurationMinutes: 15}
	repo.On("GetSettings", "u1").Return(existing, nil)
	repo.On("UpsertSettings", mock.AnythingOfType("*domain.FeedLimitSetting")).Return(nil)

	req := domain.UpdateFeedLimitRequest{IsEnabled: true, MaxPostsPerDay: &max}
	s, err := uc.UpdateSettings("u1", req)

	assert.NoError(t, err)
	assert.True(t, s.IsEnabled)
	assert.Equal(t, 50, *s.MaxPostsPerDay)
}

func TestUpdateSettings_EnableWithNoLimit_Fails(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	existing := &domain.FeedLimitSetting{UserID: "u1", MaxOverridesPerDay: 3, OverrideDurationMinutes: 15}
	repo.On("GetSettings", "u1").Return(existing, nil)

	// Enable without providing any limit
	req := domain.UpdateFeedLimitRequest{IsEnabled: true}
	_, err := uc.UpdateSettings("u1", req)

	assert.EqualError(t, err, "at least one limit (posts or scroll time) must be set when enabling")
}

func TestUpdateSettings_MaxPostsInvalid_Fails(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	max := 9999
	req := domain.UpdateFeedLimitRequest{IsEnabled: true, MaxPostsPerDay: &max}
	_, err := uc.UpdateSettings("u1", req)

	assert.Contains(t, err.Error(), "max_posts_per_day must be a positive number")
}

func TestUpdateSettings_MaxScrollInvalid_Fails(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	max := 9999
	req := domain.UpdateFeedLimitRequest{IsEnabled: true, MaxScrollMinutes: &max}
	_, err := uc.UpdateSettings("u1", req)

	assert.Contains(t, err.Error(), "max_scroll_minutes must be between 1 and")
}

func TestDisableLimit_Success(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	repo.On("DisableSettings", "u1").Return(nil)
	err := uc.DisableLimit("u1")
	assert.NoError(t, err)
}

// --- Track Consumption Tests ---

func TestTrackConsumption_InvalidEventType(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	req := domain.TrackConsumptionRequest{EventType: "swipe", Count: 5}
	_, err := uc.TrackConsumption("u1", req)

	assert.EqualError(t, err, "invalid event_type (must be 'post_view' or 'scroll_time')")
}

func TestTrackConsumption_ZeroCount_Fails(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	req := domain.TrackConsumptionRequest{EventType: "post_view", Count: 0}
	_, err := uc.TrackConsumption("u1", req)

	assert.EqualError(t, err, "count must be a positive number")
}

func TestTrackConsumption_UnderLimit_Success(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	maxPosts := 50
	settings := &domain.FeedLimitSetting{IsEnabled: true, MaxPostsPerDay: &maxPosts, MaxOverridesPerDay: 3, OverrideDurationMinutes: 15}

	repo.On("GetSettings", "u1").Return(settings, nil)
	repo.On("IncrementUsage", "u1", 5, 0).Return(&domain.FeedUsageDaily{PostsViewed: 15, ScrollMinutes: 0}, nil)

	req := domain.TrackConsumptionRequest{EventType: "post_view", Count: 5}
	resp, err := uc.TrackConsumption("u1", req)

	assert.NoError(t, err)
	assert.False(t, resp.IsLimitReached)
	assert.Equal(t, 15, resp.PostsViewed)
}

func TestTrackConsumption_LimitReached_Success(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	maxPosts := 50
	settings := &domain.FeedLimitSetting{IsEnabled: true, MaxPostsPerDay: &maxPosts, MaxOverridesPerDay: 3}
	afterUsage := &domain.FeedUsageDaily{PostsViewed: 52, ScrollMinutes: 0, IsLimitReached: false}

	repo.On("GetSettings", "u1").Return(settings, nil)
	repo.On("IncrementUsage", "u1", 5, 0).Return(afterUsage, nil)
	repo.On("MarkLimitReached", "u1").Return(nil)
	repo.On("CountTodayOverrides", "u1").Return(1, nil)

	req := domain.TrackConsumptionRequest{EventType: "post_view", Count: 5}
	resp, err := uc.TrackConsumption("u1", req)

	assert.NoError(t, err)
	assert.True(t, resp.IsLimitReached)
	assert.True(t, resp.CanOverride)
	assert.Contains(t, resp.Message, "Take a break")
}

func TestTrackConsumption_FeatureDisabled_StillCountsSilently(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	settings := &domain.FeedLimitSetting{IsEnabled: false}
	after := &domain.FeedUsageDaily{PostsViewed: 5}

	repo.On("GetSettings", "u1").Return(settings, nil)
	repo.On("IncrementUsage", "u1", 5, 0).Return(after, nil)

	req := domain.TrackConsumptionRequest{EventType: "post_view", Count: 5}
	resp, err := uc.TrackConsumption("u1", req)

	assert.NoError(t, err)
	assert.False(t, resp.IsLimitReached)
}

// --- Override Tests ---

func TestActivateOverride_Success(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	settings := &domain.FeedLimitSetting{MaxOverridesPerDay: 3, OverrideDurationMinutes: 15}
	usage := &domain.FeedUsageDaily{IsLimitReached: true}
	override := &domain.FeedLimitOverride{ID: "o1"}

	repo.On("GetSettings", "u1").Return(settings, nil)
	repo.On("GetTodayUsage", "u1").Return(usage, nil)
	repo.On("GetActiveOverride", "u1").Return(nil, nil)
	repo.On("CountTodayOverrides", "u1").Return(1, nil)
	repo.On("CreateOverride", "u1", mock.AnythingOfType("time.Time")).Return(override, nil)

	o, err := uc.ActivateOverride("u1")
	assert.NoError(t, err)
	assert.NotNil(t, o)
}

func TestActivateOverride_LimitNotReached_Fails(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	settings := &domain.FeedLimitSetting{MaxOverridesPerDay: 3}
	usage := &domain.FeedUsageDaily{IsLimitReached: false}

	repo.On("GetSettings", "u1").Return(settings, nil)
	repo.On("GetTodayUsage", "u1").Return(usage, nil)

	_, err := uc.ActivateOverride("u1")
	assert.EqualError(t, err, "limit has not been reached yet")
}

func TestActivateOverride_AlreadyActive_Fails(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	settings := &domain.FeedLimitSetting{MaxOverridesPerDay: 3}
	usage := &domain.FeedUsageDaily{IsLimitReached: true}
	activeOverride := &domain.FeedLimitOverride{ID: "o1"}

	repo.On("GetSettings", "u1").Return(settings, nil)
	repo.On("GetTodayUsage", "u1").Return(usage, nil)
	repo.On("GetActiveOverride", "u1").Return(activeOverride, nil)

	_, err := uc.ActivateOverride("u1")
	assert.EqualError(t, err, "an override is already active")
}

func TestActivateOverride_NoOverridesRemaining_Fails(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	settings := &domain.FeedLimitSetting{MaxOverridesPerDay: 3}
	usage := &domain.FeedUsageDaily{IsLimitReached: true}

	repo.On("GetSettings", "u1").Return(settings, nil)
	repo.On("GetTodayUsage", "u1").Return(usage, nil)
	repo.On("GetActiveOverride", "u1").Return(nil, nil)
	repo.On("CountTodayOverrides", "u1").Return(3, nil) // All used

	_, err := uc.ActivateOverride("u1")
	assert.EqualError(t, err, "no overrides remaining today")
}

// --- History Tests ---

func TestGetUsageHistory_DefaultDaysSuccess(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	now := time.Now()
	days := []domain.FeedUsageDaily{
		{UsageDate: now, PostsViewed: 20, ScrollMinutes: 10, IsLimitReached: false},
		{UsageDate: now.AddDate(0, 0, -1), PostsViewed: 55, ScrollMinutes: 35, IsLimitReached: true},
	}
	repo.On("GetUsageHistory", "u1", 7).Return(days, nil)

	resp, err := uc.GetUsageHistory("u1", 7)
	assert.NoError(t, err)
	assert.Len(t, resp.History, 2)
	assert.Equal(t, 1, resp.Averages.DaysLimitReached)
}

func TestGetUsageHistory_ExceedMaxDays_Fails(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	_, err := uc.GetUsageHistory("u1", 100)
	assert.Contains(t, err.Error(), "days must be between 1 and")
}

func TestGetUsageHistory_RepoError(t *testing.T) {
	repo := new(MockFeedLimitRepository)
	uc := NewFeedLimitUsecase(repo)

	repo.On("GetUsageHistory", "u1", 7).Return([]domain.FeedUsageDaily{}, errors.New("db error"))

	_, err := uc.GetUsageHistory("u1", 7)
	assert.Error(t, err)
}
