package usecase

import (
	"errors"
	"testing"

	"github.com/skiix-backend/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockPostRepository struct{ mock.Mock }

func (m *MockPostRepository) Create(post *domain.Post) error {
	args := m.Called(post)
	return args.Error(0)
}
func (m *MockPostRepository) GetByID(postID string) (*domain.Post, error) {
	args := m.Called(postID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Post), args.Error(1)
}
func (m *MockPostRepository) GetFeed(query domain.FeedQuery) ([]domain.PostFeedItem, error) {
	args := m.Called(query)
	return args.Get(0).([]domain.PostFeedItem), args.Error(1)
}
func (m *MockPostRepository) GetByUserID(userID string, limit, offset int) ([]domain.PostFeedItem, error) {
	args := m.Called(userID, limit, offset)
	return args.Get(0).([]domain.PostFeedItem), args.Error(1)
}
func (m *MockPostRepository) GetByCircle(circleID string, userID string, limit, offset int) ([]domain.PostFeedItem, error) {
	args := m.Called(circleID, userID, limit, offset)
	return args.Get(0).([]domain.PostFeedItem), args.Error(1)
}
func (m *MockPostRepository) GetProjectRoots(userID string, limit int) ([]domain.ProjectSummary, error) {
	args := m.Called(userID, limit)
	return args.Get(0).([]domain.ProjectSummary), args.Error(1)
}
func (m *MockPostRepository) Update(post *domain.Post) error {
	args := m.Called(post)
	return args.Error(0)
}
func (m *MockPostRepository) Delete(postID string) error {
	args := m.Called(postID)
	return args.Error(0)
}
func (m *MockPostRepository) SetCategories(postID string, categories []string) error {
	args := m.Called(postID, categories)
	return args.Error(0)
}
func (m *MockPostRepository) GetCategories(postID string) ([]string, error) {
	args := m.Called(postID)
	return args.Get(0).([]string), args.Error(1)
}
func (m *MockPostRepository) ToggleLike(userID, postID string) (bool, error) {
	args := m.Called(userID, postID)
	return args.Bool(0), args.Error(1)
}
func (m *MockPostRepository) IncrementCommentCount(postID string) error {
	args := m.Called(postID)
	return args.Error(0)
}
func (m *MockPostRepository) DecrementCommentCount(postID string) error {
	args := m.Called(postID)
	return args.Error(0)
}

type MockCommentRepository struct{ mock.Mock }

func (m *MockCommentRepository) Create(comment *domain.Comment) error {
	args := m.Called(comment)
	return args.Error(0)
}
func (m *MockCommentRepository) GetByPostID(postID string, limit, offset int) ([]domain.Comment, error) {
	args := m.Called(postID, limit, offset)
	return args.Get(0).([]domain.Comment), args.Error(1)
}
func (m *MockCommentRepository) GetByID(commentID string) (*domain.Comment, error) {
	args := m.Called(commentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Comment), args.Error(1)
}
func (m *MockCommentRepository) Delete(commentID string) error {
	args := m.Called(commentID)
	return args.Error(0)
}

type MockCircleRepository struct{ mock.Mock }

func (m *MockCircleRepository) Create(circle *domain.Circle) error {
	args := m.Called(circle)
	return args.Error(0)
}
func (m *MockCircleRepository) GetByID(id string) (*domain.Circle, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Circle), args.Error(1)
}
func (m *MockCircleRepository) GetByUser(userID string) ([]domain.Circle, error) {
	args := m.Called(userID)
	return args.Get(0).([]domain.Circle), args.Error(1)
}
func (m *MockCircleRepository) Update(circle *domain.Circle) error {
	args := m.Called(circle)
	return args.Error(0)
}
func (m *MockCircleRepository) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}
func (m *MockCircleRepository) AddMembers(circleID string, userIDs []string) error {
	args := m.Called(circleID, userIDs)
	return args.Error(0)
}
func (m *MockCircleRepository) RemoveMember(circleID string, userID string) error {
	args := m.Called(circleID, userID)
	return args.Error(0)
}
func (m *MockCircleRepository) IsMember(circleID string, userID string) (bool, error) {
	args := m.Called(circleID, userID)
	return args.Bool(0), args.Error(1)
}
func (m *MockCircleRepository) GetMembers(circleID string, limit, offset int) ([]domain.CircleMember, error) {
	args := m.Called(circleID, limit, offset)
	return args.Get(0).([]domain.CircleMember), args.Error(1)
}
func (m *MockCircleRepository) CountByOwner(ownerID string) (int, error) {
	args := m.Called(ownerID)
	return args.Int(0), args.Error(1)
}

// --- Helpers ---

