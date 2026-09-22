package domain

import "time"

// Post represents a user's post in the feed.
// Each post belongs to one user and can have multiple categories,
// likes, and comments.
type Post struct {
	ID           string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID       string    `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Type         string    `json:"type" example:"text"` // text | reel | story | project
	Content      string    `json:"content" example:"Hello world"`
	ImageURL     *string   `json:"image_url,omitempty" example:"https://cdn.skiix.com/image.jpg"` // legacy alias
	MediaURL     *string   `json:"media_url,omitempty" example:"https://cdn.skiix.com/media.mp4"`
	IsProject    bool      `json:"is_project"`                  // true → "Join Project" CTA vs "Follow"
	ProjectID    *string   `json:"project_id,omitempty"`        // when is_project=true, defaults to post id
	Visibility   string    `json:"visibility" example:"public"` // public, circle
	CircleID     *string   `json:"circle_id,omitempty" example:"550e8400-e29b-..."`
	LikeCount    int       `json:"like_count" example:"5"`
	CommentCount int       `json:"comment_count" example:"2"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PostCategory represents a category tag assigned to a post.
// Uses a junction table pattern to support multi-category tagging
// (e.g., a post can be "news" + "sports" simultaneously).
type PostCategory struct {
	ID           string    `json:"id"`
	PostID       string    `json:"post_id"`
	CategoryName string    `json:"category_name"`
	CreatedAt    time.Time `json:"created_at"`
}

// PostLike records that a specific user liked a specific post.
// The UNIQUE(user_id, post_id) constraint prevents duplicate likes.
type PostLike struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	PostID    string    `json:"post_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Comment represents a user's comment on a post.
type Comment struct {
	ID        string    `json:"id"`
	PostID    string    `json:"post_id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PostFeedItem is a DTO enriching a Post with the author's email
// and the current user's like status for use in feed responses.
type PostFeedItem struct {
	Post
	AuthorEmail string `json:"author_email"`
	IsLiked     bool   `json:"is_liked"`
	// Type is duplicated at top-level for backwards compatibility with old clients.
	// It matches Post.Type.
	Type        string `json:"type"` // "project" | "text" | "reel" | "story"
	IsJoined    bool   `json:"is_joined"`
	IsFollowing bool   `json:"is_following"`
}

// ProjectSummary represents a selectable project root in the Project Update flow.
type ProjectSummary struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// CreatePostRequest is the payload for creating a new post.
type CreatePostRequest struct {
	Type       string   `json:"type"`
	Content    string   `json:"content" binding:"required"`
	ImageURL   *string  `json:"image_url"`
	MediaURL   *string  `json:"media_url"`
	IsProject  bool     `json:"is_project"`
	ProjectID  *string  `json:"project_id"`
	Visibility string   `json:"visibility" binding:"required,oneof=public circle"`
	CircleID   *string  `json:"circle_id"`
	Categories []string `json:"categories"`
}

// UpdatePostRequest is the payload for editing an existing post.
type UpdatePostRequest struct {
	Content  string  `json:"content" binding:"required"`
	ImageURL *string `json:"image_url"`
}

// FeedQuery contains pagination and filtering parameters for feed requests.
type FeedQuery struct {
	Limit  int
	Offset int
	UserID string // The requesting user (for is_liked computation)
}

// PostRepository defines the data access contract for posts.
type PostRepository interface {
	// Create inserts a new post into the database.
	Create(post *Post) error

	// GetByID retrieves a single post by its ID.
	GetByID(postID string) (*Post, error)

	// GetFeed retrieves a paginated list of posts for the feed.
	// Returns PostFeedItems enriched with author info and like status.
	GetFeed(query FeedQuery) ([]PostFeedItem, error)

	// GetByUserID retrieves all posts by a specific user.
	GetByUserID(userID string, limit, offset int) ([]PostFeedItem, error)

	// GetByCircle retrieves all posts in a specific circle.
	GetByCircle(circleID string, userID string, limit, offset int) ([]PostFeedItem, error)

	// GetProjectRoots returns "project" posts that represent project roots for a user.
	// A project root is a project-type post with project_id = id (or legacy is_project=true and project_id=id).
	GetProjectRoots(userID string, limit int) ([]ProjectSummary, error)

	// Update modifies an existing post's content.
	Update(post *Post) error

	// Delete removes a post by ID. Only the owner can delete.
	Delete(postID string) error

	// SetCategories replaces all categories for a post atomically.
	SetCategories(postID string, categories []string) error

	// GetCategories returns all category names for a post.
	GetCategories(postID string) ([]string, error)

	// ToggleLike adds a like if not exists, removes if exists (idempotent toggle).
	// Returns the new like status (true = liked, false = unliked).
	ToggleLike(userID, postID string) (bool, error)

	// IncrementCommentCount atomically increments the comment_count on a post.
	IncrementCommentCount(postID string) error

	// DecrementCommentCount atomically decrements the comment_count on a post.
	DecrementCommentCount(postID string) error
}

// CommentRepository defines the data access contract for comments.
type CommentRepository interface {
	// Create inserts a new comment.
	Create(comment *Comment) error

	// GetByPostID retrieves all comments for a post, paginated.
	GetByPostID(postID string, limit, offset int) ([]Comment, error)

	// Delete removes a comment by ID.
	Delete(commentID string) error

	// GetByID retrieves a single comment by ID.
	GetByID(commentID string) (*Comment, error)
}
