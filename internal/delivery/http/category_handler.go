package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/skiix-backend/internal/domain"
)

type CategoriesResponse struct {
	Categories []string `json:"categories"`
}

// GetCategories returns the list of valid post categories.
// @Summary      Get valid post categories
// @Description  Returns the predefined list of categories that can be assigned to posts. Use these values for filtering and post creation.
// @Tags         categories
// @Produce      json
// @Success      200  {object}  CategoriesResponse
// @Router       /categories [get]
func GetCategories(c *gin.Context) {
	c.JSON(http.StatusOK, CategoriesResponse{
		Categories: domain.ValidCategories,
	})
}
