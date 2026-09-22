package infrastructure

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type PasswordService interface {
	HashPassword(password string) (string, error)
	VerifyPassword(hashedPassword, password string) error
}

type PasswordServiceImpl struct{}

func NewPasswordService() PasswordService {
	return &PasswordServiceImpl{}
}

func (p *PasswordServiceImpl) HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedPassword), nil
}

func (p *PasswordServiceImpl) VerifyPassword(hashedPassword, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}
	return nil
}
