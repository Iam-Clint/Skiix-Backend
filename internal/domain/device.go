package domain

import "time"

type UserDevice struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	FCMToken   string    `json:"fcm_token"`
	DeviceType string    `json:"device_type"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type DeviceRepository interface {
	SaveToken(device *UserDevice) error
	GetTokensByUserID(userID string) ([]string, error)
	RemoveToken(fcmToken string) error
}

type DeviceUsecase interface {
	RegisterDeviceToken(userID, fcmToken, deviceType string) error
	UnregisterDeviceToken(fcmToken string) error
}

type FirebaseService interface {
	Init(credentialsFilePath string) error
	SendPushNotification(tokens []string, title string, body string, data map[string]string) error
	GenerateCustomToken(userID string) (string, error)
}
