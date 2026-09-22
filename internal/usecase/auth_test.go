package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/skiix-backend/internal/domain"
	"github.com/skiix-backend/internal/usecase"
	"github.com/stretchr/testify/assert"
)

// --- Mocks ---

type MockUserRepository struct {
	CreateFunc                 func(user *domain.User) error
	FindByEmailFunc            func(email string) (*domain.User, error)
	FindByIDFunc               func(id string) (*domain.User, error)
	FindByFirebaseUIDFunc      func(uid string) (*domain.User, error)
	UpdateUserFirebaseInfoFunc func(user *domain.User) error
}

func (m *MockUserRepository) Create(user *domain.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(user)
	}
	return nil
}

func (m *MockUserRepository) FindByEmail(email string) (*domain.User, error) {
	if m.FindByEmailFunc != nil {
		return m.FindByEmailFunc(email)
	}
	return nil, errors.New("not found")
}

func (m *MockUserRepository) FindByID(id string) (*domain.User, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(id)
	}
	return nil, errors.New("not found")
}

func (m *MockUserRepository) FindByFirebaseUID(uid string) (*domain.User, error) {
	if m.FindByFirebaseUIDFunc != nil {
		return m.FindByFirebaseUIDFunc(uid)
	}
	return nil, errors.New("user not found")
}

func (m *MockUserRepository) UpdateUserFirebaseInfo(user *domain.User) error {
	if m.UpdateUserFirebaseInfoFunc != nil {
		return m.UpdateUserFirebaseInfoFunc(user)
	}
	return nil
}

type MockJWTService struct {
	GenerateTokenFunc        func(payload *domain.TokenPayload) (string, error)
	ValidateTokenFunc        func(token string) (*domain.TokenPayload, error)
	GenerateRefreshTokenFunc func(payload *domain.TokenPayload) (string, error)
	ValidateRefreshTokenFunc func(token string) (*domain.TokenPayload, error)
}

func (m *MockJWTService) GenerateToken(payload *domain.TokenPayload) (string, error) {
	if m.GenerateTokenFunc != nil {
		return m.GenerateTokenFunc(payload)
	}
	return "mock_access_token", nil
}

func (m *MockJWTService) ValidateToken(token string) (*domain.TokenPayload, error) {
	if m.ValidateTokenFunc != nil {
		return m.ValidateTokenFunc(token)
	}
	return &domain.TokenPayload{}, nil
}

func (m *MockJWTService) GenerateRefreshToken(payload *domain.TokenPayload) (string, error) {
	if m.GenerateRefreshTokenFunc != nil {
		return m.GenerateRefreshTokenFunc(payload)
	}
	return "mock_refresh_token", nil
}

func (m *MockJWTService) ValidateRefreshToken(token string) (*domain.TokenPayload, error) {
	if m.ValidateRefreshTokenFunc != nil {
		return m.ValidateRefreshTokenFunc(token)
	}
	return &domain.TokenPayload{UserID: "user-123", Email: "user@example.com"}, nil
}

func (m *MockJWTService) GetAccessTokenExpiry() int {
	return 900
}

type MockPasswordService struct {
	HashPasswordFunc   func(password string) (string, error)
	VerifyPasswordFunc func(hashedPassword, password string) error
}

func (m *MockPasswordService) HashPassword(password string) (string, error) {
	if m.HashPasswordFunc != nil {
		return m.HashPasswordFunc(password)
	}
	return "hashed_password", nil
}

func (m *MockPasswordService) VerifyPassword(hashedPassword, password string) error {
	if m.VerifyPasswordFunc != nil {
		return m.VerifyPasswordFunc(hashedPassword, password)
	}
	return nil
}

type MockRefreshTokenRepository struct {
	CreateFunc          func(token *domain.RefreshToken) error
	FindByHashFunc      func(tokenHash string) (*domain.RefreshToken, error)
	DeleteByHashFunc    func(tokenHash string) error
	DeleteAllByUserFunc func(userID string) error
}

func (m *MockRefreshTokenRepository) Create(token *domain.RefreshToken) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(token)
	}
	return nil
}

func (m *MockRefreshTokenRepository) FindByHash(tokenHash string) (*domain.RefreshToken, error) {
	if m.FindByHashFunc != nil {
		return m.FindByHashFunc(tokenHash)
	}
	return &domain.RefreshToken{
		ID:        "rt-123",
		UserID:    "user-123",
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}, nil
}

