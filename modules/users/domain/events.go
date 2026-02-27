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

func newUserDeletedEvent(userID UserID) contracts.UserDeletedEvent {
	return contracts.UserDeletedEvent{
		BaseEvent: events.NewBaseEvent(contracts.UserDeletedEventType, userID.String()),
		UserID:    userID.String(),
	}
}
