package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/skiix-backend/internal/domain"
)

type PostgresDeviceRepository struct {
	db *sql.DB
}

func NewPostgresDeviceRepository(db *sql.DB) domain.DeviceRepository {
	return &PostgresDeviceRepository{db: db}
}

func (r *PostgresDeviceRepository) SaveToken(device *domain.UserDevice) error {
	if device.ID == "" {
		device.ID = uuid.New().String()
	}
	device.CreatedAt = time.Now()
	device.UpdatedAt = time.Now()

	// ON CONFLICT DO UPDATE so that if a token exists for this user, we just update the timestamp
	query := `
		INSERT INTO user_devices (id, user_id, fcm_token, device_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, fcm_token) 
		DO UPDATE SET updated_at = $6, device_type = $4
	`
	_, err := r.db.Exec(query, device.ID, device.UserID, device.FCMToken, device.DeviceType, device.CreatedAt, device.UpdatedAt)
	return err
}

func (r *PostgresDeviceRepository) GetTokensByUserID(userID string) ([]string, error) {
	query := `SELECT fcm_token FROM user_devices WHERE user_id = $1`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func (r *PostgresDeviceRepository) RemoveToken(fcmToken string) error {
	query := `DELETE FROM user_devices WHERE fcm_token = $1`
	_, err := r.db.Exec(query, fcmToken)
	return err
}
