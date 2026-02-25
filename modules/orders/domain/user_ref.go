package domain

import (
	"errors"

	"github.com/google/uuid"
)

// ErrInvalidUserRef indicates the user reference format is invalid.
var ErrInvalidUserRef = errors.New("invalid user reference format")

// UserRef represents a reference to a user from another module.
// This is the orders module's own type for referencing users,
// following the Anti-Corruption Layer pattern to prevent coupling
// to the users module's internal domain types.
type UserRef struct {
	value string
}

// NewUserRef creates a UserRef from a validated string.
func NewUserRef(s string) (UserRef, error) {
	if _, err := uuid.Parse(s); err != nil {
		return UserRef{}, ErrInvalidUserRef
	}
	return UserRef{value: s}, nil
}

// MustNewUserRef creates a UserRef, panicking if invalid.
// Use only for trusted input (e.g., from database).
func MustNewUserRef(s string) UserRef {
	ref, err := NewUserRef(s)
	if err != nil {
		panic(err)
	}
	return ref
}

func (r UserRef) String() string { return r.value }
func (r UserRef) IsZero() bool   { return r.value == "" }
