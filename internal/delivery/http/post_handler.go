package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skiix-backend/internal/domain"
	"github.com/skiix-backend/internal/usecase"
)

// PostHandler handles HTTP requests for the posts/feed feature.
type PostHandler struct {
	postUsecase usecase.PostUsecase
	users       domain.UserRepository
}

// NewPostHandler creates a new PostHandler.
func NewPostHandler(postUsecase usecase.PostUsecase, users domain.UserRepository) *PostHandler {
	return &PostHandler{postUsecase: postUsecase, users: users}
}

// CreatePost godoc
// @Summary      Create a new post
// @Description  Create a new post with optional image and categories
// @Tags         posts
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        request  body      domain.CreatePostRequest  true  "Post content"
// @Success      201      {object}  domain.Post
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Router       /api/v1/posts [post]
func (h *PostHandler) CreatePost(c *gin.Context) {
	userID := getUserIDFromContext(c)

	var req domain.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	post, err := h.postUsecase.CreatePost(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, post)
}

// GetFeed godoc
// @Summary      Get feed
// @Description  Get paginated feed of all posts
// @Tags         posts
// @Produce      json
// @Security     Bearer
// @Param        limit   query     int  false  "Number of posts to return (max 50, default 20)"
// @Param        offset  query     int  false  "Number of posts to skip"
// @Success      200     {array}   domain.PostFeedItem
// @Failure      401     {object}  map[string]string
// @Router       /api/v1/feed [get]
func (h *PostHandler) GetFeed(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		// Public feed: allow passing viewer_email to compute follow/join states.
		if viewerEmail := strings.TrimSpace(strings.ToLower(c.Query("viewer_email"))); viewerEmail != "" && h.users != nil {
			if u, err := h.users.FindByEmail(viewerEmail); err == nil && u != nil {
				userID = u.ID
			}
		}
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, err := h.postUsecase.GetFeed(userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   items,
		"limit":  limit,
		"offset": offset,
	})
}

// GetUserPosts godoc
// @Summary      Get posts by user
// @Description  Get all posts by a specific user
// @Tags         posts
// @Produce      json
// @Security     Bearer
// @Param        user_id  path      string  true   "User ID"
// @Param        limit    query     int     false  "Limit"
// @Param        offset   query     int     false  "Offset"
// @Success      200      {array}   domain.PostFeedItem
// @Failure      401      {object}  map[string]string
// @Router       /api/v1/users/{user_id}/posts [get]
func (h *PostHandler) GetUserPosts(c *gin.Context) {
	userID := c.Param("user_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, err := h.postUsecase.GetUserPosts(userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   items,
		"limit":  limit,
		"offset": offset,
	})
}

