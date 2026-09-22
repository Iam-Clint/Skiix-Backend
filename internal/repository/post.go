package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skiix-backend/internal/domain"
)

// PostgresPostRepository implements domain.PostRepository
// using raw SQL with parameterized queries.
type PostgresPostRepository struct {
	db *sql.DB
}

// NewPostgresPostRepository creates a new post repository.
func NewPostgresPostRepository(db *sql.DB) domain.PostRepository {
	return &PostgresPostRepository{db: db}
}

// Create inserts a new post into the database.
func (r *PostgresPostRepository) Create(post *domain.Post) error {
	query := `
		INSERT INTO posts (id, user_id, type, content, media_url, image_url, is_project, project_id, visibility, circle_id, like_count, comment_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	post.CreatedAt = time.Now()
	post.UpdatedAt = time.Now()

	_, err := r.db.Exec(query,
		post.ID, post.UserID, post.Type, post.Content,
		post.MediaURL, post.ImageURL,
		post.IsProject, post.ProjectID,
		post.Visibility, post.CircleID,
		post.LikeCount, post.CommentCount,
		post.CreatedAt, post.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	return nil
}

// GetByID retrieves a single post by its ID.
func (r *PostgresPostRepository) GetByID(postID string) (*domain.Post, error) {
	query := `
		SELECT id, user_id, type, content, media_url, image_url, is_project, project_id, visibility, circle_id, like_count, comment_count, created_at, updated_at
		FROM posts
		WHERE id = $1
	`

	post := &domain.Post{}
	err := r.db.QueryRow(query, postID).Scan(
		&post.ID, &post.UserID, &post.Type, &post.Content,
		&post.MediaURL, &post.ImageURL,
		&post.IsProject, &post.ProjectID,
		&post.Visibility, &post.CircleID,
		&post.LikeCount, &post.CommentCount,
		&post.CreatedAt, &post.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("post not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	return post, nil
}

// GetFeed retrieves a paginated list of posts for the main feed,
// enriched with author email and like status for the requesting user.
func (r *PostgresPostRepository) GetFeed(query domain.FeedQuery) ([]domain.PostFeedItem, error) {
	sqlQuery := `
		SELECT
			p.id, p.user_id, p.type, p.content, p.media_url, p.image_url, p.is_project,
			CASE WHEN p.type = 'project' OR p.is_project THEN COALESCE(p.project_id, p.id) ELSE NULL END AS project_id,
			p.like_count, p.comment_count,
			p.created_at, p.updated_at,
			u.email AS author_email,
			EXISTS(
				SELECT 1 FROM post_likes pl
				WHERE pl.post_id = p.id AND pl.user_id = $1
			) AS is_liked
			,
			CASE WHEN p.type IS NOT NULL AND p.type <> '' THEN p.type
			     WHEN p.is_project THEN 'project' ELSE 'text' END AS type,
			EXISTS(
				SELECT 1 FROM project_members pm
				WHERE pm.project_id = COALESCE(p.project_id, p.id) AND pm.user_id = $1
			) AS is_joined,
			EXISTS(
				SELECT 1 FROM followers f
				WHERE f.follower_id = $1 AND f.following_id = p.user_id
			) AS is_following
		FROM posts p
		INNER JOIN users u ON u.id = p.user_id
		WHERE NOT (
			COALESCE((SELECT is_enabled FROM focus_mode_settings WHERE user_id = $1), FALSE)
			AND p.id IN (
				SELECT pc.post_id
				FROM post_categories pc
				INNER JOIN focus_mode_blocked_categories fmbc
					ON pc.category_name = fmbc.category_name
				WHERE fmbc.user_id = $1
			)
		)
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(sqlQuery, query.UserID, query.Limit, query.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}
	defer rows.Close()

	items := make([]domain.PostFeedItem, 0)
	for rows.Next() {
		var item domain.PostFeedItem
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.Post.Type, &item.Content, &item.MediaURL, &item.ImageURL,
			&item.IsProject, &item.ProjectID,
			&item.LikeCount, &item.CommentCount,
			&item.CreatedAt, &item.UpdatedAt,
			&item.AuthorEmail, &item.IsLiked,
			&item.Type, &item.IsJoined, &item.IsFollowing,
		); err != nil {
			return nil, fmt.Errorf("failed to scan feed item: %w", err)
		}
		// Keep embedded struct consistent for JSON.
		item.Post.Type = item.Type
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feed: %w", err)
	}

	return items, nil
}

