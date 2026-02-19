package domain

import (
	"errors"

	"github.com/google/uuid"
)

// ErrInvalidOrderID indicates the order ID format is invalid.
var ErrInvalidOrderID = errors.New("invalid order ID format")

// OrderID represents a unique identifier for an order.
type OrderID struct {
	value string
}

func NewOrderID() OrderID {
	return OrderID{value: uuid.New().String()}
}

func ParseOrderID(s string) (OrderID, error) {
	if _, err := uuid.Parse(s); err != nil {
		return OrderID{}, ErrInvalidOrderID
	}
	return OrderID{value: s}, nil
}

func (id OrderID) String() string { return id.value }
func (id OrderID) IsZero() bool   { return id.value == "" }
