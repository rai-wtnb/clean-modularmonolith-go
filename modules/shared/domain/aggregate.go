// Package domain provides shared domain primitives.
package domain

import "github.com/rai/clean-modularmonolith-go/modules/shared/events"

// AggregateRoot is a base type for aggregate roots that collect domain events.
// Embed this in aggregate structs to gain event collection capability.
//
// Example:
//
//	type User struct {
//	    domain.AggregateRoot
//	    id    UserID
//	    email Email
//	}
//
//	func (u *User) Delete() error {
//	    u.status = StatusDeleted
//	    u.AddDomainEvent(NewUserDeletedEvent(u.id))
//	    return nil
//	}
type AggregateRoot struct {
	domainEvents []events.Event
}

func NewAggregateRoot() AggregateRoot {
	return AggregateRoot{
		domainEvents: make([]events.Event, 0),
	}
}

// AddDomainEvent adds an event to the aggregate's internal collection.
// Events are collected during business operations and dispatched later
// within the same transaction.
func (a *AggregateRoot) AddDomainEvent(event events.Event) {
	a.domainEvents = append(a.domainEvents, event)
}

// PopDomainEvents returns all collected domain events and clears them.
// This ensures events are only processed once.
func (a *AggregateRoot) PopDomainEvents() []events.Event {
	es := a.domainEvents
	a.domainEvents = []events.Event{}
	return es
}
