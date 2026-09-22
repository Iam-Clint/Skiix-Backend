package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skiix-backend/internal/domain"
)

type PostgresProjectMemberRepository struct {
	db *sql.DB
}

func NewPostgresProjectMemberRepository(db *sql.DB) domain.ProjectMemberRepository {
	return &PostgresProjectMemberRepository{db: db}
}

func (r *PostgresProjectMemberRepository) Join(userID, projectID string) (bool, error) {
	userID = strings.TrimSpace(userID)
	projectID = strings.TrimSpace(projectID)
	if userID == "" || projectID == "" {
		return false, fmt.Errorf("missing user_id or project_id")
	}
	id := uuid.New().String()
	now := time.Now().UTC()
	res, err := r.db.Exec(
		`INSERT INTO project_members (id, user_id, project_id, created_at)
		 VALUES ($1,$2,$3,$4)
		 ON CONFLICT (user_id, project_id) DO NOTHING`,
		id, userID, projectID, now,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 0, nil
}

type PostgresFollowerRepository struct {
	db *sql.DB
}

func NewPostgresFollowerRepository(db *sql.DB) domain.FollowerRepository {
	return &PostgresFollowerRepository{db: db}
}

func (r *PostgresFollowerRepository) Follow(followerID, followingID string) (bool, error) {
	followerID = strings.TrimSpace(followerID)
	followingID = strings.TrimSpace(followingID)
	if followerID == "" || followingID == "" {
		return false, fmt.Errorf("missing follower_id or following_id")
	}
	if followerID == followingID {
		return false, fmt.Errorf("cannot follow yourself")
	}
	id := uuid.New().String()
	now := time.Now().UTC()
	res, err := r.db.Exec(
		`INSERT INTO followers (id, follower_id, following_id, created_at)
		 VALUES ($1,$2,$3,$4)
		 ON CONFLICT (follower_id, following_id) DO NOTHING`,
		id, followerID, followingID, now,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 0, nil
}
