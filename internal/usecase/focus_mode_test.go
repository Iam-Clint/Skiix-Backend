package usecase

import (
	"errors"
	"strings"
	"testing"

	"github.com/skiix-backend/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Repository
// ============================================================================

// MockFocusModeRepository is a testify mock implementing domain.FocusModeRepository.
// It allows us to test the usecase layer in isolation, without a real database.
type MockFocusModeRepository struct {
	mock.Mock
}

func (m *MockFocusModeRepository) GetSettings(userID string) (*domain.FocusModeSetting, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.FocusModeSetting), args.Error(1)
}

func (m *MockFocusModeRepository) UpsertSettings(userID string, isEnabled bool) error {
	args := m.Called(userID, isEnabled)
	return args.Error(0)
}

func (m *MockFocusModeRepository) GetBlockedCategories(userID string) ([]string, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockFocusModeRepository) SetBlockedCategories(userID string, categories []string) error {
	args := m.Called(userID, categories)
	return args.Error(0)
}

// ============================================================================
// GetStatus Tests
// ============================================================================

func TestGetStatus_NewUser_ReturnsDefaults(t *testing.T) {
	// Scenario: User has never configured focus mode.
	// Expected: Return is_enabled=false, blocked_categories=[]
	mockRepo := new(MockFocusModeRepository)
	uc := NewFocusModeUsecase(mockRepo)

	mockRepo.On("GetSettings", "user-123").Return(nil, nil)
	mockRepo.On("GetBlockedCategories", "user-123").Return([]string{}, nil)

	status, err := uc.GetStatus("user-123")

	assert.NoError(t, err)
	assert.False(t, status.IsEnabled)
	assert.Empty(t, status.BlockedCategories)
	assert.NotNil(t, status.BlockedCategories) // Must be [] not nil
	mockRepo.AssertExpectations(t)
}

func TestGetStatus_ExistingUser_ReturnsSettings(t *testing.T) {
	// Scenario: User has configured focus mode with blocked categories.
	mockRepo := new(MockFocusModeRepository)
	uc := NewFocusModeUsecase(mockRepo)

	mockRepo.On("GetSettings", "user-123").Return(&domain.FocusModeSetting{
		ID:        "setting-1",
		UserID:    "user-123",
		IsEnabled: true,
	}, nil)
	mockRepo.On("GetBlockedCategories", "user-123").Return(
		[]string{"entertainment", "news"}, nil,
	)

	status, err := uc.GetStatus("user-123")

	assert.NoError(t, err)
	assert.True(t, status.IsEnabled)
	assert.Equal(t, []string{"entertainment", "news"}, status.BlockedCategories)
	mockRepo.AssertExpectations(t)
}

func TestGetStatus_EmptyUserID_ReturnsError(t *testing.T) {
	mockRepo := new(MockFocusModeRepository)
	uc := NewFocusModeUsecase(mockRepo)

	status, err := uc.GetStatus("")

	assert.Nil(t, status)
	assert.EqualError(t, err, "user id is required")
}

func TestGetStatus_RepositoryError_PropagatesError(t *testing.T) {
	mockRepo := new(MockFocusModeRepository)
	uc := NewFocusModeUsecase(mockRepo)

	mockRepo.On("GetSettings", "user-123").Return(nil, errors.New("db connection lost"))

	status, err := uc.GetStatus("user-123")

	assert.Nil(t, status)
	assert.EqualError(t, err, "db connection lost")
}

// ============================================================================
// Toggle Tests
// ============================================================================

