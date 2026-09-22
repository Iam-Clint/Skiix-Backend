package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skiix-backend/internal/domain"
	"github.com/skiix-backend/internal/usecase"
)

type PublicPostsHandler struct {
	users domain.UserRepository
	posts usecase.PostUsecase
}

func NewPublicPostsHandler(users domain.UserRepository, posts usecase.PostUsecase) *PublicPostsHandler {
	return &PublicPostsHandler{users: users, posts: posts}
}

type createPublicPostRequest struct {
	AuthorEmail string  `json:"author_email"`
	Type        string  `json:"type"`
	Content     string  `json:"content"`
	ImageURL    *string `json:"image_url"`
	MediaURL    *string `json:"media_url"`
	IsProject   bool    `json:"is_project"`
	ProjectID   *string `json:"project_id"`
}

// CreatePublicPost creates a post for a given email.
// It auto-creates a matching row in the backend `users` table if missing.
func (h *PublicPostsHandler) CreatePublicPost(c *gin.Context) {
	var req createPublicPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.AuthorEmail))
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "author_email required"})
		return
	}

	u, err := h.users.FindByEmail(email)
	if err != nil {
		// Auto-create a minimal user row
		provider := "supabase"
		u = &domain.User{
			ID:            uuid.New().String(),
			Email:         email,
			Password:      nil,
			Provider:      &provider,
			EmailVerified: true,
		}
		if err2 := h.users.Create(u); err2 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}
	}

	p, err := h.posts.CreatePost(u.ID, domain.CreatePostRequest{
		Type:       req.Type,
		Content:    req.Content,
		ImageURL:   req.ImageURL,
		MediaURL:   req.MediaURL,
		IsProject:  req.IsProject,
		ProjectID:  req.ProjectID,
		Visibility: "public",
		CircleID:   nil,
		Categories: nil,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": p})
}
