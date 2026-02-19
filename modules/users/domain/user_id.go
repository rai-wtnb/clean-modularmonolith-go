package domain

import (
	"errors"

	"github.com/google/uuid"
)

// ErrInvalidUserID indicates the user ID format is invalid.
var ErrInvalidUserID = errors.New("invalid user ID format")

// UserID represents a unique identifier for a user.
// Using a distinct type prevents mixing up different ID types.
type UserID struct {
	value string
}

func NewUserID() UserID {
	return UserID{value: uuid.New().String()}
}

func ParseUserID(s string) (UserID, error) {
	if _, err := uuid.Parse(s); err != nil {
		return UserID{}, ErrInvalidUserID
	}
	return UserID{value: s}, nil
}

func (id UserID) String() string { return id.value }
func (id UserID) IsZero() bool   { return id.value == "" }
