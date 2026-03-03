package application

import (
	"context"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events/contracts"
	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// EventPublisher translates internal domain events to cross-module contract events
// and delegates to the underlying events.Publisher.
//
// This is the only place in the users module that knows about both domain event
// types and cross-module contract types. Command handlers remain unaware of
// the translation — they publish domain events and this adapter handles the rest.
//
// Internal events (UserCreatedEvent, UserUpdatedEvent) are intentionally not
// forwarded; they are intra-module only and have no cross-module subscribers.
type EventPublisher struct {
	inner events.Publisher
}

var _ events.Publisher = (*EventPublisher)(nil)

func NewUsersPublisher(inner events.Publisher) *EventPublisher {
	return &EventPublisher{inner: inner}
}

func (p *EventPublisher) Publish(ctx context.Context, domainEvents ...events.Event) error {
	contractEvents := make([]events.Event, 0, len(domainEvents))
	for _, evt := range domainEvents {
		switch e := evt.(type) {
		case domain.UserDeletedEvent:
			// Reuse BaseEvent to preserve the original event ID and timestamp.
			contractEvents = append(contractEvents, contracts.UserDeletedEvent{
				BaseEvent: e.BaseEvent,
				UserID:    e.UserID,
			})
		}
	}
	if len(contractEvents) == 0 {
		return nil
	}
	return p.inner.Publish(ctx, contractEvents...)
}
