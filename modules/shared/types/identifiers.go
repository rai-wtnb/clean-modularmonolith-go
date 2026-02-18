// Package types provides shared value objects and type definitions
// used across multiple modules (Shared Kernel pattern).
package types

import (
	"github.com/google/uuid"
)

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
		return UserID{}, ErrInvalidID
	}
	return UserID{value: s}, nil
}

func (id UserID) String() string { return id.value }
func (id UserID) IsZero() bool   { return id.value == "" }

// OrderID represents a unique identifier for an order.
type OrderID struct {
	value string
}

func NewOrderID() OrderID {
	return OrderID{value: uuid.New().String()}
}

func ParseOrderID(s string) (OrderID, error) {
	if _, err := uuid.Parse(s); err != nil {
		return OrderID{}, ErrInvalidID
	}
	return OrderID{value: s}, nil
}

func (id OrderID) String() string { return id.value }
func (id OrderID) IsZero() bool   { return id.value == "" }
