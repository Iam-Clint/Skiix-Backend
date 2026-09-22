package main

// @title           Skiix API
// @version         1.0
// @description     Skiix authentication API with JWT and OAuth2 support
// @host            localhost:8080
// @basePath         /
// @schemes          http https
// @securityDefinitions.apikey Bearer
// @in               header
// @name             Authorization
// @description      Type "Bearer" followed by a space and JWT token
// @x-tokenName      Authorization

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/google"
	_ "github.com/skiix-backend/docs"
	httpHandler "github.com/skiix-backend/internal/delivery/http"
	"github.com/skiix-backend/internal/infrastructure"
	"github.com/skiix-backend/internal/repository"
	"github.com/skiix-backend/internal/usecase"
	"github.com/skiix-backend/pkg/migration"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	// Load .env configuration
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: no .env file found")
	}

	baseURL := os.Getenv("BASE_URL")

	// Database configuration: DATABASE_URL (full string) wins over DB_* for Supabase pooler, etc.
	dbConfig := infrastructure.DatabaseConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	// Go Authentication OAuth configuration
	goth.UseProviders(
		google.New(
			os.Getenv("GOOGLE_CLIENT_ID"),
			os.Getenv("GOOGLE_CLIENT_SECRET"),
			fmt.Sprintf("%s/auth/google/callback", baseURL),
			"email", "profile",
		),
	)

	// Connect to database
	var db *sql.DB
	var err error
	if dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL")); dbURL != "" {
		log.Println("database: using DATABASE_URL (DB_* fields ignored)")
		db, err = infrastructure.NewPostgresConnectionFromURL(dbURL)
	} else {
		db, err = infrastructure.NewPostgresConnection(dbConfig)
	}
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer infrastructure.CloseDatabase(db)

	// Run migrations
	if err := migration.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize repositories and services
	userRepo := repository.NewPostgresUserRepository(db)
	httpHandler.InitSupabaseAuth(userRepo)
	focusModeRepo := repository.NewPostgresFocusModeRepository(db)
	postRepo := repository.NewPostgresPostRepository(db)
	commentRepo := repository.NewPostgresCommentRepository(db)
	circleRepo := repository.NewPostgresCircleRepository(db)
	feedLimitRepo := repository.NewPostgresFeedLimitRepository(db)
	deviceRepo := repository.NewPostgresDeviceRepository(db)
	refreshTokenRepo := repository.NewPostgresRefreshTokenRepository(db)
	storyRepo := repository.NewPostgresStoryRepository(db)
	connectionRepo := repository.NewPostgresConnectionRepository(db)
	projectMemberRepo := repository.NewPostgresProjectMemberRepository(db)
	followerRepo := repository.NewPostgresFollowerRepository(db)
	jwtService := infrastructure.NewJWTService()
	passwordService := infrastructure.NewPasswordService()
	cloudinaryClient, cloudErr := infrastructure.NewCloudinary()
	if cloudErr != nil {
		log.Printf("Warning: Cloudinary not configured (%v). Upload endpoint will fail until env vars are set.", cloudErr)
	}

	firebaseService := infrastructure.NewFirebaseService()
	credentialsPath := os.Getenv("FIREBASE_CREDENTIALS_PATH")
	if credentialsPath != "" {
		if err := firebaseService.Init(credentialsPath); err != nil {
			log.Printf("Failed to initialize Firebase Admin SDK: %v", err)
		}
	} else {
		log.Println("Warning: FIREBASE_CREDENTIALS_PATH not set. Push notifications will be disabled.")
	}

	// Initialize use cases
	authUsecase := usecase.NewAuthUsecase(userRepo, jwtService, passwordService, refreshTokenRepo)
	focusModeUsecase := usecase.NewFocusModeUsecase(focusModeRepo)
	circleUsecase := usecase.NewCircleUsecase(circleRepo)
	postUsecase := usecase.NewPostUsecase(postRepo, commentRepo, circleRepo, deviceRepo, firebaseService)
	feedLimitUsecase := usecase.NewFeedLimitUsecase(feedLimitRepo)
	deviceUsecase := usecase.NewDeviceUsecase(deviceRepo)

	// Setup router
	router := gin.Default()

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/docs", func(c *gin.Context) {
		c.Redirect(302, "/swagger/index.html")
	})

	httpHandler.RegisterAuthRoutes(router, authUsecase, jwtService)
	httpHandler.RegisterFocusModeRoutes(router, focusModeUsecase, jwtService)
	httpHandler.RegisterPostRoutes(router, postUsecase, userRepo, jwtService)
	httpHandler.RegisterCircleRoutes(router, circleUsecase, jwtService)
	httpHandler.RegisterFeedLimitRoutes(router, feedLimitUsecase, jwtService)
	httpHandler.RegisterDeviceRoutes(router, deviceUsecase, jwtService)
	httpHandler.RegisterChatRoutes(router, firebaseService, jwtService)
	httpHandler.RegisterUploadRoutes(router, jwtService, cloudinaryClient)
	httpHandler.RegisterSocialRoutes(router, storyRepo, connectionRepo)
	httpHandler.RegisterFollowProjectRoutes(router, userRepo, followerRepo, projectMemberRepo, postRepo)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
