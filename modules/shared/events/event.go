// Package events provides domain event infrastructure for inter-module communication.
// Events enable loose coupling between modules - modules publish events without
// knowing who will handle them.
package events

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Event represents a domain event that occurred in the system.
// Events are immutable facts about something that happened.
type Event interface {
	// EventID returns the unique identifier for this event instance.
	EventID() string
	// EventType returns the type name of the event (e.g., "UserCreated").
	EventType() string
	// OccurredAt returns when the event occurred.
	OccurredAt() time.Time
	// AggregateID returns the ID of the aggregate that produced this event.
	AggregateID() string
}

// BaseEvent provides common event fields. Embed this in concrete event types.
type BaseEvent struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Timestamp   time.Time `json:"timestamp"`
	AggregateId string    `json:"aggregate_id"`
}

func NewBaseEvent(eventType, aggregateID string) BaseEvent {
	return BaseEvent{
		ID:          uuid.New().String(),
		Type:        eventType,
		Timestamp:   time.Now().UTC(),
		AggregateId: aggregateID,
	}
}

func (e BaseEvent) EventID() string       { return e.ID }
func (e BaseEvent) EventType() string     { return e.Type }
func (e BaseEvent) OccurredAt() time.Time { return e.Timestamp }
func (e BaseEvent) AggregateID() string   { return e.AggregateId }

// Publisher publishes domain events for other modules to consume.
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

// Handler handles a specific type of domain event.
type Handler interface {
	Handle(ctx context.Context, event Event) error
}

// Subscriber subscribes to domain events.
type Subscriber interface {
	Subscribe(eventType string, handler Handler) error
}
