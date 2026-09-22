package http

import (
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/gin-gonic/gin"
	"github.com/skiix-backend/internal/domain"
	"github.com/skiix-backend/internal/usecase"
)

type ProfileResponse struct {
	ID            string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email         string  `json:"email" example:"user@example.com"`
	Provider      *string `json:"provider,omitempty" example:"local"`
	EmailVerified bool    `json:"email_verified"`
	CreatedAt     string  `json:"created_at" example:"2026-03-23T10:00:00Z"`
	UpdatedAt     string  `json:"updated_at" example:"2026-03-23T10:00:00Z"`
}

func RegisterAuthRoutes(
	router *gin.Engine,
	authUsecase usecase.AuthUsecase,
	jwtService domain.JWTService,
) {
	authHandler := NewAuthHandler(authUsecase)
	oauthHandler := NewOAuthHandler(authUsecase)

	auth := router.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/logout", JWTMiddleware(jwtService), authHandler.Logout)
		auth.GET("/profile", JWTMiddleware(jwtService), authHandler.GetProfile)
		auth.GET("/google", oauthHandler.GoogleLogin)
		auth.GET("/google/callback", oauthHandler.GoogleCallback)
	}

	// Public endpoints (no auth required)
	api := router.Group("/api/v1")
	{
		api.GET("/categories", GetCategories)
	}
}

// RegisterFocusModeRoutes sets up the Focus Mode API endpoints.
// All endpoints require JWT authentication because focus settings
// are user-specific and private.
func RegisterFocusModeRoutes(
	router *gin.Engine,
	focusModeUsecase usecase.FocusModeUsecase,
	jwtService domain.JWTService,
) {
	focusModeHandler := NewFocusModeHandler(focusModeUsecase)

	focusMode := router.Group("/api/v1/focus-mode")
	focusMode.Use(JWTMiddleware(jwtService))
	{
		focusMode.GET("", focusModeHandler.GetStatus)
		focusMode.PUT("", focusModeHandler.Toggle)
		focusMode.PUT("/categories", focusModeHandler.SetBlockedCategories)
	}
}

// RegisterPostRoutes sets up Posts, Feed, and Comments API endpoints.
// All endpoints require JWT authentication since all interactions are
// user-specific (ownership checks, like status computation, etc.).
func RegisterPostRoutes(
	router *gin.Engine,
	postUsecase usecase.PostUsecase,
	userRepo domain.UserRepository,
	jwtService domain.JWTService,
) {
	postHandler := NewPostHandler(postUsecase, userRepo)
	publicPostHandler := NewPublicPostsHandler(userRepo, postUsecase)

	// Public feed (read-only) so the app can show real posts without requiring
	// the legacy backend JWT. Create/like/etc remain protected.
	public := router.Group("/api/v1")
	{
		public.GET("/feed", postHandler.GetFeed)
		public.POST("/public/posts", publicPostHandler.CreatePublicPost)
	}

	api := router.Group("/api/v1")
	api.Use(JWTMiddleware(jwtService))
	{
		// Post CRUD
		api.POST("/posts", postHandler.CreatePost)
		api.GET("/posts/:post_id", postHandler.GetPost)
		api.PUT("/posts/:post_id", postHandler.UpdatePost)
		api.DELETE("/posts/:post_id", postHandler.DeletePost)

		// Like toggle (idempotent)
		api.POST("/posts/:post_id/like", postHandler.ToggleLike)

		// Comments
		api.POST("/posts/:post_id/comments", postHandler.AddComment)
		api.GET("/posts/:post_id/comments", postHandler.GetComments)
		api.DELETE("/posts/:post_id/comments/:comment_id", postHandler.DeleteComment)

		// User's own posts
		api.GET("/users/:user_id/posts", postHandler.GetUserPosts)

		// Circle feed
		api.GET("/circles/:id/posts", postHandler.GetCircleFeed)
	}
}

// RegisterUploadRoutes sets up media upload endpoints.
// Uploads are authenticated to prevent anonymous abuse.
func RegisterUploadRoutes(router *gin.Engine, jwtService domain.JWTService, cld *cloudinary.Cloudinary) {
	uploadHandler := NewUploadHandler(cld)

	// Keep both paths for compatibility:
	// - /api/v1/upload (backend convention)
	// - /api/upload (requested)
	api := router.Group("/api/v1")
	{
		api.POST("/upload", uploadHandler.UploadMedia)
	}
	legacy := router.Group("/api")
	{
		legacy.POST("/upload", uploadHandler.UploadMedia)
	}
}

