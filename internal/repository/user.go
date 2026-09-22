package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/skiix-backend/internal/domain"
)

type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) domain.UserRepository {
	return &PostgresUserRepository{
		db: db,
	}
}

func (r *PostgresUserRepository) Create(user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password, provider, firebase_uid, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := r.db.Exec(query, user.ID, user.Email, user.Password, user.Provider, user.FirebaseUID, user.EmailVerified, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint \"users_email_key\"" {
			return errors.New("user with this email already exists")
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func scanUser(row *sql.Row) (*domain.User, error) {
	user := &domain.User{}
	var firebaseNull sql.NullString
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Provider,
		&firebaseNull,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}
	if firebaseNull.Valid {
		s := firebaseNull.String
		user.FirebaseUID = &s
	}
	return user, nil
}

func (r *PostgresUserRepository) FindByEmail(email string) (*domain.User, error) {
	q := `SELECT id, email, password, provider, firebase_uid, email_verified, created_at, updated_at
		FROM users WHERE email = $1`
	user, err := scanUser(r.db.QueryRow(q, email))
	if err != nil && err.Error() == "user not found" {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return user, nil
}

func (r *PostgresUserRepository) FindByID(id string) (*domain.User, error) {
	q := `SELECT id, email, password, provider, firebase_uid, email_verified, created_at, updated_at
		FROM users WHERE id = $1`
	user, err := scanUser(r.db.QueryRow(q, id))
	if err != nil && err.Error() == "user not found" {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return user, nil
}

func (r *PostgresUserRepository) FindByFirebaseUID(uid string) (*domain.User, error) {
	q := `SELECT id, email, password, provider, firebase_uid, email_verified, created_at, updated_at
		FROM users WHERE firebase_uid = $1`
	user, err := scanUser(r.db.QueryRow(q, uid))
	if err != nil && err.Error() == "user not found" {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return user, nil
}

func (r *PostgresUserRepository) UpdateUserFirebaseInfo(user *domain.User) error {
	now := time.Now()
	_, err := r.db.Exec(
		`UPDATE users SET firebase_uid = $1, email_verified = $2, updated_at = $3, provider = COALESCE($4, provider) WHERE id = $5`,
		user.FirebaseUID,
		user.EmailVerified,
		now,
		user.Provider,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user firebase: %w", err)
	}
	return nil
}
