package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skiix-backend/internal/domain"
)

type SocialHandler struct {
	stories domain.StoryRepository
	conns   domain.ConnectionRepository
}

func NewSocialHandler(stories domain.StoryRepository, conns domain.ConnectionRepository) *SocialHandler {
	return &SocialHandler{stories: stories, conns: conns}
}

type createStoryRequest struct {
	AuthorEmail string `json:"author_email"`
	MediaURL    string `json:"media_url"`
	MediaType   string `json:"media_type"` // image|video
	Caption     string `json:"caption"`
}

type storyResponse struct {
	ID          string  `json:"id"`
	AuthorEmail string  `json:"author_email"`
	MediaURL    string  `json:"media_url"`
	MediaType   string  `json:"media_type"`
	Caption     *string `json:"caption"`
	CreatedAt   string  `json:"created_at"`
	ExpiresAt   string  `json:"expires_at"`
}

func (h *SocialHandler) CreateStory(c *gin.Context) {
	var req createStoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	req.AuthorEmail = strings.TrimSpace(strings.ToLower(req.AuthorEmail))
	req.MediaURL = strings.TrimSpace(req.MediaURL)
	req.MediaType = strings.TrimSpace(strings.ToLower(req.MediaType))

	if req.AuthorEmail == "" || req.MediaURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "author_email and media_url required"})
		return
	}
	if req.MediaType != "image" && req.MediaType != "video" {
		req.MediaType = "image"
	}

	now := time.Now().UTC()
	exp := now.Add(24 * time.Hour)
	var captionPtr *string
	if strings.TrimSpace(req.Caption) != "" {
		s := strings.TrimSpace(req.Caption)
		captionPtr = &s
	}

	story := &domain.Story{
		ID:          uuid.New().String(),
		AuthorEmail: req.AuthorEmail,
		MediaURL:    req.MediaURL,
		MediaType:   req.MediaType,
		Caption:     captionPtr,
		CreatedAt:   now,
		ExpiresAt:   exp,
	}
	if err := h.stories.Create(story); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create story"})
		return
	}

	c.JSON(http.StatusOK, storyResponse{
		ID:          story.ID,
		AuthorEmail: story.AuthorEmail,
		MediaURL:    story.MediaURL,
		MediaType:   story.MediaType,
		Caption:     story.Caption,
		CreatedAt:   story.CreatedAt.Format(time.RFC3339),
		ExpiresAt:   story.ExpiresAt.Format(time.RFC3339),
	})
}

func (h *SocialHandler) GetStoryFeed(c *gin.Context) {
	email := strings.TrimSpace(strings.ToLower(c.Query("email")))
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}

	items, err := h.stories.Feed(email, 40)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load stories"})
		return
	}

	out := make([]storyResponse, 0, len(items))
	for _, s := range items {
		s2 := s
		out = append(out, storyResponse{
			ID:          s2.ID,
			AuthorEmail: s2.AuthorEmail,
			MediaURL:    s2.MediaURL,
			MediaType:   s2.MediaType,
			Caption:     s2.Caption,
			CreatedAt:   s2.CreatedAt.Format(time.RFC3339),
			ExpiresAt:   s2.ExpiresAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": out})
}

type requestConnectionBody struct {
	FromEmail string `json:"from_email"`
	ToEmail   string `json:"to_email"`
}

type connectionResponse struct {
	ID        string `json:"id"`
	FromEmail string `json:"from_email"`
	ToEmail   string `json:"to_email"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (h *SocialHandler) RequestConnection(c *gin.Context) {
	var req requestConnectionBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	conn, err := h.conns.CreateRequest(req.FromEmail, req.ToEmail)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": connectionResponse{
		ID:        conn.ID,
		FromEmail: conn.FromEmail,
		ToEmail:   conn.ToEmail,
		Status:    conn.Status,
		CreatedAt: conn.CreatedAt.Format(time.RFC3339),
		UpdatedAt: conn.UpdatedAt.Format(time.RFC3339),
	}})
}

type acceptConnectionBody struct {
	ID string `json:"id"`
}

func (h *SocialHandler) AcceptConnection(c *gin.Context) {
	var req acceptConnectionBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.conns.Accept(req.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to accept"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *SocialHandler) ListConnections(c *gin.Context) {
	email := strings.TrimSpace(strings.ToLower(c.Query("email")))
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}
	items, err := h.conns.List(email, 50, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load connections"})
		return
	}
	out := make([]connectionResponse, 0, len(items))
	for _, it := range items {
		c2 := it
		out = append(out, connectionResponse{
			ID:        c2.ID,
			FromEmail: c2.FromEmail,
			ToEmail:   c2.ToEmail,
			Status:    c2.Status,
			CreatedAt: c2.CreatedAt.Format(time.RFC3339),
			UpdatedAt: c2.UpdatedAt.Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

type removeConnectionBody struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (h *SocialHandler) RemoveConnection(c *gin.Context) {
	var req removeConnectionBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.ID = strings.TrimSpace(req.ID)
	if req.Email == "" || req.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and id required"})
		return
	}
	if err := h.conns.Delete(req.ID, req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
