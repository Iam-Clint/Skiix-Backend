package usecase

import (
	"testing"

	"github.com/skiix-backend/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateCircle_Success(t *testing.T) {
	circleRepo := new(MockCircleRepository)
	uc := NewCircleUsecase(circleRepo)

	circleRepo.On("CountByOwner", "owner-1").Return(0, nil)
	circleRepo.On("Create", mock.AnythingOfType("*domain.Circle")).Return(nil)

	req := domain.CreateCircleRequest{Name: "Close Friends"}
	circle, err := uc.CreateCircle("owner-1", req)

	assert.NoError(t, err)
	assert.NotNil(t, circle)
	assert.Equal(t, "Close Friends", circle.Name)
	assert.Equal(t, "owner-1", circle.OwnerID)
	circleRepo.AssertExpectations(t)
}

func TestCreateCircle_MaxLimitReached(t *testing.T) {
	circleRepo := new(MockCircleRepository)
	uc := NewCircleUsecase(circleRepo)

	circleRepo.On("CountByOwner", "owner-1").Return(20, nil)

	req := domain.CreateCircleRequest{Name: "Too many"}
	_, err := uc.CreateCircle("owner-1", req)

	assert.EqualError(t, err, "maximum circles limit reached (max 20)")
	circleRepo.AssertNotCalled(t, "Create")
}

func TestAddMembers_Success(t *testing.T) {
	circleRepo := new(MockCircleRepository)
	uc := NewCircleUsecase(circleRepo)

	circleRepo.On("GetByID", "c1").Return(&domain.Circle{ID: "c1", OwnerID: "owner-1", MemberCount: 5}, nil)
	circleRepo.On("AddMembers", "c1", []string{"user-2", "user-3"}).Return(nil)

	err := uc.AddMembers("c1", "owner-1", []string{"user-2", "user-3"})

	assert.NoError(t, err)
	circleRepo.AssertExpectations(t)
}

func TestAddMembers_NotOwner(t *testing.T) {
	circleRepo := new(MockCircleRepository)
	uc := NewCircleUsecase(circleRepo)

	circleRepo.On("GetByID", "c1").Return(&domain.Circle{ID: "c1", OwnerID: "owner-1"}, nil)

	err := uc.AddMembers("c1", "imposter", []string{"user-2"})

	assert.EqualError(t, err, "only the circle owner can perform this action")
	circleRepo.AssertNotCalled(t, "AddMembers")
}

func TestRemoveMember_Success(t *testing.T) {
	circleRepo := new(MockCircleRepository)
	uc := NewCircleUsecase(circleRepo)

	circleRepo.On("GetByID", "c1").Return(&domain.Circle{ID: "c1", OwnerID: "owner-1"}, nil)
	circleRepo.On("RemoveMember", "c1", "target-1").Return(nil)

	// Owner can remove someone else
	err := uc.RemoveMember("c1", "owner-1", "target-1")

	assert.NoError(t, err)
	circleRepo.AssertExpectations(t)
}

func TestRemoveMember_SelfRemove(t *testing.T) {
	circleRepo := new(MockCircleRepository)
	uc := NewCircleUsecase(circleRepo)

	circleRepo.On("GetByID", "c1").Return(&domain.Circle{ID: "c1", OwnerID: "owner-1"}, nil)
	circleRepo.On("RemoveMember", "c1", "target-1").Return(nil)

	// Member can remove themselves
	err := uc.RemoveMember("c1", "target-1", "target-1")

	assert.NoError(t, err)
	circleRepo.AssertExpectations(t)
}

func TestRemoveMember_OwnerRemoveSelf_Fails(t *testing.T) {
	circleRepo := new(MockCircleRepository)
	uc := NewCircleUsecase(circleRepo)

	circleRepo.On("GetByID", "c1").Return(&domain.Circle{ID: "c1", OwnerID: "owner-1"}, nil)

	err := uc.RemoveMember("c1", "owner-1", "owner-1")

	assert.EqualError(t, err, "owner cannot be removed, must delete circle instead")
}
