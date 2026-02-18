package domain

import "errors"

var (
	ErrOrderNotFound        = errors.New("order not found")
	ErrOrderNotDraft        = errors.New("order is not in draft status")
	ErrOrderNotPending      = errors.New("order is not pending")
	ErrOrderNotConfirmed    = errors.New("order is not confirmed")
	ErrOrderEmpty           = errors.New("order has no items")
	ErrOrderAlreadyCancelled = errors.New("order is already cancelled")
	ErrOrderCompleted       = errors.New("order is already completed")
	ErrItemNotFound         = errors.New("item not found in order")
	ErrInvalidQuantity      = errors.New("quantity must be positive")
)
