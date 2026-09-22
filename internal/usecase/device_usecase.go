package usecase

import (
	"github.com/skiix-backend/internal/domain"
)

type deviceUsecase struct {
	deviceRepo domain.DeviceRepository
}

func NewDeviceUsecase(repo domain.DeviceRepository) domain.DeviceUsecase {
	return &deviceUsecase{deviceRepo: repo}
}

func (u *deviceUsecase) RegisterDeviceToken(userID, fcmToken, deviceType string) error {
	device := &domain.UserDevice{
		UserID:     userID,
		FCMToken:   fcmToken,
		DeviceType: deviceType,
	}
	return u.deviceRepo.SaveToken(device)
}

func (u *deviceUsecase) UnregisterDeviceToken(fcmToken string) error {
	return u.deviceRepo.RemoveToken(fcmToken)
}
