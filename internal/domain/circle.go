package domain

import "time"

// Circle represents a user-created private group.
type Circle struct {
	ID          string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	OwnerID     string    `json:"owner_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name        string    `json:"name" example:"Close Friends"`
	Description *string   `json:"description,omitempty" example:"My inner circle"`
	AvatarURL   *string   `json:"avatar_url,omitempty" example:"https://cdn.skiix.com/avatar.jpg"`
	MemberCount int       `json:"member_count" example:"1"`
	CreatedAt   time.Time `json:"created_at" example:"2026-04-10T12:00:00Z"`
	UpdatedAt   time.Time `json:"updated_at" example:"2026-04-10T12:00:00Z"`
}

// CircleMember represents the junction between Users and Circles.
type CircleMember struct {
	ID       string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440002"`
	CircleID string    `json:"circle_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID   string    `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Role     string    `json:"role" example:"owner"`
	JoinedAt time.Time `json:"joined_at" example:"2026-04-10T12:00:00Z"`
}

// CreateCircleRequest is the payload for creating a circle.
type CreateCircleRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=50"`
	Description *string `json:"description" binding:"omitempty,max=255"`
}

// UpdateCircleRequest is the payload for updating a circle.
type UpdateCircleRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=50"`
	Description *string `json:"description" binding:"omitempty,max=255"`
}

// AddMembersRequest is the payload for adding members to a circle.
type AddMembersRequest struct {
	UserIDs []string `json:"user_ids" binding:"required,min=1,max=50"`
}

// CircleRepository defines the data access contract for Circles.
type CircleRepository interface {
	Create(circle *Circle) error
	GetByID(id string) (*Circle, error)
	GetByUser(userID string) ([]Circle, error)
	Update(circle *Circle) error
	Delete(id string) error
	AddMembers(circleID string, userIDs []string) error
	RemoveMember(circleID string, userID string) error
	IsMember(circleID string, userID string) (bool, error)
	GetMembers(circleID string, limit, offset int) ([]CircleMember, error)
	CountByOwner(ownerID string) (int, error)
}
