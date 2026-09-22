package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skiix-backend/internal/domain"
)

type PostgresStoryRepository struct {
	db *sql.DB
}

func NewPostgresStoryRepository(db *sql.DB) domain.StoryRepository {
	return &PostgresStoryRepository{db: db}
}

func (r *PostgresStoryRepository) Create(story *domain.Story) error {
	_, err := r.db.Exec(
		`INSERT INTO stories (id, author_email, media_url, media_type, caption, created_at, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		story.ID,
		story.AuthorEmail,
		story.MediaURL,
		story.MediaType,
		story.Caption,
		story.CreatedAt,
		story.ExpiresAt,
	)
	return err
}

func (r *PostgresStoryRepository) Feed(forEmail string, limit int) ([]domain.Story, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 60 {
		limit = 60
	}
	forEmail = strings.TrimSpace(strings.ToLower(forEmail))
	now := time.Now().UTC()

	// Stories are visible if:
	// - authored by the user, OR
	// - authored by someone with an accepted connection (either direction).
	q := `
	SELECT s.id, s.author_email, s.media_url, s.media_type, s.caption, s.created_at, s.expires_at
	FROM stories s
	WHERE s.expires_at > $2
	  AND (
	    s.author_email = $1
	    OR EXISTS (
	      SELECT 1 FROM connections c
	      WHERE c.status = 'accepted'
	        AND (
	          (c.from_email = $1 AND c.to_email = s.author_email)
	          OR (c.to_email = $1 AND c.from_email = s.author_email)
	        )
	    )
	  )
	ORDER BY s.created_at DESC
	LIMIT $3;
	`

	rows, err := r.db.Query(q, forEmail, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Story
	for rows.Next() {
		var s domain.Story
		var caption sql.NullString
		if err := rows.Scan(&s.ID, &s.AuthorEmail, &s.MediaURL, &s.MediaType, &caption, &s.CreatedAt, &s.ExpiresAt); err != nil {
			return nil, err
		}
		if caption.Valid {
			s.Caption = &caption.String
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

type PostgresConnectionRepository struct {
	db *sql.DB
}

func NewPostgresConnectionRepository(db *sql.DB) domain.ConnectionRepository {
	return &PostgresConnectionRepository{db: db}
}

func (r *PostgresConnectionRepository) CreateRequest(fromEmail, toEmail string) (*domain.Connection, error) {
	fromEmail = strings.TrimSpace(strings.ToLower(fromEmail))
	toEmail = strings.TrimSpace(strings.ToLower(toEmail))
	if fromEmail == "" || toEmail == "" {
		return nil, fmt.Errorf("missing email")
	}
	if fromEmail == toEmail {
		return nil, fmt.Errorf("cannot connect to yourself")
	}

	id := uuid.New().String()
	now := time.Now().UTC()
	_, err := r.db.Exec(
		`INSERT INTO connections (id, from_email, to_email, status, created_at, updated_at)
		 VALUES ($1,$2,$3,'pending',$4,$4)
		 ON CONFLICT (from_email, to_email) DO UPDATE SET updated_at = EXCLUDED.updated_at
		`,
		id, fromEmail, toEmail, now,
	)
	if err != nil {
		return nil, err
	}
	return &domain.Connection{
		ID:        id,
		FromEmail: fromEmail,
		ToEmail:   toEmail,
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *PostgresConnectionRepository) Accept(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("missing id")
	}
	_, err := r.db.Exec(`UPDATE connections SET status='accepted', updated_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *PostgresConnectionRepository) List(forEmail string, limit, offset int) ([]domain.Connection, error) {
	forEmail = strings.TrimSpace(strings.ToLower(forEmail))
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.db.Query(
		`SELECT id, from_email, to_email, status, created_at, updated_at
		 FROM connections
		 WHERE from_email = $1 OR to_email = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		forEmail, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Connection
	for rows.Next() {
		var c domain.Connection
		if err := rows.Scan(&c.ID, &c.FromEmail, &c.ToEmail, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PostgresConnectionRepository) Delete(id string, participantEmail string) error {
	id = strings.TrimSpace(id)
	participantEmail = strings.TrimSpace(strings.ToLower(participantEmail))
	if id == "" || participantEmail == "" {
		return fmt.Errorf("missing id or email")
	}
	res, err := r.db.Exec(
		`DELETE FROM connections WHERE id = $1 AND (LOWER(from_email) = $2 OR LOWER(to_email) = $2)`,
		id,
		participantEmail,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("connection not found")
	}
	return nil
}
