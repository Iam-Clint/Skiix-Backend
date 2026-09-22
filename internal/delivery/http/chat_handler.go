package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/skiix-backend/internal/domain"
)

type ChatHandler struct {
	firebaseSvc domain.FirebaseService
}

func NewChatHandler(fs domain.FirebaseService) *ChatHandler {
	return &ChatHandler{firebaseSvc: fs}
}

type AuthTokenResponse struct {
	Token string `json:"token"`
}

// GetAuthToken generates a Firebase Custom Token for the authenticated user
// @Summary Generate Firebase Custom Token
// @Description Generates a custom token that the mobile app can use to sign in to Firebase Realtime Database
// @Tags chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} AuthTokenResponse
// @Failure 401 {object} Response
// @Failure 500 {object} Response
// @Router /chat/auth-token [get]
func (h *ChatHandler) GetAuthToken(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{Message: "Unauthorized"})
		return
	}

	// We don't want to crash or return 500 if Firebase isn't configured locally,
	// but for production it should work.
	if h.firebaseSvc == nil {
		c.JSON(http.StatusInternalServerError, Response{Message: "Firebase service is not initialized"})
		return
	}

	token, err := h.firebaseSvc.GenerateCustomToken(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Message: "Failed to generate chat token"})
		return
	}

	c.JSON(http.StatusOK, AuthTokenResponse{Token: token})
}
