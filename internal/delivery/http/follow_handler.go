package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skiix-backend/internal/domain"
)

type FollowHandler struct {
	users   domain.UserRepository
	follows domain.FollowerRepository
	members domain.ProjectMemberRepository
	posts   domain.PostRepository
}

func NewFollowHandler(users domain.UserRepository, follows domain.FollowerRepository, members domain.ProjectMemberRepository, posts domain.PostRepository) *FollowHandler {
	return &FollowHandler{users: users, follows: follows, members: members, posts: posts}
}

type joinProjectBody struct {
	UserID    string `json:"user_id"`
	UserEmail string `json:"user_email"`
	ProjectID string `json:"project_id"`
}

// POST /api/v1/projects/join
func (h *FollowHandler) JoinProject(c *gin.Context) {
	var req joinProjectBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	req.UserID = strings.TrimSpace(req.UserID)
	req.UserEmail = strings.TrimSpace(strings.ToLower(req.UserEmail))
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	if req.ProjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id required"})
		return
	}
	userID := req.UserID
	if userID == "" {
		if req.UserEmail == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id or user_email required"})
			return
		}
		u, err := h.users.FindByEmail(req.UserEmail)
		if err != nil || u == nil {
			// Allow join from public UI: create user row if missing.
			now := time.Now().UTC()
			id := uuid.New().String()
			u2 := &domain.User{
				ID:            id,
				Email:         req.UserEmail,
				EmailVerified: true,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			if err := h.users.Create(u2); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "user not found"})
				return
			}
			userID = id
		} else {
			userID = u.ID
		}
	}

	already, err := h.members.Join(userID, req.ProjectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "joined": true, "already": already})
}

type followBody struct {
	FollowerID     string `json:"follower_id"`
	FollowingID    string `json:"following_id"`
	FollowerEmail  string `json:"follower_email"`
	FollowingEmail string `json:"following_email"`
}

// POST /api/v1/follow
func (h *FollowHandler) FollowUser(c *gin.Context) {
	var req followBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	req.FollowerID = strings.TrimSpace(req.FollowerID)
	req.FollowingID = strings.TrimSpace(req.FollowingID)
	req.FollowerEmail = strings.TrimSpace(strings.ToLower(req.FollowerEmail))
	req.FollowingEmail = strings.TrimSpace(strings.ToLower(req.FollowingEmail))

	followerID := req.FollowerID
	followingID := req.FollowingID

	ensureByEmail := func(email string) (string, error) {
		if email == "" {
			return "", nil
		}
		u, err := h.users.FindByEmail(email)
		if err == nil && u != nil {
			return u.ID, nil
		}
		now := time.Now().UTC()
		id := uuid.New().String()
		u2 := &domain.User{
			ID:            id,
			Email:         email,
			EmailVerified: true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := h.users.Create(u2); err != nil {
			return "", err
		}
		return id, nil
	}

	if followerID == "" {
		id, err := ensureByEmail(req.FollowerEmail)
		if err != nil || id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "follower_id or follower_email required"})
			return
		}
		followerID = id
	}
	if followingID == "" {
		id, err := ensureByEmail(req.FollowingEmail)
		if err != nil || id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "following_id or following_email required"})
			return
		}
		followingID = id
	}

	already, err := h.follows.Follow(followerID, followingID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "following": true, "already": already})
}

// GET /api/v1/projects?viewer_email=...
func (h *FollowHandler) ListProjects(c *gin.Context) {
	email := strings.TrimSpace(strings.ToLower(c.Query("viewer_email")))
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "viewer_email required"})
		return
	}
	u, err := h.users.FindByEmail(email)
	if err != nil || u == nil {
		c.JSON(http.StatusOK, gin.H{"data": []domain.ProjectSummary{}})
		return
	}
	items, err := h.posts.GetProjectRoots(u.ID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list projects"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}
