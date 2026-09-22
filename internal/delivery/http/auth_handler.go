package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skiix-backend/internal/domain"
	"github.com/skiix-backend/internal/usecase"
)

type AuthHandler struct {
	authUsecase usecase.AuthUsecase
}

func NewAuthHandler(authUsecase usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{
		authUsecase: authUsecase,
	}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"password123"`
}

type AuthResponse struct {
	Message string `json:"message,omitempty" example:"User registered successfully"`
}

type LoginUser struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	Provider      *string `json:"provider,omitempty"`
	EmailVerified bool    `json:"email_verified"`
}

type LoginResponse struct {
	AccessToken  string     `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	RefreshToken string     `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	ExpiresIn    int        `json:"expires_in" example:"900"`
	ExpiresAt    int64      `json:"expires_at" example:"1714147200000"`
	User         *LoginUser `json:"user,omitempty"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"invalid email format"`
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account with email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      RegisterRequest  true  "Register credentials"
// @Success      201      {object}  AuthResponse     "User registered successfully"
// @Failure      400      {object}  ErrorResponse    "Invalid input"
// @Failure      409      {object}  ErrorResponse    "Email already exists"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.authUsecase.Register(req.Email, req.Password); err != nil {
		c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{Message: "User registered successfully"})
}

type LoginRequest struct {
	Email      string `json:"email" example:"user@example.com"`
	Identifier string `json:"identifier" example:"user@example.com"`
	Password   string `json:"password" binding:"required" example:"password123"`
}

func (r *LoginRequest) resolveEmail() string {
	if r.Email != "" {
		return r.Email
	}
	return r.Identifier
}

// Login godoc
// @Summary      Login user
// @Description  Login with email and password to get access and refresh tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest     true  "Login credentials"
// @Success      200      {object}  LoginResponse    "Login successful with tokens"
// @Failure      400      {object}  ErrorResponse    "Invalid input"
// @Failure      401      {object}  ErrorResponse    "Invalid credentials"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	email := req.resolveEmail()
	if email == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "email or identifier is required"})
		return
	}

	tokens, user, err := h.authUsecase.Login(email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrEmailNotVerified) {
			c.JSON(http.StatusForbidden, ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, toLoginResponse(tokens, user))
}

func toLoginResponse(tokens *domain.AuthTokens, u *domain.User) LoginResponse {
	if tokens == nil {
		return LoginResponse{}
	}
	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second).UnixMilli()
	lu := &LoginUser{
		ID:            u.ID,
		Email:         u.Email,
		Provider:      u.Provider,
		EmailVerified: u.EmailVerified,
	}
	return LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		ExpiresAt:    expiresAt,
		User:         lu,
	}
}

// RefreshToken godoc
// @Summary      Refresh access token
// @Description  Exchange a valid refresh token for a new access token and refresh token pair
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      RefreshRequest   true  "Refresh token"
// @Success      200      {object}  LoginResponse    "New token pair"
// @Failure      400      {object}  ErrorResponse    "Invalid input"
// @Failure      401      {object}  ErrorResponse    "Invalid or expired refresh token"
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	tokens, err := h.authUsecase.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}

	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second).UnixMilli()
	c.JSON(http.StatusOK, LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		ExpiresAt:    expiresAt,
	})
}

// Logout godoc
// @Summary      Logout user
// @Description  Logout the current authenticated user and revoke all refresh tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  AuthResponse  "Logout successful"
// @Failure      401  {object}  ErrorResponse "Unauthorized"
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
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

	if err := h.authUsecase.Logout(tokenPayload.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{Message: "Logout successful"})
}

// GetProfile godoc
// @Summary      Get user profile
// @Description  Retrieve the current authenticated user's profile
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  ProfileResponse "User profile retrieved successfully"
// @Failure      401  {object}  ErrorResponse "Unauthorized"
// @Failure      404  {object}  ErrorResponse "User not found"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
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

	user, err := h.authUsecase.GetProfile(tokenPayload.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	response := ProfileResponse{
		ID:            user.ID,
		Email:         user.Email,
		Provider:      user.Provider,
		EmailVerified: user.EmailVerified,
		CreatedAt:     user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	c.JSON(http.StatusOK, response)
}
