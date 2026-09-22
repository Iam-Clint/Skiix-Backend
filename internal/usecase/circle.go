package usecase

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/skiix-backend/internal/domain"
)

type CircleUsecase struct {
	circleRepo domain.CircleRepository
}

func NewCircleUsecase(circleRepo domain.CircleRepository) *CircleUsecase {
	return &CircleUsecase{
		circleRepo: circleRepo,
	}
}

func (u *CircleUsecase) CreateCircle(ownerID string, req domain.CreateCircleRequest) (*domain.Circle, error) {
	count, err := u.circleRepo.CountByOwner(ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing circles: %w", err)
	}
	if count >= 20 {
		return nil, errors.New("maximum circles limit reached (max 20)")
	}

	circle := &domain.Circle{
		ID:          uuid.New().String(),
		OwnerID:     ownerID,
		Name:        req.Name,
		Description: req.Description,
		MemberCount: 1, // At minimum, the owner is a member
	}

	if err := u.circleRepo.Create(circle); err != nil {
		return nil, err
	}

	return circle, nil
}

func (u *CircleUsecase) GetCirclesByUser(userID string) ([]domain.Circle, error) {
	return u.circleRepo.GetByUser(userID)
}

func (u *CircleUsecase) UpdateCircle(circleID string, ownerID string, req domain.UpdateCircleRequest) error {
	circle, err := u.circleRepo.GetByID(circleID)
	if err != nil {
		return err
	}
	if circle.OwnerID != ownerID {
		return errors.New("only the circle owner can perform this action")
	}

	circle.Name = req.Name
	circle.Description = req.Description

	return u.circleRepo.Update(circle)
}

func (u *CircleUsecase) DeleteCircle(circleID string, ownerID string) error {
	circle, err := u.circleRepo.GetByID(circleID)
	if err != nil {
		return err
	}
	if circle.OwnerID != ownerID {
		return errors.New("only the circle owner can perform this action")
	}
	return u.circleRepo.Delete(circleID)
}

func (u *CircleUsecase) AddMembers(circleID string, ownerID string, userIDs []string) error {
	circle, err := u.circleRepo.GetByID(circleID)
	if err != nil {
		return err
	}
	if circle.OwnerID != ownerID {
		return errors.New("only the circle owner can perform this action")
	}

	// Calculate if limits will be exceeded (simplified check to prevent >150)
	if circle.MemberCount+len(userIDs) > 150 {
		return errors.New("circle member limit reached (max 150)")
	}
	if len(userIDs) > 50 {
		return errors.New("max 50 members added per request")
	}

	return u.circleRepo.AddMembers(circleID, userIDs)
}

func (u *CircleUsecase) RemoveMember(circleID string, requesterID string, targetUserID string) error {
	circle, err := u.circleRepo.GetByID(circleID)
	if err != nil {
		return err
	}

	if requesterID != circle.OwnerID && requesterID != targetUserID {
		return errors.New("only the circle owner can perform this action")
	}

	if targetUserID == circle.OwnerID {
		return errors.New("owner cannot be removed, must delete circle instead")
	}

	return u.circleRepo.RemoveMember(circleID, targetUserID)
}

func (u *CircleUsecase) GetMembers(circleID string, requesterID string, limit, offset int) ([]domain.CircleMember, error) {
	isMember, err := u.circleRepo.IsMember(circleID, requesterID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("you are not a member of this circle")
	}

	if limit <= 0 || limit > 150 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	return u.circleRepo.GetMembers(circleID, limit, offset)
}
