package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/markbates/goth/gothic"
	"github.com/skiix-backend/internal/usecase"
)

type OAuthHandler struct {
	authUsecase usecase.AuthUsecase
}

func NewOAuthHandler(authUsecase usecase.AuthUsecase) *OAuthHandler {
	return &OAuthHandler{
		authUsecase: authUsecase,
	}
}

// GoogleLogin godoc
// @Summary      Initiate Google OAuth login
// @Description  Redirects to Google OAuth login page
// @Tags         auth
// @Produce      json
// @Success      302 "Redirect to Google login"
// @Router       /auth/google [get]
func (h *OAuthHandler) GoogleLogin(c *gin.Context) {
	c.Request.URL.RawQuery = c.Request.URL.RawQuery + "&provider=google"
	gothic.BeginAuthHandler(c.Writer, c.Request)
}

func (h *OAuthHandler) GoogleCallback(c *gin.Context) {
	user, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "failed to complete google authentication"})
		return
	}

	tokens, u, err := h.authUsecase.OAuthLogin(user.Email, "google")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, toLoginResponse(tokens, u))
}
