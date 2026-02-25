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

// AddDomainEvent adds an event to the aggregate's internal collection.
// Events are collected during business operations and dispatched later
// within the same transaction.
func (a *AggregateRoot) AddDomainEvent(event events.Event) {
	a.domainEvents = append(a.domainEvents, event)
}

// DomainEvents returns all collected domain events.
// Use this after saving the aggregate to publish events.
func (a *AggregateRoot) DomainEvents() []events.Event {
	return a.domainEvents
}

// ClearDomainEvents removes all collected events.
// Call this after events have been successfully processed.
func (a *AggregateRoot) ClearDomainEvents() {
	a.domainEvents = nil
}
