package domain

import (
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events/contracts"
)

// Domain events for the users bounded context.
// Events represent facts about what happened in the domain.
//
// For cross-module events, we use types from contracts package
// to ensure type compatibility with subscribers in other modules.

const (
	UserCreatedEventType events.EventType = "users.UserCreated"
	UserUpdatedEventType events.EventType = "users.UserUpdated"
)

// Re-export from contracts for convenience within this module
const UserDeletedEventType = contracts.UserDeletedEventType

// UserCreatedEvent is published when a new user is created.
type UserCreatedEvent struct {
	events.BaseEvent
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func NewUserCreatedEvent(user *User) UserCreatedEvent {
	return UserCreatedEvent{
		BaseEvent: events.NewBaseEvent(UserCreatedEventType, user.ID().String()),
		UserID:    user.ID().String(),
		Email:     user.Email().String(),
		FirstName: user.Name().FirstName(),
		LastName:  user.Name().LastName(),
	}
}

// UserUpdatedEvent is published when a user is updated.
type UserUpdatedEvent struct {
	events.BaseEvent
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func NewUserUpdatedEvent(user *User) UserUpdatedEvent {
	return UserUpdatedEvent{
		BaseEvent: events.NewBaseEvent(UserUpdatedEventType, user.ID().String()),
		UserID:    user.ID().String(),
		Email:     user.Email().String(),
		FirstName: user.Name().FirstName(),
		LastName:  user.Name().LastName(),
	}
}

// UserDeletedEvent is a type alias for the cross-module contract.
// This ensures subscribers in other modules can type-assert correctly.
type UserDeletedEvent = contracts.UserDeletedEvent

func NewUserDeletedEvent(userID UserID) UserDeletedEvent {
	return UserDeletedEvent{
		BaseEvent: events.NewBaseEvent(UserDeletedEventType, userID.String()),
		UserID:    userID.String(),
	}
}
