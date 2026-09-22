package domain

import (
	"errors"
	"time"
)

// ErrEmailNotVerified is returned when a Firebase email/password user has not completed email verification.
var ErrEmailNotVerified = errors.New("email not verified: check your inbox and verify before signing in")

// TokenPayload carries user identity inside JWTs.
type TokenPayload struct {
	UserID string
	Email  string
}

// AuthTokens is the dual-token response returned on login/refresh.
type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds until access token expires
}

// RefreshToken represents a stored refresh token in the database.
type RefreshToken struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	TokenHash  string    `json:"token_hash"`
	DeviceInfo string    `json:"device_info"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// JWTService handles access and refresh token generation/validation.
type JWTService interface {
	GenerateToken(payload *TokenPayload) (string, error)
	ValidateToken(token string) (*TokenPayload, error)
	GenerateRefreshToken(payload *TokenPayload) (string, error)
	ValidateRefreshToken(token string) (*TokenPayload, error)
	GetAccessTokenExpiry() int // returns expiry in seconds
}

// RefreshTokenRepository manages refresh token persistence.
type RefreshTokenRepository interface {
	Create(token *RefreshToken) error
	FindByHash(tokenHash string) (*RefreshToken, error)
	DeleteByHash(tokenHash string) error
	DeleteAllByUserID(userID string) error
}
