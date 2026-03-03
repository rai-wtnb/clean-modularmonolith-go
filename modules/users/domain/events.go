package domain

import (
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// Domain events for the users bounded context.
// Events represent facts about what happened in the domain.
//
// These are internal domain events. Cross-module contract events are produced
// by the application layer (UsersPublisher) which translates these types.

const (
	UserCreatedEventType events.EventType = "users.UserCreated"
	UserUpdatedEventType events.EventType = "users.UserUpdated"
	UserDeletedEventType events.EventType = "users.UserDeleted"
)

// UserCreatedEvent is published when a new user is created.
type UserCreatedEvent struct {
	events.BaseEvent
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func newUserCreatedEvent(user *User) UserCreatedEvent {
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

func newUserUpdatedEvent(user *User) UserUpdatedEvent {
	return UserUpdatedEvent{
		BaseEvent: events.NewBaseEvent(UserUpdatedEventType, user.ID().String()),
		UserID:    user.ID().String(),
		Email:     user.Email().String(),
		FirstName: user.Name().FirstName(),
		LastName:  user.Name().LastName(),
	}
}

// UserDeletedEvent is published when a user is deleted.
type UserDeletedEvent struct {
	events.BaseEvent
	UserID string `json:"user_id"`
}

func newUserDeletedEvent(userID UserID) UserDeletedEvent {
	return UserDeletedEvent{
		BaseEvent: events.NewBaseEvent(UserDeletedEventType, userID.String()),
		UserID:    userID.String(),
	}
}
