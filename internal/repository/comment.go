package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/skiix-backend/internal/domain"
)

// PostgresCommentRepository implements domain.CommentRepository.
type PostgresCommentRepository struct {
	db *sql.DB
}

// NewPostgresCommentRepository creates a new comment repository.
func NewPostgresCommentRepository(db *sql.DB) domain.CommentRepository {
	return &PostgresCommentRepository{db: db}
}

// Create inserts a new comment.
func (r *PostgresCommentRepository) Create(comment *domain.Comment) error {
	query := `
		INSERT INTO comments (id, post_id, user_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	comment.CreatedAt = time.Now()
	comment.UpdatedAt = time.Now()

	_, err := r.db.Exec(query,
		comment.ID, comment.PostID, comment.UserID,
		comment.Content, comment.CreatedAt, comment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	return nil
}

// GetByPostID retrieves all comments for a post, ordered by creation time.
func (r *PostgresCommentRepository) GetByPostID(postID string, limit, offset int) ([]domain.Comment, error) {
	query := `
		SELECT id, post_id, user_id, content, created_at, updated_at
		FROM comments
		WHERE post_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, postID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	defer rows.Close()

	comments := make([]domain.Comment, 0)
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(
			&c.ID, &c.PostID, &c.UserID,
			&c.Content, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating comments: %w", err)
	}

	return comments, nil
}

// GetByID retrieves a single comment by ID.
func (r *PostgresCommentRepository) GetByID(commentID string) (*domain.Comment, error) {
	query := `
		SELECT id, post_id, user_id, content, created_at, updated_at
		FROM comments
		WHERE id = $1
	`

	c := &domain.Comment{}
	err := r.db.QueryRow(query, commentID).Scan(
		&c.ID, &c.PostID, &c.UserID,
		&c.Content, &c.CreatedAt, &c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("comment not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	return c, nil
}

// Delete removes a comment by ID.
func (r *PostgresCommentRepository) Delete(commentID string) error {
	result, err := r.db.Exec(`DELETE FROM comments WHERE id = $1`, commentID)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("comment not found")
	}

	return nil
}
