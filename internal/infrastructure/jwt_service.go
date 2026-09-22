package infrastructure

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/skiix-backend/internal/domain"
)

const (
	accessTokenExpiry  = 15 * time.Minute   // Short-lived for security
	refreshTokenExpiry = 7 * 24 * time.Hour // 7 days
)

type JWTServiceImpl struct {
	accessSecret  string
	refreshSecret string
}

func NewJWTService() domain.JWTService {
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if refreshSecret == "" {
		// Fallback: derive from access secret if not set separately
		refreshSecret = os.Getenv("JWT_SECRET") + "_refresh"
	}
	return &JWTServiceImpl{
		accessSecret:  os.Getenv("JWT_SECRET"),
		refreshSecret: refreshSecret,
	}
}

type CustomClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GetAccessTokenExpiry returns the access token lifetime in seconds.
func (s *JWTServiceImpl) GetAccessTokenExpiry() int {
	return int(accessTokenExpiry.Seconds())
}

// GenerateToken creates a short-lived access token (15 minutes).
func (s *JWTServiceImpl) GenerateToken(payload *domain.TokenPayload) (string, error) {
	return s.generateTokenWithSecret(payload, s.accessSecret, accessTokenExpiry)
}

// ValidateToken validates an access token.
func (s *JWTServiceImpl) ValidateToken(tokenString string) (*domain.TokenPayload, error) {
	return s.validateTokenWithSecret(tokenString, s.accessSecret)
}

// GenerateRefreshToken creates a long-lived refresh token (7 days).
func (s *JWTServiceImpl) GenerateRefreshToken(payload *domain.TokenPayload) (string, error) {
	return s.generateTokenWithSecret(payload, s.refreshSecret, refreshTokenExpiry)
}

// ValidateRefreshToken validates a refresh token using the refresh secret.
func (s *JWTServiceImpl) ValidateRefreshToken(tokenString string) (*domain.TokenPayload, error) {
	return s.validateTokenWithSecret(tokenString, s.refreshSecret)
}

// --- Internal helpers ---

func (s *JWTServiceImpl) generateTokenWithSecret(payload *domain.TokenPayload, secret string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := CustomClaims{
		UserID: payload.UserID,
		Email:  payload.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, nil
}

func (s *JWTServiceImpl) validateTokenWithSecret(tokenString string, secret string) (*domain.TokenPayload, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return &domain.TokenPayload{
		UserID: claims.UserID,
		Email:  claims.Email,
	}, nil
}
