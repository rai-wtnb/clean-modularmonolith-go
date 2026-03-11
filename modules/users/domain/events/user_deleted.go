package events

import "github.com/rai/clean-modularmonolith-go/modules/shared/events"

const UserDeletedEventType events.EventType = "users.UserDeleted"

// UserDeletedEvent is published when a user is deleted.
// This is a public domain event — it may be imported by event handlers in other modules.
type UserDeletedEvent struct {
	events.BaseEvent
	UserID string
}
