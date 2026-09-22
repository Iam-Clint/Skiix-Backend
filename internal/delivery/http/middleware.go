package http

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skiix-backend/internal/domain"
)

const UserContextKey = "user"

var (
	authUserRepo    domain.UserRepository
	supabaseURL     string
	supabaseAnonKey string
	supabaseHTTP    = &http.Client{Timeout: 8 * time.Second}
)

// InitSupabaseAuth wires up the Supabase fallback verification path used by
// JWTMiddleware. Call this once from main.go, before routes are registered.
//
// Why this exists: this backend has its own JWT auth (login/register/JWT_SECRET)
// completely separate from Supabase Auth, which is what the mobile app actually
// signs users in with. Without this, every protected endpoint rejects real users
// because their Supabase session token doesn't match this backend's own signing
// key. This adds a second verification path — if a token isn't one we issued
// ourselves, we ask Supabase directly whether it's valid.
func InitSupabaseAuth(userRepo domain.UserRepository) {
	authUserRepo = userRepo
	supabaseURL = strings.TrimRight(os.Getenv("SUPABASE_URL"), "/")
	supabaseAnonKey = os.Getenv("SUPABASE_ANON_KEY")
}

type supabaseUserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// verifySupabaseToken checks a token this backend didn't issue itself by
// asking Supabase's own Auth API whether it's a valid session. Returns nil
// if the token isn't a valid Supabase session (or Supabase isn't configured).
func verifySupabaseToken(token string) *domain.TokenPayload {
	if supabaseURL == "" || supabaseAnonKey == "" {
		return nil
	}

	req, err := http.NewRequest("GET", supabaseURL+"/auth/v1/user", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", supabaseAnonKey)

	resp, err := supabaseHTTP.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var su supabaseUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&su); err != nil || su.ID == "" {
		return nil
	}

	// Just-in-time provisioning: every foreign key in this backend (posts,
	// devices, focus_mode_settings, circles, etc.) points at users.id, so a
	// local row must exist the first time a Supabase-authenticated user hits
	// a protected route. This never touches the user's password — there
	// isn't one, Supabase owns that.
	if authUserRepo != nil {
		if _, err := authUserRepo.FindByID(su.ID); err != nil {
			provider := "supabase"
			_ = authUserRepo.Create(&domain.User{
				ID:            su.ID,
				Email:         su.Email,
				Provider:      &provider,
				EmailVerified: true,
			})
		}
	}

	return &domain.TokenPayload{UserID: su.ID, Email: su.Email}
}

func JWTMiddleware(jwtService domain.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Path 1: token this backend issued itself (legacy /auth/login, /auth/register).
		payload, err := jwtService.ValidateToken(tokenString)

		// Path 2: not ours — check if it's a real Supabase session token.
		if err != nil {
			payload = verifySupabaseToken(tokenString)
		}

		if payload == nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid token"})
			c.Abort()
			return
		}

		c.Set(UserContextKey, payload)
		c.Set("user_id", payload.UserID) // some handlers (post/circle/device/chat) read this key directly
		c.Next()
	}
}
