package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skiix-backend/internal/domain"
)

// PostgresRefreshTokenRepository implements RefreshTokenRepository using PostgreSQL.
type PostgresRefreshTokenRepository struct {
	db *sql.DB
}

func NewPostgresRefreshTokenRepository(db *sql.DB) domain.RefreshTokenRepository {
	return &PostgresRefreshTokenRepository{db: db}
}

// Create inserts a new refresh token record.
func (r *PostgresRefreshTokenRepository) Create(token *domain.RefreshToken) error {
	token.ID = uuid.New().String()
	token.CreatedAt = time.Now()

	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, device_info, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(query,
		token.ID, token.UserID, token.TokenHash, token.DeviceInfo, token.ExpiresAt, token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}
	return nil
}

// FindByHash retrieves a refresh token by its SHA256 hash.
func (r *PostgresRefreshTokenRepository) FindByHash(tokenHash string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, device_info, expires_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`
	token := &domain.RefreshToken{}
	err := r.db.QueryRow(query, tokenHash).Scan(
		&token.ID, &token.UserID, &token.TokenHash, &token.DeviceInfo, &token.ExpiresAt, &token.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("refresh token not found: %w", err)
	}
	return token, nil
}

// DeleteByHash removes a single refresh token (used during token rotation).
func (r *PostgresRefreshTokenRepository) DeleteByHash(tokenHash string) error {
	query := `DELETE FROM refresh_tokens WHERE token_hash = $1`
	_, err := r.db.Exec(query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}
	return nil
}

// DeleteAllByUserID removes all refresh tokens for a user (used on logout to revoke all sessions).
func (r *PostgresRefreshTokenRepository) DeleteAllByUserID(userID string) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete all refresh tokens: %w", err)
	}
	return nil
}
