package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skiix-backend/internal/domain"
)

type PostgresCircleRepository struct {
	db *sql.DB
}

func NewPostgresCircleRepository(db *sql.DB) domain.CircleRepository {
	return &PostgresCircleRepository{db: db}
}

func (r *PostgresCircleRepository) Create(circle *domain.Circle) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	circle.CreatedAt = time.Now()
	circle.UpdatedAt = time.Now()

	createCircleQuery := `
		INSERT INTO circles (id, owner_id, name, description, avatar_url, member_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = tx.Exec(createCircleQuery,
		circle.ID, circle.OwnerID, circle.Name, circle.Description, circle.AvatarURL,
		circle.MemberCount, circle.CreatedAt, circle.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert circle: %w", err)
	}

	createMemberQuery := `
		INSERT INTO circle_members (id, circle_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = tx.Exec(createMemberQuery, uuid.New().String(), circle.ID, circle.OwnerID, "owner", circle.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert owner as member: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *PostgresCircleRepository) GetByID(id string) (*domain.Circle, error) {
	query := `
		SELECT id, owner_id, name, description, avatar_url, member_count, created_at, updated_at
		FROM circles
		WHERE id = $1
	`
	circle := &domain.Circle{}
	err := r.db.QueryRow(query, id).Scan(
		&circle.ID, &circle.OwnerID, &circle.Name, &circle.Description,
		&circle.AvatarURL, &circle.MemberCount, &circle.CreatedAt, &circle.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("circle not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check circle: %w", err)
	}
	return circle, nil
}

func (r *PostgresCircleRepository) GetByUser(userID string) ([]domain.Circle, error) {
	query := `
		SELECT c.id, c.owner_id, c.name, c.description, c.avatar_url, c.member_count, c.created_at, c.updated_at
		FROM circles c
		INNER JOIN circle_members cm ON c.id = cm.circle_id
		WHERE cm.user_id = $1
		ORDER BY c.created_at DESC
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list circles: %w", err)
	}
	defer rows.Close()

	circles := make([]domain.Circle, 0)
	for rows.Next() {
		var c domain.Circle
		if err := rows.Scan(
			&c.ID, &c.OwnerID, &c.Name, &c.Description,
			&c.AvatarURL, &c.MemberCount, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		circles = append(circles, c)
	}
	return circles, nil
}

func (r *PostgresCircleRepository) Update(circle *domain.Circle) error {
	circle.UpdatedAt = time.Now()
	query := `
		UPDATE circles
		SET name = $1, description = $2, avatar_url = $3, updated_at = $4
		WHERE id = $5
	`
	result, err := r.db.Exec(query, circle.Name, circle.Description, circle.AvatarURL, circle.UpdatedAt, circle.ID)
	if err != nil {
		return fmt.Errorf("failed to update circle: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return errors.New("circle not found")
	}
	return nil
}

func (r *PostgresCircleRepository) Delete(id string) error {
	query := `DELETE FROM circles WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete circle: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return errors.New("circle not found")
	}
	return nil
}

func (r *PostgresCircleRepository) AddMembers(circleID string, userIDs []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	insertQuery := `
		INSERT INTO circle_members (id, circle_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (circle_id, user_id) DO NOTHING
	`
	now := time.Now()
	addedCount := 0
	for _, uid := range userIDs {
		res, err := tx.Exec(insertQuery, uuid.New().String(), circleID, uid, "member", now)
		if err != nil {
			return err
		}
		aff, _ := res.RowsAffected()
		if aff > 0 {
			addedCount++
		}
	}

	if addedCount > 0 {
		if _, err := tx.Exec(`UPDATE circles SET member_count = member_count + $1 WHERE id = $2`, addedCount, circleID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresCircleRepository) RemoveMember(circleID string, userID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`DELETE FROM circle_members WHERE circle_id = $1 AND user_id = $2`, circleID, userID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected > 0 {
		if _, err := tx.Exec(`UPDATE circles SET member_count = GREATEST(member_count - 1, 1) WHERE id = $1`, circleID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresCircleRepository) IsMember(circleID string, userID string) (bool, error) {
	var id string
	err := r.db.QueryRow(`SELECT id FROM circle_members WHERE circle_id = $1 AND user_id = $2`, circleID, userID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *PostgresCircleRepository) GetMembers(circleID string, limit, offset int) ([]domain.CircleMember, error) {
	query := `
		SELECT id, circle_id, user_id, role, joined_at
		FROM circle_members
		WHERE circle_id = $1
		ORDER BY joined_at ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(query, circleID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]domain.CircleMember, 0)
	for rows.Next() {
		var m domain.CircleMember
		if err := rows.Scan(&m.ID, &m.CircleID, &m.UserID, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

func (r *PostgresCircleRepository) CountByOwner(ownerID string) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM circles WHERE owner_id = $1`, ownerID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