// GetByUserID retrieves all posts by a specific user.
func (r *PostgresPostRepository) GetByUserID(userID string, limit, offset int) ([]domain.PostFeedItem, error) {
	query := `
		SELECT
			p.id, p.user_id, p.type, p.content, p.media_url, p.image_url, p.is_project,
			CASE WHEN p.type = 'project' OR p.is_project THEN COALESCE(p.project_id, p.id) ELSE NULL END AS project_id,
			p.like_count, p.comment_count,
			p.created_at, p.updated_at,
			u.email AS author_email,
			FALSE AS is_liked,
			CASE WHEN p.type IS NOT NULL AND p.type <> '' THEN p.type
			     WHEN p.is_project THEN 'project' ELSE 'text' END AS type,
			FALSE AS is_joined,
			FALSE AS is_following
		FROM posts p
		INNER JOIN users u ON u.id = p.user_id
		WHERE p.user_id = $1
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get user posts: %w", err)
	}
	defer rows.Close()

	items := make([]domain.PostFeedItem, 0)
	for rows.Next() {
		var item domain.PostFeedItem
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.Post.Type, &item.Content, &item.MediaURL, &item.ImageURL,
			&item.IsProject, &item.ProjectID,
			&item.LikeCount, &item.CommentCount,
			&item.CreatedAt, &item.UpdatedAt,
			&item.AuthorEmail, &item.IsLiked,
			&item.Type, &item.IsJoined, &item.IsFollowing,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user post: %w", err)
		}
		item.Post.Type = item.Type
		items = append(items, item)
	}

	return items, nil
}

// GetByCircle retrieves paginated posts within a specific circle.
func (r *PostgresPostRepository) GetByCircle(circleID string, userID string, limit, offset int) ([]domain.PostFeedItem, error) {
	query := `
		SELECT
			p.id, p.user_id, p.type, p.content, p.media_url, p.image_url, p.is_project,
			CASE WHEN p.type = 'project' OR p.is_project THEN COALESCE(p.project_id, p.id) ELSE NULL END AS project_id,
			p.like_count, p.comment_count,
			p.created_at, p.updated_at,
			u.email AS author_email,
			EXISTS(
				SELECT 1 FROM post_likes pl
				WHERE pl.post_id = p.id AND pl.user_id = $2
			) AS is_liked
			,
			CASE WHEN p.type IS NOT NULL AND p.type <> '' THEN p.type
			     WHEN p.is_project THEN 'project' ELSE 'text' END AS type,
			EXISTS(
				SELECT 1 FROM project_members pm
				WHERE pm.project_id = COALESCE(p.project_id, p.id) AND pm.user_id = $2
			) AS is_joined,
			EXISTS(
				SELECT 1 FROM followers f
				WHERE f.follower_id = $2 AND f.following_id = p.user_id
			) AS is_following
		FROM posts p
		INNER JOIN users u ON u.id = p.user_id
		WHERE p.circle_id = $1
		ORDER BY p.created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.Query(query, circleID, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get circle posts: %w", err)
	}
	defer rows.Close()

	items := make([]domain.PostFeedItem, 0)
	for rows.Next() {
		var item domain.PostFeedItem
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.Post.Type, &item.Content, &item.MediaURL, &item.ImageURL,
			&item.IsProject, &item.ProjectID,
			&item.LikeCount, &item.CommentCount,
			&item.CreatedAt, &item.UpdatedAt,
			&item.AuthorEmail, &item.IsLiked,
			&item.Type, &item.IsJoined, &item.IsFollowing,
		); err != nil {
			return nil, fmt.Errorf("failed to scan circle post: %w", err)
		}
		item.Post.Type = item.Type
		items = append(items, item)
	}

	return items, nil
}

