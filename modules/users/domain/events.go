package domain

import (
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// Domain events for the users bounded context.
// Events represent facts about what happened in the domain.
//
// Per ADR (domain-event placement), every DomainEvent type — including
// cross-module ones — is defined in the publishing subdomain's domain package.
// Handlers in other subdomains may import the event TYPE only; the event's
// constructor stays unexported so creation is confined to this subdomain.

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
		BaseEvent: events.NewBaseEvent(UserCreatedEventType),
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
		BaseEvent: events.NewBaseEvent(UserUpdatedEventType),
		UserID:    user.ID().String(),
		Email:     user.Email().String(),
		FirstName: user.Name().FirstName(),
		LastName:  user.Name().LastName(),
	}
}

// UserDeletedEvent is published when a user is deleted.
// This is a public domain event — event handlers in other subdomains may
// import this type. Only the type may be referenced cross-module; the
// constructor is unexported so events are created only within this subdomain.
type UserDeletedEvent struct {
	events.BaseEvent
	UserID string
}

func newUserDeletedEvent(userID UserID) UserDeletedEvent {
	return UserDeletedEvent{
		BaseEvent: events.NewBaseEvent(UserDeletedEventType),
		UserID:    userID.String(),
	}
}
