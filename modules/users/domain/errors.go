package domain

import "errors"

// Domain errors - business rule violations.
// These errors are part of the domain language.
var (
	// User errors
	ErrUserNotFound = errors.New("user not found")
	ErrUserDeleted  = errors.New("user has been deleted")

	// Email errors
	ErrEmailRequired = errors.New("email is required")
	ErrEmailInvalid  = errors.New("email format is invalid")
	ErrEmailExists   = errors.New("email already exists")

	// Name errors
	ErrFirstNameRequired = errors.New("first name is required")
	ErrFirstNameLength   = errors.New("first name must be 2-50 characters")
	ErrLastNameRequired  = errors.New("last name is required")
	ErrLastNameLength    = errors.New("last name must be 2-50 characters")
)
