// Package queries contains read use cases for the users module.
// Queries return data and don't change state (CQRS pattern).
package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// UserDTO is a read model for user data.
// DTOs are optimized for reading and decoupled from domain entities.
type UserDTO struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	FullName  string    `json:"full_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetUserQuery represents a request to get a user by ID.
type GetUserQuery struct {
	UserID string
}

// GetUserHandler handles GetUserQuery.
type GetUserHandler struct {
	repo domain.UserRepository
}

func NewGetUserHandler(repo domain.UserRepository) *GetUserHandler {
	return &GetUserHandler{repo: repo}
}

// Handle executes the get user query.
func (h *GetUserHandler) Handle(ctx context.Context, query GetUserQuery) (*UserDTO, error) {
	userID, err := domain.ParseUserID(query.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := h.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return toUserDTO(user), nil
}

func toUserDTO(user *domain.User) *UserDTO {
	return &UserDTO{
		ID:        user.ID().String(),
		Email:     user.Email().String(),
		FirstName: user.Name().FirstName(),
		LastName:  user.Name().LastName(),
		FullName:  user.Name().FullName(),
		Status:    user.Status().String(),
		CreatedAt: user.CreatedAt(),
		UpdatedAt: user.UpdatedAt(),
	}
}
