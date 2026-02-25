// Package domain contains the business entities and rules for users.
// This is the innermost layer - it has no dependencies on outer layers.
package domain

import (
	"time"

	shareddomain "github.com/rai/clean-modularmonolith-go/modules/shared/domain"
)

// User is the aggregate root for the user bounded context.
// It encapsulates all user-related business rules.
type User struct {
	shareddomain.AggregateRoot

	id        UserID
	email     Email
	name      Name
	status    Status
	createdAt time.Time
	updatedAt time.Time
}

// NewUser creates a new User with validated inputs.
// Factory function enforces all invariants at creation time.
// Adds UserCreatedEvent to be dispatched after persistence.
func NewUser(email Email, name Name) *User {
	u := &User{
		id:        NewUserID(),
		email:     email,
		name:      name,
		status:    StatusActive,
		createdAt: time.Now().UTC(),
		updatedAt: time.Now().UTC(),
	}
	u.AddDomainEvent(NewUserCreatedEvent(u))
	return u
}

// Reconstitute recreates a User from persistence.
// Used by repositories to rebuild aggregates from stored data.
func Reconstitute(
	id UserID,
	email Email,
	name Name,
	status Status,
	createdAt, updatedAt time.Time,
) *User {
	return &User{
		id:        id,
		email:     email,
		name:      name,
		status:    status,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

// Getters - expose state without allowing direct mutation

func (u *User) ID() UserID           { return u.id }
func (u *User) Email() Email         { return u.email }
func (u *User) Name() Name           { return u.name }
func (u *User) Status() Status       { return u.status }
func (u *User) CreatedAt() time.Time { return u.createdAt }
func (u *User) UpdatedAt() time.Time { return u.updatedAt }

// Business methods - encapsulate business rules

// UpdateProfile updates the user's profile information.
// Adds UserUpdatedEvent to be dispatched after persistence.
func (u *User) UpdateProfile(name Name) error {
	if u.status == StatusDeleted {
		return ErrUserDeleted
	}
	u.name = name
	u.updatedAt = time.Now().UTC()
	u.AddDomainEvent(NewUserUpdatedEvent(u))
	return nil
}

// ChangeEmail changes the user's email address.
// Adds UserUpdatedEvent to be dispatched after persistence.
func (u *User) ChangeEmail(email Email) error {
	if u.status == StatusDeleted {
		return ErrUserDeleted
	}
	u.email = email
	u.updatedAt = time.Now().UTC()
	u.AddDomainEvent(NewUserUpdatedEvent(u))
	return nil
}

// Deactivate deactivates the user account.
func (u *User) Deactivate() error {
	if u.status == StatusDeleted {
		return ErrUserDeleted
	}
	u.status = StatusInactive
	u.updatedAt = time.Now().UTC()
	return nil
}

// Activate activates the user account.
func (u *User) Activate() error {
	if u.status == StatusDeleted {
		return ErrUserDeleted
	}
	u.status = StatusActive
	u.updatedAt = time.Now().UTC()
	return nil
}

// Delete marks the user as deleted (soft delete).
// Adds UserDeletedEvent to be dispatched after persistence.
func (u *User) Delete() error {
	u.status = StatusDeleted
	u.updatedAt = time.Now().UTC()
	u.AddDomainEvent(NewUserDeletedEvent(u.id))
	return nil
}

// IsActive returns true if the user account is active.
func (u *User) IsActive() bool {
	return u.status == StatusActive
}
