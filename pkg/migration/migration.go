package migration

import (
	"database/sql"
	"fmt"
	"log"
)

// RunMigrations executes all database migrations
func RunMigrations(db *sql.DB) error {
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(36) PRIMARY KEY,
		email VARCHAR(254) UNIQUE NOT NULL,
		password VARCHAR(255),
		provider VARCHAR(50),
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);
	`

	_, err := db.Exec(createUsersTable)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Devices: stores user FCM tokens for push notifications
	createUserDevices := `
	CREATE TABLE IF NOT EXISTS user_devices (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		fcm_token VARCHAR(255) UNIQUE NOT NULL,
		device_type VARCHAR(20) NOT NULL DEFAULT 'unknown',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		CONSTRAINT uq_user_fcm_token UNIQUE (user_id, fcm_token)
	);
	`
	if _, err = db.Exec(createUserDevices); err != nil {
		return fmt.Errorf("failed to create user_devices table: %w", err)
	}

	// Focus Mode: settings table (1:1 with users)
	createFocusModeSettings := `
	CREATE TABLE IF NOT EXISTS focus_mode_settings (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		is_enabled BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		CONSTRAINT uq_focus_mode_user UNIQUE (user_id)
	);
	`

	_, err = db.Exec(createFocusModeSettings)
	if err != nil {
		return fmt.Errorf("failed to create focus_mode_settings table: %w", err)
	}

	// Focus Mode: blocked categories table (1:N with users)
	createFocusModeCategories := `
	CREATE TABLE IF NOT EXISTS focus_mode_blocked_categories (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		category_name VARCHAR(100) NOT NULL,
		created_at TIMESTAMP NOT NULL,
		CONSTRAINT uq_user_category UNIQUE (user_id, category_name)
	);
	`

	_, err = db.Exec(createFocusModeCategories)
	if err != nil {
		return fmt.Errorf("failed to create focus_mode_blocked_categories table: %w", err)
	}

	// Create indexes for fast lookups
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_focus_settings_user ON focus_mode_settings(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_focus_blocked_user ON focus_mode_blocked_categories(user_id)`,
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// ============================
	// Circles
	// ============================
	createCircles := `
	CREATE TABLE IF NOT EXISTS circles (
		id              VARCHAR(36) PRIMARY KEY,
		owner_id        VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name            VARCHAR(50) NOT NULL,
		description     VARCHAR(255),
		avatar_url      TEXT,
		member_count    INTEGER NOT NULL DEFAULT 1,
		created_at      TIMESTAMP NOT NULL,
		updated_at      TIMESTAMP NOT NULL,
		CONSTRAINT chk_circle_name_not_empty CHECK (char_length(name) >= 1)
	);
	`
	if _, err = db.Exec(createCircles); err != nil {
		return fmt.Errorf("failed to create circles table: %w", err)
	}

	// ============================
	// Circle Members (Junction)
	// ============================
	createCircleMembers := `
	CREATE TABLE IF NOT EXISTS circle_members (
		id          VARCHAR(36) PRIMARY KEY,
		circle_id   VARCHAR(36) NOT NULL REFERENCES circles(id) ON DELETE CASCADE,
		user_id     VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		role        VARCHAR(20) NOT NULL DEFAULT 'member',
		joined_at   TIMESTAMP NOT NULL,
		CONSTRAINT uq_circle_member UNIQUE (circle_id, user_id),
		CONSTRAINT chk_member_role CHECK (role IN ('owner', 'member'))
	);
	`
	if _, err = db.Exec(createCircleMembers); err != nil {
		return fmt.Errorf("failed to create circle_members table: %w", err)
	}

	// ============================
	// Posts: main posts table (updated with visibility and circle_id)
	// ============================
	createPosts := `
	CREATE TABLE IF NOT EXISTS posts (
		id            VARCHAR(36) PRIMARY KEY,
		user_id       VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		content       TEXT,
		image_url     TEXT,
		visibility    VARCHAR(10) NOT NULL DEFAULT 'public',
		circle_id     VARCHAR(36) REFERENCES circles(id) ON DELETE SET NULL,
		like_count    INT NOT NULL DEFAULT 0,
		comment_count INT NOT NULL DEFAULT 0,
		created_at    TIMESTAMP NOT NULL,
		updated_at    TIMESTAMP NOT NULL,
		CONSTRAINT chk_visibility CHECK (visibility IN ('public', 'circle')),
		CONSTRAINT chk_circle_visibility_integrity CHECK (
			(visibility = 'circle' AND circle_id IS NOT NULL) OR
			(visibility = 'public' AND circle_id IS NULL)
		)
	);
	`
	if _, err = db.Exec(createPosts); err != nil {
		return fmt.Errorf("failed to create posts table: %w", err)
	}

	// Posts: category tags (junction table — supports multi-category per post)
	createPostCategories := `
	CREATE TABLE IF NOT EXISTS post_categories (
		id            VARCHAR(36) PRIMARY KEY,
		post_id       VARCHAR(36) NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
		category_name VARCHAR(100) NOT NULL,
		created_at    TIMESTAMP NOT NULL,
		CONSTRAINT uq_post_category UNIQUE (post_id, category_name)
	);
	`
	if _, err = db.Exec(createPostCategories); err != nil {
		return fmt.Errorf("failed to create post_categories table: %w", err)
	}

	// Posts: likes (each user can like a post once)
	createPostLikes := `
	CREATE TABLE IF NOT EXISTS post_likes (
		id         VARCHAR(36) PRIMARY KEY,
		user_id    VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		post_id    VARCHAR(36) NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
		created_at TIMESTAMP NOT NULL,
		CONSTRAINT uq_user_post_like UNIQUE (user_id, post_id)
	);
	`
	if _, err = db.Exec(createPostLikes); err != nil {
		return fmt.Errorf("failed to create post_likes table: %w", err)
	}

	// Comments: per post
	createComments := `
	CREATE TABLE IF NOT EXISTS comments (
		id         VARCHAR(36) PRIMARY KEY,
		post_id    VARCHAR(36) NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
		user_id    VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		content    TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		CONSTRAINT chk_comment_not_empty CHECK (char_length(content) >= 1)
	);
	`
	if _, err = db.Exec(createComments); err != nil {
		return fmt.Errorf("failed to create comments table: %w", err)
	}

	// Indexes for posts module
	postIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_circles_owner ON circles(owner_id)`,
		`CREATE INDEX IF NOT EXISTS idx_circle_members_circle ON circle_members(circle_id)`,
		`CREATE INDEX IF NOT EXISTS idx_circle_members_user ON circle_members(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_circle_id ON posts(circle_id)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_visibility ON posts(visibility, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_post_categories_post_id ON post_categories(post_id)`,
		`CREATE INDEX IF NOT EXISTS idx_post_categories_name ON post_categories(category_name)`,
		`CREATE INDEX IF NOT EXISTS idx_post_likes_post_id ON post_likes(post_id)`,
		`CREATE INDEX IF NOT EXISTS idx_post_likes_user_id ON post_likes(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments(post_id, created_at)`,
	}

	for _, idx := range postIndexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create post index: %w", err)
		}
	}

	postAlters := []string{
		`ALTER TABLE posts ADD COLUMN IF NOT EXISTS is_project BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE posts ADD COLUMN IF NOT EXISTS project_id VARCHAR(36)`,
		`ALTER TABLE posts ADD COLUMN IF NOT EXISTS type VARCHAR(16) NOT NULL DEFAULT 'text'`,
		`ALTER TABLE posts ADD COLUMN IF NOT EXISTS media_url TEXT`,
	}
	for _, q := range postAlters {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("failed to alter posts (is_project/project_id): %w", err)
		}
	}

	// Backfill type/media_url for legacy rows (safe to run repeatedly).
	_, _ = db.Exec(`
		UPDATE posts
		SET
			type = CASE WHEN is_project THEN 'project' ELSE 'text' END
		WHERE type IS NULL OR type = '';
	`)
	_, _ = db.Exec(`
		UPDATE posts
		SET media_url = image_url
		WHERE media_url IS NULL AND image_url IS NOT NULL;
	`)

	// =========================================
	// Projects (lightweight): project_members + followers
	// =========================================
	createProjectMembers := `
	CREATE TABLE IF NOT EXISTS project_members (
		id          VARCHAR(36) PRIMARY KEY,
		user_id     VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		project_id  VARCHAR(36) NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
		created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		CONSTRAINT uq_project_member UNIQUE (user_id, project_id)
	);
	`
	if _, err = db.Exec(createProjectMembers); err != nil {
		return fmt.Errorf("failed to create project_members table: %w", err)
	}

	createFollowers := `
	CREATE TABLE IF NOT EXISTS followers (
		id            VARCHAR(36) PRIMARY KEY,
		follower_id   VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		following_id  VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		CONSTRAINT uq_follower_pair UNIQUE (follower_id, following_id),
		CONSTRAINT chk_no_self_follow CHECK (follower_id <> following_id)
	);
	`
	if _, err = db.Exec(createFollowers); err != nil {
		return fmt.Errorf("failed to create followers table: %w", err)
	}

	projectFollowIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_project_members_project ON project_members(project_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_project_members_user ON project_members(user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_followers_following ON followers(following_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_followers_follower ON followers(follower_id, created_at DESC)`,
	}
	for _, idx := range projectFollowIndexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create project/follow index: %w", err)
		}
	}

	// =========================================
	// Daily Feed Limit Tables
	// =========================================

	// Table 1: feed_limit_settings — stores user's daily limit preferences (1:1 with users)
	createFeedLimitSettings := `
	CREATE TABLE IF NOT EXISTS feed_limit_settings (
		id                          VARCHAR(36) PRIMARY KEY,
		user_id                     VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		is_enabled                  BOOLEAN NOT NULL DEFAULT FALSE,
		max_posts_per_day           INTEGER,
		max_scroll_minutes          INTEGER,
		max_overrides_per_day       INTEGER NOT NULL DEFAULT 3,
		override_duration_minutes   INTEGER NOT NULL DEFAULT 15,
		created_at                  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at                  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

		CONSTRAINT uq_feed_limit_user UNIQUE (user_id),
		CONSTRAINT chk_max_posts CHECK (max_posts_per_day IS NULL OR max_posts_per_day > 0),
		CONSTRAINT chk_max_scroll CHECK (max_scroll_minutes IS NULL OR max_scroll_minutes > 0),
		CONSTRAINT chk_max_overrides CHECK (max_overrides_per_day >= 0 AND max_overrides_per_day <= 10)
	);
	`
	if _, err = db.Exec(createFeedLimitSettings); err != nil {
		return fmt.Errorf("failed to create feed_limit_settings table: %w", err)
	}

	// Table 2: feed_usage_daily — tracks daily consumption per user (1:N with users, keyed by date)
	createFeedUsageDaily := `
	CREATE TABLE IF NOT EXISTS feed_usage_daily (
		id                  VARCHAR(36) PRIMARY KEY,
		user_id             VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		usage_date          DATE NOT NULL DEFAULT CURRENT_DATE,
		posts_viewed        INTEGER NOT NULL DEFAULT 0,
		scroll_minutes      INTEGER NOT NULL DEFAULT 0,
		limit_reached_at    TIMESTAMP WITH TIME ZONE,
		is_limit_reached    BOOLEAN NOT NULL DEFAULT FALSE,
		created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

		CONSTRAINT uq_user_usage_date UNIQUE (user_id, usage_date),
		CONSTRAINT chk_posts_viewed CHECK (posts_viewed >= 0),
		CONSTRAINT chk_scroll_minutes CHECK (scroll_minutes >= 0)
	);
	`
	if _, err = db.Exec(createFeedUsageDaily); err != nil {
		return fmt.Errorf("failed to create feed_usage_daily table: %w", err)
	}

	// Table 3: feed_limit_overrides — audit log of each snooze override (1:N with users)
	createFeedLimitOverrides := `
	CREATE TABLE IF NOT EXISTS feed_limit_overrides (
		id              VARCHAR(36) PRIMARY KEY,
		user_id         VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		override_date   DATE NOT NULL DEFAULT CURRENT_DATE,
		activated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		expires_at      TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	);
	`
	if _, err = db.Exec(createFeedLimitOverrides); err != nil {
		return fmt.Errorf("failed to create feed_limit_overrides table: %w", err)
	}

	// Indexes for feed limit tables
	feedLimitIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_feed_limit_user ON feed_limit_settings(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_feed_usage_user_date ON feed_usage_daily(user_id, usage_date)`,
		`CREATE INDEX IF NOT EXISTS idx_feed_override_user_date ON feed_limit_overrides(user_id, override_date)`,
	}
	for _, idx := range feedLimitIndexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create feed limit index: %w", err)
		}
	}

	// =========================================
	// Social: Connections + Stories (lightweight)
	// =========================================
	createConnections := `
	CREATE TABLE IF NOT EXISTS connections (
		id            VARCHAR(36) PRIMARY KEY,
		from_email    VARCHAR(254) NOT NULL,
		to_email      VARCHAR(254) NOT NULL,
		status        VARCHAR(20) NOT NULL DEFAULT 'pending',
		created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		CONSTRAINT chk_connection_status CHECK (status IN ('pending', 'accepted', 'rejected')),
		CONSTRAINT uq_connection_pair UNIQUE (from_email, to_email)
	);
	`
	if _, err = db.Exec(createConnections); err != nil {
		return fmt.Errorf("failed to create connections table: %w", err)
	}

	createStories := `
	CREATE TABLE IF NOT EXISTS stories (
		id            VARCHAR(36) PRIMARY KEY,
		author_email  VARCHAR(254) NOT NULL,
		media_url     TEXT NOT NULL,
		media_type    VARCHAR(10) NOT NULL DEFAULT 'image',
		caption       TEXT,
		created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		expires_at    TIMESTAMP WITH TIME ZONE NOT NULL,
		CONSTRAINT chk_story_media_type CHECK (media_type IN ('image', 'video'))
	);
	`
	if _, err = db.Exec(createStories); err != nil {
		return fmt.Errorf("failed to create stories table: %w", err)
	}

	socialIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_connections_from ON connections(from_email, status, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_connections_to ON connections(to_email, status, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_stories_author ON stories(author_email, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_stories_expires ON stories(expires_at)`,
	}
	for _, idx := range socialIndexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create social index: %w", err)
		}
	}

	// =========================================
	// Refresh Tokens (for JWT refresh flow)
	// =========================================
	createRefreshTokens := `
	CREATE TABLE IF NOT EXISTS refresh_tokens (
		id              VARCHAR(36) PRIMARY KEY,
		user_id         VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		token_hash      VARCHAR(64) UNIQUE NOT NULL,
		device_info     VARCHAR(255) NOT NULL DEFAULT 'unknown',
		expires_at      TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	);
	`
	if _, err = db.Exec(createRefreshTokens); err != nil {
		return fmt.Errorf("failed to create refresh_tokens table: %w", err)
	}

	refreshIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens(token_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON refresh_tokens(expires_at)`,
	}
	for _, idx := range refreshIndexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create refresh token index: %w", err)
		}
	}

	// Users: Firebase Auth linking + email verification flag
	alterUserFirebase := []string{
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS firebase_uid VARCHAR(128)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS users_firebase_uid_key ON users(firebase_uid) WHERE firebase_uid IS NOT NULL`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT true`,
	}
	for _, q := range alterUserFirebase {
		if _, err = db.Exec(q); err != nil {
			return fmt.Errorf("failed to alter users (firebase / email_verified): %w", err)
		}
	}
	log.Println("Migrations completed successfully")
	return nil
}
