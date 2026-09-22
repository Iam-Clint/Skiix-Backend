package domain

import "time"

type Story struct {
	ID          string
	AuthorEmail string
	MediaURL    string
	MediaType   string // image|video
	Caption     *string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

type Connection struct {
	ID        string
	FromEmail string
	ToEmail   string
	Status    string // pending|accepted|rejected
	CreatedAt time.Time
	UpdatedAt time.Time
}

type StoryRepository interface {
	Create(story *Story) error
	// Feed returns unexpired stories for the given user email:
	// includes self + accepted connections (either direction).
	Feed(forEmail string, limit int) ([]Story, error)
}

type ConnectionRepository interface {
	CreateRequest(fromEmail, toEmail string) (*Connection, error)
	Accept(id string) error
	List(forEmail string, limit, offset int) ([]Connection, error)
	// Delete removes a connection row if the given email is from_email or to_email.
	Delete(id string, participantEmail string) error
}
