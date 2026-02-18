package domain

import (
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/types"
)

// Domain events for the users bounded context.
// Events represent facts about what happened in the domain.

const (
	UserCreatedEventType = "users.UserCreated"
	UserUpdatedEventType = "users.UserUpdated"
	UserDeletedEventType = "users.UserDeleted"
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

// UserDeletedEvent is published when a user is deleted.
type UserDeletedEvent struct {
	events.BaseEvent
	UserID types.UserID `json:"user_id"`
}

func NewUserDeletedEvent(userID types.UserID) UserDeletedEvent {
	return UserDeletedEvent{
		BaseEvent: events.NewBaseEvent(UserDeletedEventType, userID.String()),
		UserID:    userID,
	}
}
