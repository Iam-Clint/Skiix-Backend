package domain

import "time"

type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Password      *string   `json:"-"`
	Provider      *string   `json:"provider,omitempty"`
	FirebaseUID   *string   `json:"-"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type UserRepository interface {
	Create(user *User) error
	FindByEmail(email string) (*User, error)
	FindByID(id string) (*User, error)
	FindByFirebaseUID(uid string) (*User, error)
	UpdateUserFirebaseInfo(user *User) error
}
