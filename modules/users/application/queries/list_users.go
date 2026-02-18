package queries

import (
	"context"

	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// UserListDTO contains a paginated list of users.
type UserListDTO struct {
	Users      []*UserDTO `json:"users"`
	TotalCount int        `json:"total_count"`
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
}

// ListUsersQuery represents a request to list users with pagination.
type ListUsersQuery struct {
	Offset int
	Limit  int
}

// ListUsersHandler handles ListUsersQuery.
type ListUsersHandler struct {
	repo domain.UserRepository
}

func NewListUsersHandler(repo domain.UserRepository) *ListUsersHandler {
	return &ListUsersHandler{repo: repo}
}

// Handle executes the list users query.
func (h *ListUsersHandler) Handle(ctx context.Context, query ListUsersQuery) (*UserListDTO, error) {
	// Apply defaults
	offset := query.Offset
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	users, total, err := h.repo.FindAll(ctx, offset, limit)
	if err != nil {
		return nil, err
	}

	dtos := make([]*UserDTO, len(users))
	for i, user := range users {
		dtos[i] = toUserDTO(user)
	}

	return &UserListDTO{
		Users:      dtos,
		TotalCount: total,
		Offset:     offset,
		Limit:      limit,
	}, nil
}
