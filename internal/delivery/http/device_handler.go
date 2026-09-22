package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/skiix-backend/internal/domain"
)

type DeviceHandler struct {
	deviceUsecase domain.DeviceUsecase
}

func NewDeviceHandler(du domain.DeviceUsecase) *DeviceHandler {
	return &DeviceHandler{deviceUsecase: du}
}

type RegisterDeviceRequest struct {
	FCMToken   string `json:"fcm_token" binding:"required"`
	DeviceType string `json:"device_type"` // e.g. "android", "ios"
}

type Response struct {
	Message string `json:"message"`
}

// RegisterToken handles storing an FCM token for push notifications
// @Summary Register FCM Device Token
// @Description Register a mobile device FCM token for the authenticated user to receive push notifications
// @Tags devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RegisterDeviceRequest true "Device Token Data"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Router /devices/register [post]
func (h *DeviceHandler) RegisterToken(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{Message: "Unauthorized"})
		return
	}

	var req RegisterDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Message: "Invalid input"})
		return
	}

	if req.DeviceType == "" {
		req.DeviceType = "unknown"
	}

	err := h.deviceUsecase.RegisterDeviceToken(userID.(string), req.FCMToken, req.DeviceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Message: "Failed to register device token"})
		return
	}

	c.JSON(http.StatusOK, Response{Message: "Device token registered successfully"})
}

// UnregisterToken removes an FCM token
// @Summary Unregister FCM Device Token
// @Description Remove an FCM token (e.g., when logging out)
// @Tags devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fcm_token query string true "FCM Token to remove"
// @Success 200 {object} Response
// @Router /devices/unregister [delete]
func (h *DeviceHandler) UnregisterToken(c *gin.Context) {
	fcmToken := c.Query("fcm_token")
	if fcmToken == "" {
		c.JSON(http.StatusBadRequest, Response{Message: "fcm_token is required"})
		return
	}

	err := h.deviceUsecase.UnregisterDeviceToken(fcmToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Message: "Failed to unregister device token"})
		return
	}

	c.JSON(http.StatusOK, Response{Message: "Device token unregistered successfully"})
}
