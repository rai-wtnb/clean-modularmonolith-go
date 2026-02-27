// Package contracts defines public event contracts for inter-module communication.
// Modules should import event types from here, NOT from other module's domain packages.
package contracts

import "github.com/rai/clean-modularmonolith-go/modules/shared/events"

const (
	UserDeletedEventType events.EventType = "users.UserDeleted"
)

type UserDeletedEvent struct {
	events.BaseEvent
	UserID string
}