// RegisterSocialRoutes sets up lightweight social endpoints for:
// - Stories (24h media + caption)
// - Connections (connect requests)
//
// Note: These endpoints are currently email-based and unauthenticated to unblock
// the mobile UI quickly. Next step is to switch them to Supabase JWT identity.
func RegisterSocialRoutes(router *gin.Engine, stories domain.StoryRepository, conns domain.ConnectionRepository) {
	h := NewSocialHandler(stories, conns)

	api := router.Group("/api/v1")
	{
		// Stories
		api.GET("/stories/feed", h.GetStoryFeed)
		api.POST("/stories", h.CreateStory)

		// Connections
		api.POST("/connections/request", h.RequestConnection)
		api.POST("/connections/accept", h.AcceptConnection)
		api.POST("/connections/remove", h.RemoveConnection)
		api.GET("/connections", h.ListConnections)
	}
}

// RegisterFollowProjectRoutes sets up join-project and follow actions.
// These endpoints are currently email-based and unauthenticated to match the mobile app's public feed flow.
func RegisterFollowProjectRoutes(
	router *gin.Engine,
	users domain.UserRepository,
	follows domain.FollowerRepository,
	members domain.ProjectMemberRepository,
	posts domain.PostRepository,
) {
	h := NewFollowHandler(users, follows, members, posts)
	api := router.Group("/api/v1")
	{
		api.POST("/projects/join", h.JoinProject)
		api.GET("/projects", h.ListProjects)
		api.POST("/follow", h.FollowUser)
	}
}

// RegisterCircleRoutes sets up Circle API endpoints.
func RegisterCircleRoutes(
	router *gin.Engine,
	circleUsecase *usecase.CircleUsecase,
	jwtService domain.JWTService,
) {
	circleHandler := NewCircleHandler(circleUsecase)

	circles := router.Group("/api/v1/circles")
	circles.Use(JWTMiddleware(jwtService))
	{
		circles.POST("", circleHandler.CreateCircle)
		circles.GET("", circleHandler.GetCircles)
		circles.PUT("/:id", circleHandler.UpdateCircle)
		circles.DELETE("/:id", circleHandler.DeleteCircle)

		circles.POST("/:id/members", circleHandler.AddMembers)
		circles.GET("/:id/members", circleHandler.GetMembers)
		circles.DELETE("/:id/members/:user_id", circleHandler.RemoveMember)
	}
}

// RegisterFeedLimitRoutes sets up the Daily Feed Limit API endpoints.
// All endpoints require JWT authentication since limits are user-specific.
func RegisterFeedLimitRoutes(
	router *gin.Engine,
	feedLimitUsecase usecase.FeedLimitUsecase,
	jwtService domain.JWTService,
) {
	h := NewFeedLimitHandler(feedLimitUsecase)

	feedLimit := router.Group("/api/v1/feed-limit")
	feedLimit.Use(JWTMiddleware(jwtService))
	{
		// Settings CRUD
		feedLimit.GET("", h.GetSettings)
		feedLimit.PUT("", h.UpdateSettings)
		feedLimit.DELETE("", h.DisableLimit)

		// Usage tracking
		feedLimit.GET("/usage", h.GetTodayUsage)
		feedLimit.POST("/track", h.TrackConsumption)
		feedLimit.POST("/override", h.ActivateOverride)
		feedLimit.GET("/history", h.GetUsageHistory)
	}
}

// RegisterDeviceRoutes sets up the Device / FCM Token API endpoints.
func RegisterDeviceRoutes(
	r *gin.Engine,
	uc domain.DeviceUsecase,
	jwtService domain.JWTService,
) {
	handler := NewDeviceHandler(uc)
	api := r.Group("/api/v1")
	api.Use(JWTMiddleware(jwtService))
	{
		devices := api.Group("/devices")
		{
			devices.POST("/register", handler.RegisterToken)
			devices.DELETE("/unregister", handler.UnregisterToken)
		}
	}
}

// RegisterChatRoutes sets up the Chat API endpoints.
func RegisterChatRoutes(
	r *gin.Engine,
	fs domain.FirebaseService,
	jwtService domain.JWTService,
) {
	handler := NewChatHandler(fs)
	api := r.Group("/api/v1")
	api.Use(JWTMiddleware(jwtService))
	{
		chat := api.Group("/chat")
		{
			chat.GET("/auth-token", handler.GetAuthToken)
		}
	}
}