func (r *PostgresPostRepository) GetProjectRoots(userID string, limit int) ([]domain.ProjectSummary, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := r.db.Query(
		`SELECT id, content, created_at
		 FROM posts
		 WHERE user_id = $1
		   AND (type = 'project' OR is_project = TRUE)
		   AND COALESCE(project_id, id) = id
		 ORDER BY created_at DESC
		 LIMIT $2`,
		userID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	out := make([]domain.ProjectSummary, 0)
	for rows.Next() {
		var p domain.ProjectSummary
		if err := rows.Scan(&p.ID, &p.Content, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Update modifies an existing post's content and image_url.
func (r *PostgresPostRepository) Update(post *domain.Post) error {
	query := `
		UPDATE posts
		SET content = $1, image_url = $2, updated_at = $3
		WHERE id = $4
	`

	post.UpdatedAt = time.Now()
	result, err := r.db.Exec(query, post.Content, post.ImageURL, post.UpdatedAt, post.ID)
	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("post not found")
	}

	return nil
}

// Delete removes a post by ID.
func (r *PostgresPostRepository) Delete(postID string) error {
	query := `DELETE FROM posts WHERE id = $1`

	result, err := r.db.Exec(query, postID)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("post not found")
	}

	return nil
}

// SetCategories replaces all categories for a post atomically
// using a database transaction (DELETE all + INSERT new).
func (r *PostgresPostRepository) SetCategories(postID string, categories []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing categories
	if _, err := tx.Exec(`DELETE FROM post_categories WHERE post_id = $1`, postID); err != nil {
		return fmt.Errorf("failed to delete categories: %w", err)
	}

	// Insert new categories
	if len(categories) > 0 {
		insertQuery := `INSERT INTO post_categories (id, post_id, category_name, created_at) VALUES ($1, $2, $3, $4)`
		now := time.Now()
		for _, cat := range categories {
			if _, err := tx.Exec(insertQuery, uuid.New().String(), postID, cat, now); err != nil {
				return fmt.Errorf("failed to insert category '%s': %w", cat, err)
			}
		}
	}

	return tx.Commit()
}

// GetCategories returns all category names for a post.
func (r *PostgresPostRepository) GetCategories(postID string) ([]string, error) {
	query := `SELECT category_name FROM post_categories WHERE post_id = $1 ORDER BY created_at ASC`

	rows, err := r.db.Query(query, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer rows.Close()

	categories := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, name)
	}

	return categories, nil
}

// ToggleLike adds a like if not exists, removes if exists.
// Returns true if liked after toggle, false if unliked.
// Atomically updates the like_count on the post using a transaction.
func (r *PostgresPostRepository) ToggleLike(userID, postID string) (bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if like already exists
	var existingID string
	checkQuery := `SELECT id FROM post_likes WHERE user_id = $1 AND post_id = $2`
	err = tx.QueryRow(checkQuery, userID, postID).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Like does not exist — add it
		insertQuery := `INSERT INTO post_likes (id, user_id, post_id, created_at) VALUES ($1, $2, $3, $4)`
		if _, err := tx.Exec(insertQuery, uuid.New().String(), userID, postID, time.Now()); err != nil {
			return false, fmt.Errorf("failed to insert like: %w", err)
		}

		// Increment like_count
		if _, err := tx.Exec(`UPDATE posts SET like_count = like_count + 1 WHERE id = $1`, postID); err != nil {
			return false, fmt.Errorf("failed to increment like count: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("failed to commit: %w", err)
		}
		return true, nil
	}

	if err != nil {
		return false, fmt.Errorf("failed to check like: %w", err)
	}

	// Like exists — remove it
	if _, err := tx.Exec(`DELETE FROM post_likes WHERE id = $1`, existingID); err != nil {
		return false, fmt.Errorf("failed to delete like: %w", err)
	}

	// Decrement like_count (minimum 0)
	if _, err := tx.Exec(`UPDATE posts SET like_count = GREATEST(like_count - 1, 0) WHERE id = $1`, postID); err != nil {
		return false, fmt.Errorf("failed to decrement like count: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("failed to commit: %w", err)
	}
	return false, nil
}

// IncrementCommentCount atomically increments comment_count.
func (r *PostgresPostRepository) IncrementCommentCount(postID string) error {
	_, err := r.db.Exec(`UPDATE posts SET comment_count = comment_count + 1 WHERE id = $1`, postID)
	if err != nil {
		return fmt.Errorf("failed to increment comment count: %w", err)
	}
	return nil
}

// DecrementCommentCount atomically decrements comment_count (minimum 0).
func (r *PostgresPostRepository) DecrementCommentCount(postID string) error {
	_, err := r.db.Exec(`UPDATE posts SET comment_count = GREATEST(comment_count - 1, 0) WHERE id = $1`, postID)
	if err != nil {
		return fmt.Errorf("failed to decrement comment count: %w", err)
	}
	return nil
}
