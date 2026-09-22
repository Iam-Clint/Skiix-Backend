package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/skiix-backend/internal/domain"
	"github.com/skiix-backend/internal/usecase"
)

// FocusModeHandler handles HTTP requests for the Focus Mode feature.
type FocusModeHandler struct {
	focusModeUsecase usecase.FocusModeUsecase
}

// NewFocusModeHandler creates a new handler with the given usecase.
func NewFocusModeHandler(focusModeUsecase usecase.FocusModeUsecase) *FocusModeHandler {
	return &FocusModeHandler{
		focusModeUsecase: focusModeUsecase,
	}
}

// ToggleRequest represents the JSON body for enabling/disabling focus mode.
type ToggleRequest struct {
	IsEnabled bool `json:"is_enabled" example:"true"`
}

// ToggleResponse represents the response after toggling focus mode.
type ToggleResponse struct {
	Message   string `json:"message" example:"focus mode updated"`
	IsEnabled bool   `json:"is_enabled" example:"true"`
}

// SetCategoriesRequest represents the JSON body for setting blocked categories.
type SetCategoriesRequest struct {
	Categories []string `json:"categories" example:"entertainment,news,memes"`
}

// SetCategoriesResponse represents the response after updating blocked categories.
type SetCategoriesResponse struct {
	Message           string   `json:"message" example:"blocked categories updated"`
	BlockedCategories []string `json:"blocked_categories" example:"entertainment,news,memes"`
}

// GetStatus godoc
// @Summary      Get focus mode status
// @Description  Retrieve the current focus mode status and blocked categories for the authenticated user
// @Tags         focus-mode
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  domain.FocusModeStatus  "Focus mode status retrieved"
// @Failure      401  {object}  ErrorResponse            "Unauthorized"
// @Failure      500  {object}  ErrorResponse            "Internal server error"
// @Router       /focus-mode [get]
func (h *FocusModeHandler) GetStatus(c *gin.Context) {
	userPayload, exists := c.Get(UserContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	tokenPayload, ok := userPayload.(*domain.TokenPayload)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	status, err := h.focusModeUsecase.GetStatus(tokenPayload.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// Toggle godoc
// @Summary      Toggle focus mode
// @Description  Enable or disable focus mode for the authenticated user
// @Tags         focus-mode
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        request  body      ToggleRequest   true  "Toggle payload"
// @Success      200      {object}  ToggleResponse  "Focus mode toggled"
// @Failure      400      {object}  ErrorResponse   "Invalid input"
// @Failure      401      {object}  ErrorResponse   "Unauthorized"
// @Failure      500      {object}  ErrorResponse   "Internal server error"
// @Router       /focus-mode [put]
func (h *FocusModeHandler) Toggle(c *gin.Context) {
	userPayload, exists := c.Get(UserContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	tokenPayload, ok := userPayload.(*domain.TokenPayload)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	var req ToggleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if err := h.focusModeUsecase.Toggle(tokenPayload.UserID, req.IsEnabled); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}

	c.JSON(http.StatusOK, ToggleResponse{
		Message:   "focus mode updated",
		IsEnabled: req.IsEnabled,
	})
}

// SetBlockedCategories godoc
// @Summary      Set blocked categories
// @Description  Replace the list of blocked content categories for the authenticated user
// @Tags         focus-mode
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        request  body      SetCategoriesRequest   true  "Categories payload"
// @Success      200      {object}  SetCategoriesResponse  "Categories updated"
// @Failure      400      {object}  ErrorResponse          "Validation error"
// @Failure      401      {object}  ErrorResponse          "Unauthorized"
// @Failure      500      {object}  ErrorResponse          "Internal server error"
// @Router       /focus-mode/categories [put]
func (h *FocusModeHandler) SetBlockedCategories(c *gin.Context) {
	userPayload, exists := c.Get(UserContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	tokenPayload, ok := userPayload.(*domain.TokenPayload)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	var req SetCategoriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	// Ensure categories is not nil (treat null as empty array)
	if req.Categories == nil {
		req.Categories = make([]string, 0)
	}

	if err := h.focusModeUsecase.SetBlockedCategories(tokenPayload.UserID, req.Categories); err != nil {
		// Business validation errors from usecase return 400
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SetCategoriesResponse{
		Message:           "blocked categories updated",
		BlockedCategories: req.Categories,
	})
}
