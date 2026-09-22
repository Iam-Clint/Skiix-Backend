package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/skiix-backend/internal/domain"
	"github.com/skiix-backend/internal/usecase"
)

// FeedLimitHandler handles HTTP requests for the daily feed limit feature.
type FeedLimitHandler struct {
	uc usecase.FeedLimitUsecase
}

// NewFeedLimitHandler creates a new FeedLimitHandler.
func NewFeedLimitHandler(uc usecase.FeedLimitUsecase) *FeedLimitHandler {
	return &FeedLimitHandler{uc: uc}
}

// GetSettings godoc
// @Summary      Get feed limit settings
// @Description  Retrieves the current daily feed limit configuration. Returns defaults if not configured.
// @Tags         feed-limit
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  domain.FeedLimitSetting
// @Failure      401  {object}  map[string]string
// @Router       /api/v1/feed-limit [get]
func (h *FeedLimitHandler) GetSettings(c *gin.Context) {
	userID := getUserIDFromContext(c)

	settings, err := h.uc.GetSettings(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateSettings godoc
// @Summary      Update feed limit settings
// @Description  Creates or updates the user's daily feed limit configuration.
// @Tags         feed-limit
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        request  body      domain.UpdateFeedLimitRequest  true  "Feed limit settings"
// @Success      200      {object}  domain.FeedLimitSetting
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Router       /api/v1/feed-limit [put]
func (h *FeedLimitHandler) UpdateSettings(c *gin.Context) {
	userID := getUserIDFromContext(c)

	var req domain.UpdateFeedLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	settings, err := h.uc.UpdateSettings(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// DisableLimit godoc
// @Summary      Disable feed limit
// @Description  Disables the daily feed limit. Settings are preserved but the feature is turned off.
// @Tags         feed-limit
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/feed-limit [delete]
func (h *FeedLimitHandler) DisableLimit(c *gin.Context) {
	userID := getUserIDFromContext(c)

	if err := h.uc.DisableLimit(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "feed limit disabled", "is_enabled": false})
}

// GetTodayUsage godoc
// @Summary      Get today's feed usage
// @Description  Returns the user's feed consumption stats for today versus their configured limits.
// @Tags         feed-limit
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  domain.FeedUsageResponse
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/feed-limit/usage [get]
func (h *FeedLimitHandler) GetTodayUsage(c *gin.Context) {
	userID := getUserIDFromContext(c)

	resp, err := h.uc.GetTodayUsage(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// TrackConsumption godoc
// @Summary      Track feed consumption
// @Description  Reports a feed consumption event (post view or scroll time). Returns current limit status.
// @Tags         feed-limit
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        request  body      domain.TrackConsumptionRequest  true  "Consumption event"
// @Success      200      {object}  domain.TrackResponse
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Router       /api/v1/feed-limit/track [post]
func (h *FeedLimitHandler) TrackConsumption(c *gin.Context) {
	userID := getUserIDFromContext(c)

	var req domain.TrackConsumptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	resp, err := h.uc.TrackConsumption(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ActivateOverride godoc
// @Summary      Activate feed limit override (snooze)
// @Description  Grants temporary extended access when the daily limit has been reached.
// @Tags         feed-limit
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  domain.FeedLimitOverride
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Router       /api/v1/feed-limit/override [post]
func (h *FeedLimitHandler) ActivateOverride(c *gin.Context) {
	userID := getUserIDFromContext(c)

	override, err := h.uc.ActivateOverride(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, override)
}

// GetUsageHistory godoc
// @Summary      Get feed usage history
// @Description  Returns daily feed usage summaries for the past N days (default 7, max 30).
// @Tags         feed-limit
// @Produce      json
// @Security     Bearer
// @Param        days  query     int  false  "Number of days to retrieve (1-30)"
// @Success      200   {object}  domain.UsageHistoryResponse
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Router       /api/v1/feed-limit/history [get]
func (h *FeedLimitHandler) GetUsageHistory(c *gin.Context) {
	userID := getUserIDFromContext(c)
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	resp, err := h.uc.GetUsageHistory(userID, days)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
