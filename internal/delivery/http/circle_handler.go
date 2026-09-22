package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/skiix-backend/internal/domain"
	"github.com/skiix-backend/internal/usecase"
)

// CircleHandler handles HTTP requests for circle operations.
type CircleHandler struct {
	circleUsecase *usecase.CircleUsecase
}

// NewCircleHandler creates a new CircleHandler.
func NewCircleHandler(circleUsecase *usecase.CircleUsecase) *CircleHandler {
	return &CircleHandler{
		circleUsecase: circleUsecase,
	}
}

// CreateCircle godoc
// @Summary Create a new circle
// @Description Creates a new private circle.
// @Tags circles
// @Accept json
// @Produce json
// @Param request body domain.CreateCircleRequest true "Circle info"
// @Success 201 {object} domain.Circle
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /circles [post]
func (h *CircleHandler) CreateCircle(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req domain.CreateCircleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	circle, err := h.circleUsecase.CreateCircle(userID.(string), req)
	if err != nil {
		if err.Error() == "maximum circles limit reached (max 20)" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, circle)
}

// GetCircles godoc
// @Summary Get user circles
// @Description Returns a list of all circles the current user is a member of.
// @Tags circles
// @Produce json
// @Success 200 {array} domain.Circle
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /circles [get]
func (h *CircleHandler) GetCircles(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	circles, err := h.circleUsecase.GetCirclesByUser(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, circles)
}

// UpdateCircle godoc
// @Summary Update an existing circle
// @Description Updates the name/description of a circle (owner only).
// @Tags circles
// @Accept json
// @Produce json
// @Param id path string true "Circle ID"
// @Param request body domain.UpdateCircleRequest true "Circle info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /circles/{id} [put]
func (h *CircleHandler) UpdateCircle(c *gin.Context) {
	circleID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req domain.UpdateCircleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.circleUsecase.UpdateCircle(circleID, userID.(string), req)
	if err != nil {
		if err.Error() == "only the circle owner can perform this action" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// DeleteCircle godoc
// @Summary Delete a circle
// @Description Deletes a circle (owner only).
// @Tags circles
// @Produce json
// @Param id path string true "Circle ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /circles/{id} [delete]
func (h *CircleHandler) DeleteCircle(c *gin.Context) {
	circleID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := h.circleUsecase.DeleteCircle(circleID, userID.(string))
	if err != nil {
		if err.Error() == "only the circle owner can perform this action" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// AddMembers godoc
// @Summary Add members to a circle
// @Description Adds multiple users to a circle (owner only).
// @Tags circles
// @Accept json
// @Produce json
// @Param id path string true "Circle ID"
// @Param request body domain.AddMembersRequest true "Array of user IDs"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /circles/{id}/members [post]
func (h *CircleHandler) AddMembers(c *gin.Context) {
	circleID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req domain.AddMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.circleUsecase.AddMembers(circleID, userID.(string), req.UserIDs)
	if err != nil {
		if err.Error() == "only the circle owner can perform this action" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		// Treat limits as Bad Request
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// RemoveMember godoc
// @Summary Remove a member from a circle
// @Description Owner can remove any member, member can remove themselves.
// @Tags circles
// @Produce json
// @Param id path string true "Circle ID"
// @Param user_id path string true "User ID to remove"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /circles/{id}/members/{user_id} [delete]
func (h *CircleHandler) RemoveMember(c *gin.Context) {
	circleID := c.Param("id")
	targetUserID := c.Param("user_id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := h.circleUsecase.RemoveMember(circleID, userID.(string), targetUserID)
	if err != nil {
		if err.Error() == "owner cannot be removed, must delete circle instead" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "only the circle owner can perform this action" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// GetMembers godoc
// @Summary Get members of a circle
// @Description Returns paginated members of a circle (must be a member).
// @Tags circles
// @Produce json
// @Param id path string true "Circle ID"
// @Param limit query int false "Pagination limit"
// @Param offset query int false "Pagination offset"
// @Success 200 {array} domain.CircleMember
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /circles/{id}/members [get]
func (h *CircleHandler) GetMembers(c *gin.Context) {
	circleID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	members, err := h.circleUsecase.GetMembers(circleID, userID.(string), limit, offset)
	if err != nil {
		if err.Error() == "you are not a member of this circle" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, members)
}