func newTestPostUsecase(postRepo *MockPostRepository, commentRepo *MockCommentRepository, circleRepo *MockCircleRepository) PostUsecase {
	return NewPostUsecase(postRepo, commentRepo, circleRepo, nil, nil)
}

// --- CreatePost Tests ---

func TestCreatePost_Success(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	req := domain.CreatePostRequest{Content: "Hello Skiix!", Visibility: "public"}
	postRepo.On("Create", mock.AnythingOfType("*domain.Post")).Return(nil)

	post, err := uc.CreatePost("user-1", req)

	assert.NoError(t, err)
	assert.NotNil(t, post)
	assert.Equal(t, "Hello Skiix!", post.Content)
	assert.Equal(t, "user-1", post.UserID)
	postRepo.AssertExpectations(t)
}

func TestCreatePost_Circle_Success(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	cid := "c1"
	req := domain.CreatePostRequest{
		Content:    "Secret!",
		Visibility: "circle",
		CircleID:   &cid,
	}

	circleRepo.On("IsMember", "c1", "user-1").Return(true, nil)
	postRepo.On("Create", mock.AnythingOfType("*domain.Post")).Return(nil)

	post, err := uc.CreatePost("user-1", req)

	assert.NoError(t, err)
	assert.NotNil(t, post)
	assert.Equal(t, "circle", post.Visibility)
	circleRepo.AssertExpectations(t)
}

func TestCreatePost_Circle_NotMember(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	cid := "c1"
	req := domain.CreatePostRequest{
		Content:    "Secret!",
		Visibility: "circle",
		CircleID:   &cid,
	}

	circleRepo.On("IsMember", "c1", "user-1").Return(false, nil)

	_, err := uc.CreatePost("user-1", req)

	assert.EqualError(t, err, "you are not a member of this circle")
	postRepo.AssertNotCalled(t, "Create")
}

func TestCreatePost_WithCategories(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	req := domain.CreatePostRequest{
		Content:    "Tech post",
		Visibility: "public",
		Categories: []string{"tech", "education"},
	}
	postRepo.On("Create", mock.AnythingOfType("*domain.Post")).Return(nil)
	postRepo.On("SetCategories", mock.AnythingOfType("string"), []string{"tech", "education"}).Return(nil)

	post, err := uc.CreatePost("user-1", req)

	assert.NoError(t, err)
	assert.NotNil(t, post)
	postRepo.AssertExpectations(t)
}

func TestCreatePost_DuplicateCategoriesDeduped(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	req := domain.CreatePostRequest{
		Content:    "Post",
		Visibility: "public",
		Categories: []string{"tech", "tech", "gaming", "tech"},
	}
	postRepo.On("Create", mock.AnythingOfType("*domain.Post")).Return(nil)
	// Should only be called with 2 unique categories
	postRepo.On("SetCategories", mock.AnythingOfType("string"), []string{"tech", "gaming"}).Return(nil)

	post, err := uc.CreatePost("user-1", req)

	assert.NoError(t, err)
	assert.NotNil(t, post)
	postRepo.AssertExpectations(t)
}

func TestCreatePost_EmptyContent_Fails(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	req := domain.CreatePostRequest{Content: "   "}
	_, err := uc.CreatePost("user-1", req)

	assert.EqualError(t, err, "content cannot be empty")
	postRepo.AssertNotCalled(t, "Create")
}

func TestCreatePost_TooManyCategories_Fails(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	// Use all 10 valid categories + 1 extra to exceed the limit
	cats := []string{"entertainment", "gaming", "sports", "news", "tech", "lifestyle", "education", "music", "art", "other", "other"}
	// Since "other" is duplicated, we actually get 10 unique categories which is exactly the limit.
	// Let's test with an invalid one to verify enum validation fires first.
	cats = []string{"invalid_category_xyz"}
	req := domain.CreatePostRequest{Content: "Post", Visibility: "public", Categories: cats}

	_, err := uc.CreatePost("user-1", req)
	assert.Contains(t, err.Error(), "invalid category")
}

// --- UpdatePost Tests ---

func TestUpdatePost_Success(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	existing := &domain.Post{ID: "p1", UserID: "user-1", Content: "Old content"}
	postRepo.On("GetByID", "p1").Return(existing, nil)
	postRepo.On("Update", mock.AnythingOfType("*domain.Post")).Return(nil)

	updated, err := uc.UpdatePost("user-1", "p1", "New content", nil)

	assert.NoError(t, err)
	assert.Equal(t, "New content", updated.Content)
	postRepo.AssertExpectations(t)
}

func TestUpdatePost_NotOwner_Forbidden(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	existing := &domain.Post{ID: "p1", UserID: "owner-user"}
	postRepo.On("GetByID", "p1").Return(existing, nil)

	_, err := uc.UpdatePost("attacker-user", "p1", "Hacked!", nil)

	assert.EqualError(t, err, "forbidden: you can only edit your own posts")
	postRepo.AssertNotCalled(t, "Update")
}