func (m *MockRefreshTokenRepository) DeleteByHash(tokenHash string) error {
	if m.DeleteByHashFunc != nil {
		return m.DeleteByHashFunc(tokenHash)
	}
	return nil
}

func (m *MockRefreshTokenRepository) DeleteAllByUserID(userID string) error {
	if m.DeleteAllByUserFunc != nil {
		return m.DeleteAllByUserFunc(userID)
	}
	return nil
}

// --- Helper ---

func newAuthUsecase(repo *MockUserRepository, jwt *MockJWTService, pwd *MockPasswordService, rt *MockRefreshTokenRepository) usecase.AuthUsecase {
	if rt == nil {
		rt = &MockRefreshTokenRepository{}
	}
	return usecase.NewAuthUsecase(repo, jwt, pwd, rt)
}

// ===========================
// Register Tests
// ===========================

func TestRegister_ValidInputs(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		password   string
		setupMocks func(*MockUserRepository, *MockPasswordService)
		wantErr    bool
		errMsg     string
	}{
		{
			name:     "successful registration with valid email and password",
			email:    "user@example.com",
			password: "password123",
			setupMocks: func(mockRepo *MockUserRepository, mockPwd *MockPasswordService) {
				mockRepo.CreateFunc = func(user *domain.User) error { return nil }
				mockPwd.HashPasswordFunc = func(password string) (string, error) { return "hashed_password", nil }
			},
			wantErr: false,
		},
		{
			name:     "successful registration with minimum password length",
			email:    "test@skiix.io",
			password: "123456",
			setupMocks: func(mockRepo *MockUserRepository, mockPwd *MockPasswordService) {
				mockRepo.CreateFunc = func(user *domain.User) error { return nil }
				mockPwd.HashPasswordFunc = func(password string) (string, error) { return "hashed_password", nil }
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockUserRepository{}
			mockPwd := &MockPasswordService{}
			mockJWT := &MockJWTService{}
			tt.setupMocks(mockRepo, mockPwd)

			authUsecase := newAuthUsecase(mockRepo, mockJWT, mockPwd, nil)
			err := authUsecase.Register(tt.email, tt.password)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		errMsg   string
	}{
		{name: "empty email", email: "", password: "password123", errMsg: "email cannot be empty"},
		{name: "email exceeds maximum length", email: "a" + string(make([]byte, 254)) + "@example.com", password: "password123", errMsg: "email is too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := newAuthUsecase(&MockUserRepository{}, &MockJWTService{}, &MockPasswordService{}, nil)
			err := uc.Register(tt.email, tt.password)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestRegister_InvalidPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		errMsg   string
	}{
		{name: "empty password", password: "", errMsg: "password cannot be empty"},
		{name: "password less than 6 characters", password: "12345", errMsg: "password must be at least 6 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := newAuthUsecase(&MockUserRepository{}, &MockJWTService{}, &MockPasswordService{}, nil)
			err := uc.Register("user@example.com", tt.password)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestRegister_PasswordServiceError(t *testing.T) {
	mockPwd := &MockPasswordService{
		HashPasswordFunc: func(password string) (string, error) { return "", errors.New("hash failed") },
	}
	uc := newAuthUsecase(&MockUserRepository{}, &MockJWTService{}, mockPwd, nil)
	err := uc.Register("user@example.com", "password123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to process registration")
}

func TestRegister_UserRepositoryError(t *testing.T) {
	mockRepo := &MockUserRepository{
		CreateFunc: func(user *domain.User) error { return errors.New("database error") },
	}
	mockPwd := &MockPasswordService{
		HashPasswordFunc: func(password string) (string, error) { return "hashed_password", nil },
	}
	uc := newAuthUsecase(mockRepo, &MockJWTService{}, mockPwd, nil)
	err := uc.Register("user@example.com", "password123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create user")
}

// ===========================
// Login Tests (Dual Token)
// ===========================

func TestLogin_SuccessfulLogin(t *testing.T) {
	hashedPwd := "hashed_password"
	mockUser := &domain.User{ID: "user-123", Email: "user@example.com", Password: &hashedPwd, EmailVerified: true}

	mockRepo := &MockUserRepository{
		FindByEmailFunc: func(email string) (*domain.User, error) { return mockUser, nil },
	}
	mockJWT := &MockJWTService{
		GenerateTokenFunc:        func(p *domain.TokenPayload) (string, error) { return "access_tok", nil },
		GenerateRefreshTokenFunc: func(p *domain.TokenPayload) (string, error) { return "refresh_tok", nil },
	}
	mockPwd := &MockPasswordService{
		VerifyPasswordFunc: func(h, p string) error { return nil },
	}

	uc := newAuthUsecase(mockRepo, mockJWT, mockPwd, nil)
	tokens, _, err := uc.Login("user@example.com", "password123")

	assert.NoError(t, err)
	assert.NotNil(t, tokens)
	assert.Equal(t, "access_tok", tokens.AccessToken)
	assert.Equal(t, "refresh_tok", tokens.RefreshToken)
	assert.Equal(t, 900, tokens.ExpiresIn)
}

func TestLogin_EmptyCredentials(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		errMsg   string
	}{
		{name: "empty email", email: "", password: "password123", errMsg: "email and password are required"},
		{name: "empty password", email: "user@example.com", password: "", errMsg: "email and password are required"},
		{name: "both empty", email: "", password: "", errMsg: "email and password are required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := newAuthUsecase(&MockUserRepository{}, &MockJWTService{}, &MockPasswordService{}, nil)
			tokens, _, err := uc.Login(tt.email, tt.password)
			assert.Error(t, err)
			assert.Nil(t, tokens)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	mockRepo := &MockUserRepository{
		FindByEmailFunc: func(email string) (*domain.User, error) { return nil, errors.New("not found") },
	}
	uc := newAuthUsecase(mockRepo, &MockJWTService{}, &MockPasswordService{}, nil)
	tokens, _, err := uc.Login("nonexistent@example.com", "password123")
	assert.Error(t, err)
	assert.Nil(t, tokens)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestLogin_InvalidPassword(t *testing.T) {
	hashedPwd := "hashed_password"
	mockUser := &domain.User{ID: "user-123", Email: "user@example.com", Password: &hashedPwd, EmailVerified: true}

	mockRepo := &MockUserRepository{
		FindByEmailFunc: func(email string) (*domain.User, error) { return mockUser, nil },
	}
	mockPwd := &MockPasswordService{
		VerifyPasswordFunc: func(h, p string) error { return errors.New("password mismatch") },
	}
	uc := newAuthUsecase(mockRepo, &MockJWTService{}, mockPwd, nil)
	tokens, _, err := uc.Login("user@example.com", "wrongpassword")
	assert.Error(t, err)
	assert.Nil(t, tokens)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestLogin_TokenGenerationError(t *testing.T) {
	hashedPwd := "hashed_password"
	mockUser := &domain.User{ID: "user-123", Email: "user@example.com", Password: &hashedPwd, EmailVerified: true}

	mockRepo := &MockUserRepository{
		FindByEmailFunc: func(email string) (*domain.User, error) { return mockUser, nil },
	}
	mockJWT := &MockJWTService{
		GenerateTokenFunc: func(p *domain.TokenPayload) (string, error) { return "", errors.New("jwt error") },
	}
	mockPwd := &MockPasswordService{
		VerifyPasswordFunc: func(h, p string) error { return nil },
	}
	uc := newAuthUsecase(mockRepo, mockJWT, mockPwd, nil)
	tokens, _, err := uc.Login("user@example.com", "password123")
	assert.Error(t, err)
	assert.Nil(t, tokens)
	assert.Contains(t, err.Error(), "failed to generate access token")
}

// ===========================
// OAuth Login Tests
// ===========================

func TestOAuthLogin_SuccessfulLoginExistingUser(t *testing.T) {
	hashedPwd := "hashed_password"
	mockUser := &domain.User{ID: "user-123", Email: "user@example.com", Password: &hashedPwd, EmailVerified: true}

	mockRepo := &MockUserRepository{
		FindByEmailFunc: func(email string) (*domain.User, error) { return mockUser, nil },
	}
	mockJWT := &MockJWTService{
		GenerateTokenFunc:        func(p *domain.TokenPayload) (string, error) { return "access_tok", nil },
		GenerateRefreshTokenFunc: func(p *domain.TokenPayload) (string, error) { return "refresh_tok", nil },
	}

	uc := newAuthUsecase(mockRepo, mockJWT, &MockPasswordService{}, nil)
	tokens, _, err := uc.OAuthLogin("user@example.com", "google")
	assert.NoError(t, err)
	assert.NotNil(t, tokens)
	assert.Equal(t, "access_tok", tokens.AccessToken)
}

func TestOAuthLogin_SuccessfulLoginNewUser(t *testing.T) {
	mockRepo := &MockUserRepository{
		FindByEmailFunc: func(email string) (*domain.User, error) { return nil, errors.New("not found") },
		CreateFunc:      func(user *domain.User) error { return nil },
	}
	mockJWT := &MockJWTService{
		GenerateTokenFunc:        func(p *domain.TokenPayload) (string, error) { return "access_tok", nil },
		GenerateRefreshTokenFunc: func(p *domain.TokenPayload) (string, error) { return "refresh_tok", nil },
	}

	uc := newAuthUsecase(mockRepo, mockJWT, &MockPasswordService{}, nil)
	tokens, _, err := uc.OAuthLogin("newuser@example.com", "google")
	assert.NoError(t, err)
	assert.NotNil(t, tokens)
}

func TestOAuthLogin_EmptyEmailOrProvider(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		provider string
		errMsg   string
	}{
		{name: "empty email", email: "", provider: "google", errMsg: "email and provider are required"},
		{name: "empty provider", email: "user@example.com", provider: "", errMsg: "email and provider are required"},
		{name: "both empty", email: "", provider: "", errMsg: "email and provider are required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := newAuthUsecase(&MockUserRepository{}, &MockJWTService{}, &MockPasswordService{}, nil)
			tokens, _, err := uc.OAuthLogin(tt.email, tt.provider)
			assert.Error(t, err)
			assert.Nil(t, tokens)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestOAuthLogin_UserCreationError(t *testing.T) {
	mockRepo := &MockUserRepository{
		FindByEmailFunc: func(email string) (*domain.User, error) { return nil, errors.New("not found") },
		CreateFunc:      func(user *domain.User) error { return errors.New("database error") },
	}
	uc := newAuthUsecase(mockRepo, &MockJWTService{}, &MockPasswordService{}, nil)
	tokens, _, err := uc.OAuthLogin("newuser@example.com", "google")
	assert.Error(t, err)
	assert.Nil(t, tokens)
	assert.Contains(t, err.Error(), "failed to create user")
}

func TestOAuthLogin_TokenGenerationError(t *testing.T) {
	hashedPwd := "hashed_password"
	mockUser := &domain.User{ID: "user-123", Email: "user@example.com", Password: &hashedPwd, EmailVerified: true}

	mockRepo := &MockUserRepository{
		FindByEmailFunc: func(email string) (*domain.User, error) { return mockUser, nil },
	}
	mockJWT := &MockJWTService{
		GenerateTokenFunc: func(p *domain.TokenPayload) (string, error) { return "", errors.New("jwt error") },
	}
	uc := newAuthUsecase(mockRepo, mockJWT, &MockPasswordService{}, nil)
	tokens, _, err := uc.OAuthLogin("user@example.com", "google")
	assert.Error(t, err)
	assert.Nil(t, tokens)
	assert.Contains(t, err.Error(), "failed to generate access token")
}

// ===========================
// Refresh Token Tests
// ===========================

func TestRefreshAccessToken_Success(t *testing.T) {
	mockRepo := &MockUserRepository{
		FindByIDFunc: func(id string) (*domain.User, error) {
			return &domain.User{ID: "user-123", Email: "user@example.com", EmailVerified: true}, nil
		},
	}
	mockJWT := &MockJWTService{
		ValidateRefreshTokenFunc: func(token string) (*domain.TokenPayload, error) {
			return &domain.TokenPayload{UserID: "user-123", Email: "user@example.com"}, nil
		},
		GenerateTokenFunc:        func(p *domain.TokenPayload) (string, error) { return "new_access", nil },
		GenerateRefreshTokenFunc: func(p *domain.TokenPayload) (string, error) { return "new_refresh", nil },
	}
	mockRT := &MockRefreshTokenRepository{
		FindByHashFunc: func(h string) (*domain.RefreshToken, error) {
			return &domain.RefreshToken{UserID: "user-123", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
	}

	uc := newAuthUsecase(mockRepo, mockJWT, &MockPasswordService{}, mockRT)
	tokens, err := uc.RefreshAccessToken("old_refresh_token")

	assert.NoError(t, err)
	assert.NotNil(t, tokens)
	assert.Equal(t, "new_access", tokens.AccessToken)
	assert.Equal(t, "new_refresh", tokens.RefreshToken)
}

func TestRefreshAccessToken_InvalidToken(t *testing.T) {
	mockJWT := &MockJWTService{
		ValidateRefreshTokenFunc: func(token string) (*domain.TokenPayload, error) {
			return nil, errors.New("invalid")
		},
	}
	uc := newAuthUsecase(&MockUserRepository{}, mockJWT, &MockPasswordService{}, &MockRefreshTokenRepository{})
	tokens, err := uc.RefreshAccessToken("bad_token")

	assert.Error(t, err)
	assert.Nil(t, tokens)
	assert.Contains(t, err.Error(), "invalid or expired refresh token")
}

func TestRefreshAccessToken_RevokedToken(t *testing.T) {
	mockJWT := &MockJWTService{
		ValidateRefreshTokenFunc: func(token string) (*domain.TokenPayload, error) {
			return &domain.TokenPayload{UserID: "user-123"}, nil
		},
	}
	mockRT := &MockRefreshTokenRepository{
		FindByHashFunc: func(h string) (*domain.RefreshToken, error) {
			return nil, errors.New("not found")
		},
	}
	uc := newAuthUsecase(&MockUserRepository{}, mockJWT, &MockPasswordService{}, mockRT)
	tokens, err := uc.RefreshAccessToken("revoked_token")

	assert.Error(t, err)
	assert.Nil(t, tokens)
	assert.Contains(t, err.Error(), "revoked")
}

// ===========================
// Logout Tests
// ===========================

func TestLogout_Success(t *testing.T) {
	uc := newAuthUsecase(&MockUserRepository{}, &MockJWTService{}, &MockPasswordService{}, nil)
	err := uc.Logout("user-123")
	assert.NoError(t, err)
}

func TestLogout_EmptyUserID(t *testing.T) {
	uc := newAuthUsecase(&MockUserRepository{}, &MockJWTService{}, &MockPasswordService{}, nil)
	err := uc.Logout("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user id is required")
}

// ===========================
// GetProfile Tests
// ===========================

func TestGetProfile_Success(t *testing.T) {
	provider := "local"
	mockUser := &domain.User{ID: "user-123", Email: "user@example.com", Provider: &provider}

	mockRepo := &MockUserRepository{
		FindByIDFunc: func(id string) (*domain.User, error) { return mockUser, nil },
	}
	uc := newAuthUsecase(mockRepo, &MockJWTService{}, &MockPasswordService{}, nil)
	user, err := uc.GetProfile("user-123")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "user-123", user.ID)
	assert.Equal(t, "user@example.com", user.Email)
	assert.Equal(t, "local", *user.Provider)
}

func TestGetProfile_UserNotFound(t *testing.T) {
	mockRepo := &MockUserRepository{
		FindByIDFunc: func(id string) (*domain.User, error) { return nil, errors.New("user not found") },
	}
	uc := newAuthUsecase(mockRepo, &MockJWTService{}, &MockPasswordService{}, nil)
	user, err := uc.GetProfile("nonexistent-id")
	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestGetProfile_EmptyUserID(t *testing.T) {
	uc := newAuthUsecase(&MockUserRepository{}, &MockJWTService{}, &MockPasswordService{}, nil)
	user, err := uc.GetProfile("")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "user id is required")
}

func TestGetProfile_RepositoryError(t *testing.T) {
	mockRepo := &MockUserRepository{
		FindByIDFunc: func(id string) (*domain.User, error) { return nil, errors.New("database connection failed") },
	}
	uc := newAuthUsecase(mockRepo, &MockJWTService{}, &MockPasswordService{}, nil)
	user, err := uc.GetProfile("user-123")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "database connection failed")
}

func TestAuthUsecase_NewAuthUsecase(t *testing.T) {
	uc := newAuthUsecase(&MockUserRepository{}, &MockJWTService{}, &MockPasswordService{}, nil)
	assert.NotNil(t, uc)
}