// GetPost godoc
// @Summary      Get a single post
// @Description  Get a post by its ID
// @Tags         posts
// @Produce      json
// @Security     Bearer
// @Param        post_id  path      string  true  "Post ID"
// @Success      200      {object}  domain.Post
// @Failure      404      {object}  map[string]string
// @Router       /api/v1/posts/{post_id} [get]
func (h *PostHandler) GetPost(c *gin.Context) {
	postID := c.Param("post_id")

	post, err := h.postUsecase.GetPost(postID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	c.JSON(http.StatusOK, post)
}

// UpdatePost godoc
// @Summary      Update a post
// @Description  Update post content. Only the post owner can update.
// @Tags         posts
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        post_id  path      string  true  "Post ID"
// @Param        request  body      domain.CreatePostRequest  true  "Updated content"
// @Success      200      {object}  domain.Post
// @Failure      400      {object}  map[string]string
// @Failure      403      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Router       /api/v1/posts/{post_id} [put]
func (h *PostHandler) UpdatePost(c *gin.Context) {
	userID := getUserIDFromContext(c)
	postID := c.Param("post_id")

	var req domain.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	post, err := h.postUsecase.UpdatePost(userID, postID, req.Content, req.ImageURL)
	if err != nil {
		switch err.Error() {
		case "post not found":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "forbidden: you can only edit your own posts":
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, post)
}

// DeletePost godoc
// @Summary      Delete a post
// @Description  Delete a post. Only the post owner can delete.
// @Tags         posts
// @Produce      json
// @Security     Bearer
// @Param        post_id  path      string  true  "Post ID"
// @Success      200      {object}  map[string]string
// @Failure      403      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Router       /api/v1/posts/{post_id} [delete]
func (h *PostHandler) DeletePost(c *gin.Context) {
	userID := getUserIDFromContext(c)
	postID := c.Param("post_id")

	if err := h.postUsecase.DeletePost(userID, postID); err != nil {
		switch err.Error() {
		case "post not found":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "forbidden: you can only delete your own posts":
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "post deleted successfully"})
}

// ToggleLike godoc
// @Summary      Toggle like on a post
// @Description  Like a post if not liked, unlike if already liked
// @Tags         posts
// @Produce      json
// @Security     Bearer
// @Param        post_id  path      string  true  "Post ID"
// @Success      200      {object}  map[string]interface{}
// @Failure      404      {object}  map[string]string
// @Router       /api/v1/posts/{post_id}/like [post]
func (h *PostHandler) ToggleLike(c *gin.Context) {
	userID := getUserIDFromContext(c)
	postID := c.Param("post_id")

	liked, err := h.postUsecase.ToggleLike(userID, postID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	message := "post liked"
	if !liked {
		message = "post unliked"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  message,
		"is_liked": liked,
	})
}

// AddComment godoc
// @Summary      Add a comment to a post
// @Description  Create a comment on a specific post
// @Tags         comments
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        post_id  path      string  true  "Post ID"
// @Param        request  body      map[string]string  true  "Comment content"
// @Success      201      {object}  domain.Comment
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Router       /api/v1/posts/{post_id}/comments [post]
func (h *PostHandler) AddComment(c *gin.Context) {
	userID := getUserIDFromContext(c)
	postID := c.Param("post_id")

	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	comment, err := h.postUsecase.AddComment(userID, postID, req.Content)
	if err != nil {
		switch err.Error() {
		case "post not found":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, comment)
}

// GetComments godoc
// @Summary      Get comments on a post
// @Description  Get paginated comments for a specific post
// @Tags         comments
// @Produce      json
// @Security     Bearer
// @Param        post_id  path      string  true   "Post ID"
// @Param        limit    query     int     false  "Limit"
// @Param        offset   query     int     false  "Offset"
// @Success      200      {array}   domain.Comment
// @Failure      404      {object}  map[string]string
// @Router       /api/v1/posts/{post_id}/comments [get]
func (h *PostHandler) GetComments(c *gin.Context) {
	postID := c.Param("post_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	comments, err := h.postUsecase.GetComments(postID, limit, offset)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   comments,
		"limit":  limit,
		"offset": offset,
	})
}

// DeleteComment godoc
// @Summary      Delete a comment
// @Description  Delete a comment. Only the comment owner can delete.
// @Tags         comments
// @Produce      json
// @Security     Bearer
// @Param        post_id     path      string  true  "Post ID"
// @Param        comment_id  path      string  true  "Comment ID"
// @Success      200         {object}  map[string]string
// @Failure      403         {object}  map[string]string
// @Failure      404         {object}  map[string]string
// @Router       /api/v1/posts/{post_id}/comments/{comment_id} [delete]
func (h *PostHandler) DeleteComment(c *gin.Context) {
	userID := getUserIDFromContext(c)
	commentID := c.Param("comment_id")

	if err := h.postUsecase.DeleteComment(userID, commentID); err != nil {
		switch err.Error() {
		case "comment not found":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "forbidden: you can only delete your own comments":
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "comment deleted successfully"})
}

// getUserIDFromContext extracts the authenticated user's ID from the Gin context.
// This is set by JWTMiddleware after token validation.
func getUserIDFromContext(c *gin.Context) string {
	userID, ok := c.Get("user_id")
	if !ok || userID == nil {
		return ""
	}
	if s, ok := userID.(string); ok {
		return s
	}
	return ""
}

// GetCircleFeed godoc
// @Summary      Get circle feed
// @Description  Get paginated feed of posts within a specific circle
// @Tags         posts,circles
// @Produce      json
// @Security     Bearer
// @Param        id      path      string  true   "Circle ID"
// @Param        limit   query     int     false  "Limit"
// @Param        offset  query     int     false  "Offset"
// @Success      200     {array}   domain.PostFeedItem
// @Failure      401     {object}  map[string]string
// @Failure      403     {object}  map[string]string
// @Router       /api/v1/circles/{id}/posts [get]
func (h *PostHandler) GetCircleFeed(c *gin.Context) {
	userID := getUserIDFromContext(c)
	circleID := c.Param("id")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, err := h.postUsecase.GetCircleFeed(circleID, userID, limit, offset)
	if err != nil {
		if err.Error() == "403 Forbidden" || err.Error() == "you are not a member of this circle" {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   items,
		"limit":  limit,
		"offset": offset,
	})
}