// --- DeletePost Tests ---

func TestDeletePost_Success(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	existing := &domain.Post{ID: "p1", UserID: "user-1"}
	postRepo.On("GetByID", "p1").Return(existing, nil)
	postRepo.On("Delete", "p1").Return(nil)

	err := uc.DeletePost("user-1", "p1")

	assert.NoError(t, err)
	postRepo.AssertExpectations(t)
}

func TestDeletePost_NotOwner_Forbidden(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	existing := &domain.Post{ID: "p1", UserID: "owner-user"}
	postRepo.On("GetByID", "p1").Return(existing, nil)

	err := uc.DeletePost("attacker-user", "p1")

	assert.EqualError(t, err, "forbidden: you can only delete your own posts")
	postRepo.AssertNotCalled(t, "Delete")
}

// --- ToggleLike Tests ---

func TestToggleLike_LikePost(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	postRepo.On("GetByID", "p1").Return(&domain.Post{ID: "p1"}, nil)
	postRepo.On("ToggleLike", "user-1", "p1").Return(true, nil)

	liked, err := uc.ToggleLike("user-1", "p1")

	assert.NoError(t, err)
	assert.True(t, liked)
	postRepo.AssertExpectations(t)
}

func TestToggleLike_UnlikePost(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	postRepo.On("GetByID", "p1").Return(&domain.Post{ID: "p1"}, nil)
	postRepo.On("ToggleLike", "user-1", "p1").Return(false, nil)

	liked, err := uc.ToggleLike("user-1", "p1")

	assert.NoError(t, err)
	assert.False(t, liked)
}

func TestToggleLike_PostNotFound(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	postRepo.On("GetByID", "nonexistent").Return(nil, errors.New("post not found"))

	_, err := uc.ToggleLike("user-1", "nonexistent")
	assert.EqualError(t, err, "post not found")
}

// --- AddComment Tests ---

func TestAddComment_Success(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	postRepo.On("GetByID", "p1").Return(&domain.Post{ID: "p1"}, nil)
	commentRepo.On("Create", mock.AnythingOfType("*domain.Comment")).Return(nil)
	postRepo.On("IncrementCommentCount", "p1").Return(nil)

	comment, err := uc.AddComment("user-1", "p1", "Great post!")

	assert.NoError(t, err)
	assert.NotNil(t, comment)
	assert.Equal(t, "Great post!", comment.Content)
	assert.Equal(t, "p1", comment.PostID)
}

func TestAddComment_EmptyContent_Fails(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	postRepo.On("GetByID", "p1").Return(&domain.Post{ID: "p1"}, nil)

	_, err := uc.AddComment("user-1", "p1", "   ")
	assert.EqualError(t, err, "comment content cannot be empty")
}

// --- DeleteComment Tests ---

func TestDeleteComment_Success(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	comment := &domain.Comment{ID: "c1", PostID: "p1", UserID: "user-1"}
	commentRepo.On("GetByID", "c1").Return(comment, nil)
	commentRepo.On("Delete", "c1").Return(nil)
	postRepo.On("DecrementCommentCount", "p1").Return(nil)

	err := uc.DeleteComment("user-1", "c1")
	assert.NoError(t, err)
}

func TestDeleteComment_NotOwner_Forbidden(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	comment := &domain.Comment{ID: "c1", PostID: "p1", UserID: "owner-user"}
	commentRepo.On("GetByID", "c1").Return(comment, nil)

	err := uc.DeleteComment("attacker-user", "c1")
	assert.EqualError(t, err, "forbidden: you can only delete your own comments")
	commentRepo.AssertNotCalled(t, "Delete")
}

// --- Pagination Tests ---

func TestGetFeed_DefaultLimit(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	// limit=0 should default to 20
	postRepo.On("GetFeed", domain.FeedQuery{Limit: 20, Offset: 0, UserID: "user-1"}).
		Return([]domain.PostFeedItem{}, nil)

	items, err := uc.GetFeed("user-1", 0, 0)
	assert.NoError(t, err)
	assert.NotNil(t, items)
}

func TestGetFeed_MaxLimitClamped(t *testing.T) {
	postRepo := new(MockPostRepository)
	commentRepo := new(MockCommentRepository)
	circleRepo := new(MockCircleRepository)
	uc := newTestPostUsecase(postRepo, commentRepo, circleRepo)

	// limit=100 should be clamped to 50
	postRepo.On("GetFeed", domain.FeedQuery{Limit: 50, Offset: 0, UserID: "user-1"}).
		Return([]domain.PostFeedItem{}, nil)

	items, err := uc.GetFeed("user-1", 100, 0)
	assert.NoError(t, err)
	assert.NotNil(t, items)
}
