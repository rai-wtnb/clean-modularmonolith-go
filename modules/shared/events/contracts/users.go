// Package contracts defines public event contracts for inter-module communication.
// Modules should import event types from here, NOT from other module's domain packages.
package contracts

import "github.com/rai/clean-modularmonolith-go/modules/shared/events"

// User module event types.
// These are the "public API" of the users module for event-driven communication.
const (
	UserCreatedEventType events.EventType = "users.UserCreated"
	UserUpdatedEventType events.EventType = "users.UserUpdated"
	UserDeletedEventType events.EventType = "users.UserDeleted"
)

// UserDeletedEvent is the public contract for user deletion events.
// Other modules should use this type to handle user deletions.
type UserDeletedEvent struct {
	events.BaseEvent
	UserID string `json:"user_id"`
}
