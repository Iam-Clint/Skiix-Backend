package usecase

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skiix-backend/internal/domain"
	"github.com/skiix-backend/internal/infrastructure"
)

const invalidCredentials = "invalid credentials"

// AuthUsecase defines the authentication operations.
type AuthUsecase interface {
	Register(email, password string) error
	Login(email, password string) (*domain.AuthTokens, *domain.User, error)
	OAuthLogin(email, provider string) (*domain.AuthTokens, *domain.User, error)
	RefreshAccessToken(refreshToken string) (*domain.AuthTokens, error)
	Logout(userID string) error
	GetProfile(userID string) (*domain.User, error)
}

type AuthUsecaseImpl struct {
	userRepo         domain.UserRepository
	jwtService       domain.JWTService
	passwordService  infrastructure.PasswordService
	refreshTokenRepo domain.RefreshTokenRepository
}

func NewAuthUsecase(
	userRepo domain.UserRepository,
	jwtService domain.JWTService,
	passwordService infrastructure.PasswordService,
	refreshTokenRepo domain.RefreshTokenRepository,
) AuthUsecase {
	return &AuthUsecaseImpl{
		userRepo:         userRepo,
		jwtService:       jwtService,
		passwordService:  passwordService,
		refreshTokenRepo: refreshTokenRepo,
	}
}

func (a *AuthUsecaseImpl) Register(email string, password string) error {
	if err := a.validateEmail(email); err != nil {
		return err
	}

	if err := a.validatePassword(password); err != nil {
		return err
	}

	hashedPassword, err := a.passwordService.HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to process registration: %w", err)
	}

	provider := "local"

	user := &domain.User{
		ID:            uuid.New().String(),
		Email:         email,
		Password:      &hashedPassword,
		Provider:      &provider,
		EmailVerified: true,
	}

	if err := a.userRepo.Create(user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (a *AuthUsecaseImpl) Login(email, password string) (*domain.AuthTokens, *domain.User, error) {
	if email == "" || password == "" {
		return nil, nil, errors.New("email and password are required")
	}

	user, err := a.userRepo.FindByEmail(email)
	if err != nil {
		return nil, nil, errors.New(invalidCredentials)
	}

	if user.Password == nil {
		return nil, nil, errors.New("this account uses another sign-in method")
	}

	if err := a.passwordService.VerifyPassword(*user.Password, password); err != nil {
		return nil, nil, errors.New(invalidCredentials)
	}
	if !user.EmailVerified {
		return nil, nil, domain.ErrEmailNotVerified
	}

	tokens, err := a.generateTokenPair(user)
	if err != nil {
		return nil, nil, err
	}
	return tokens, user, nil
}

func (a *AuthUsecaseImpl) OAuthLogin(email string, provider string) (*domain.AuthTokens, *domain.User, error) {
	if email == "" || provider == "" {
		return nil, nil, errors.New("email and provider are required")
	}

	user, err := a.userRepo.FindByEmail(email)
	if err != nil {
		user = &domain.User{
			ID:            uuid.New().String(),
			Email:         email,
			Provider:      &provider,
			EmailVerified: true,
		}

		if err := a.userRepo.Create(user); err != nil {
			return nil, nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	t, err := a.generateTokenPair(user)
	if err != nil {
		return nil, nil, err
	}
	return t, user, nil
}

// RefreshAccessToken validates the refresh token, rotates it, and returns new tokens.
// Token rotation: the old refresh token is deleted and a new one is issued.
// This prevents replay attacks — if someone steals a used refresh token, it won't work.
func (a *AuthUsecaseImpl) RefreshAccessToken(refreshToken string) (*domain.AuthTokens, error) {
	// 1. Validate the JWT signature and expiry
	payload, err := a.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}

	// 2. Check if the token exists in the database (not revoked)
	tokenHash := hashToken(refreshToken)
	storedToken, err := a.refreshTokenRepo.FindByHash(tokenHash)
	if err != nil {
		return nil, errors.New("refresh token has been revoked")
	}

	// 3. Check if the token has expired in the database
	if time.Now().After(storedToken.ExpiresAt) {
		_ = a.refreshTokenRepo.DeleteByHash(tokenHash)
		return nil, errors.New("refresh token has expired")
	}

	// 4. Delete the old refresh token (rotation)
	_ = a.refreshTokenRepo.DeleteByHash(tokenHash)

	// 5. Generate a new token pair
	user, err := a.userRepo.FindByID(payload.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	return a.generateTokenPair(user)
}

func (a *AuthUsecaseImpl) Logout(userID string) error {
	if userID == "" {
		return errors.New("user id is required")
	}
	// Revoke ALL refresh tokens for this user (all sessions)
	return a.refreshTokenRepo.DeleteAllByUserID(userID)
}

func (a *AuthUsecaseImpl) GetProfile(userID string) (*domain.User, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}

	user, err := a.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// --- Internal helpers ---

// generateTokenPair creates both access and refresh tokens, stores the refresh token hash in DB.
func (a *AuthUsecaseImpl) generateTokenPair(user *domain.User) (*domain.AuthTokens, error) {
	tokenPayload := &domain.TokenPayload{
		UserID: user.ID,
		Email:  user.Email,
	}

	accessToken, err := a.jwtService.GenerateToken(tokenPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := a.jwtService.GenerateRefreshToken(tokenPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store the refresh token hash in the database
	refreshRecord := &domain.RefreshToken{
		UserID:     user.ID,
		TokenHash:  hashToken(refreshToken),
		DeviceInfo: "mobile", // Can be enhanced later with actual device info
		ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
	}

	if err := a.refreshTokenRepo.Create(refreshRecord); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &domain.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    a.jwtService.GetAccessTokenExpiry(),
	}, nil
}

// hashToken creates a SHA256 hash of the token string.
// We never store the raw token — only the hash — for security.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}

func (a *AuthUsecaseImpl) validateEmail(email string) error {
	if email == "" {
		return errors.New("email cannot be empty")
	}
	if len(email) > 254 {
		return errors.New("email is too long")
	}
	return nil
}

func (a *AuthUsecaseImpl) validatePassword(password string) error {
	if password == "" {
		return errors.New("password cannot be empty")
	}
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	return nil
}