func TestToggle_EnableFocusMode(t *testing.T) {
	mockRepo := new(MockFocusModeRepository)
	uc := NewFocusModeUsecase(mockRepo)

	mockRepo.On("UpsertSettings", "user-123", true).Return(nil)

	err := uc.Toggle("user-123", true)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestToggle_DisableFocusMode(t *testing.T) {
	mockRepo := new(MockFocusModeRepository)
	uc := NewFocusModeUsecase(mockRepo)

	mockRepo.On("UpsertSettings", "user-123", false).Return(nil)

	err := uc.Toggle("user-123", false)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestToggle_EmptyUserID_ReturnsError(t *testing.T) {
	mockRepo := new(MockFocusModeRepository)
	uc := NewFocusModeUsecase(mockRepo)

	err := uc.Toggle("", true)

	assert.EqualError(t, err, "user id is required")
}

func TestToggle_RepositoryError_PropagatesError(t *testing.T) {
	mockRepo := new(MockFocusModeRepository)
	uc := NewFocusModeUsecase(mockRepo)

	mockRepo.On("UpsertSettings", "user-123", true).Return(errors.New("db error"))

	err := uc.Toggle("user-123", true)

	assert.EqualError(t, err, "db error")
}

// ============================================================================
// SetBlockedCategories Tests
// ============================================================================

func TestSetBlockedCategories_ValidCategories(t *testing.T) {
	mockRepo := new(MockFocusModeRepository)
	uc := NewFocusModeUsecase(mockRepo)

	categories := []string{"entertainment", "news", "memes"}
	mockRepo.On("SetBlockedCategories", "user-123", categories).Return(nil)

	err := uc.SetBlockedCategories("user-123", categories)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSetBlockedCategories_EmptyArray_ClearsAll(t *testing.T) {
	// Sending [] should delete all blocked categories (reset)
	mockRepo := new(MockFocusModeRepository)
	uc := NewFocusModeUsecase(mockRepo)

	mockRepo.On("SetBlockedCategories", "user-123", []string{}).Return(nil)

	err := uc.SetBlockedCategories("user-123", []string{})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSetBlockedCategories_DuplicatesRemoved(t *testing.T) {
	// Input: ["news", "news", "memes", "memes"]
	// Expected to repo: ["news", "memes"]
	mockRepo := new(MockFocusModeRepository)
	uc := NewFocusModeUsecase(mockRepo)

	mockRepo.On("SetBlockedCategories", "user-123", []string{"news", "memes"}).Return(nil)

	err := uc.SetBlockedCategories("user-123", []string{"news", "news", "memes", "memes"})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSetBlockedCategories_TrimsWhitespace(t *testing.T) {
	// Input: ["  news  ", " memes "]
	// Expected to repo: ["news", "memes"]
	mockRepo := new(MockFocusModeRepository)
	uc := NewFocusModeUsecase(mockRepo)

	mockRepo.On("SetBlockedCategories", "user-123", []string{"news", "memes"}).Return(nil)

	err := uc.SetBlockedCategories("user-123", []string{"  news  ", " memes "})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSetBlockedCategories_TooMany_ReturnsError(t *testing.T) {
	mockRepo := new(MockFocusModeRepository)
	uc := NewFocusModeUsecase(mockRepo)

	// Generate 21 unique categories (exceeds limit of 20)
	categories := make([]string, 21)
	for i := range categories {
		categories[i] = "category-" + strings.Repeat("x", i+1)
	}

	err := uc.SetBlockedCategories("user-123", categories)

	assert.EqualError(t, err, "too many categories (max 20)")
}

func TestSetBlockedCategories_NameTooLong_ReturnsError(t *testing.T) {
	mockRepo := new(MockFocusModeRepository)
	uc := NewFocusModeUsecase(mockRepo)

	longName := strings.Repeat("a", 101)
	err := uc.SetBlockedCategories("user-123", []string{longName})

	assert.EqualError(t, err, "category name too long (max 100 characters)")
}

func TestSetBlockedCategories_EmptyUserID_ReturnsError(t *testing.T) {
	mockRepo := new(MockFocusModeRepository)
	uc := NewFocusModeUsecase(mockRepo)

	err := uc.SetBlockedCategories("", []string{"news"})

	assert.EqualError(t, err, "user id is required")
}

func TestSetBlockedCategories_WhitespaceOnlyCategories_Ignored(t *testing.T) {
	// Input contains only whitespace strings — they should be ignored
	mockRepo := new(MockFocusModeRepository)
	uc := NewFocusModeUsecase(mockRepo)

	mockRepo.On("SetBlockedCategories", "user-123", []string{}).Return(nil)

	err := uc.SetBlockedCategories("user-123", []string{"  ", "   ", ""})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
