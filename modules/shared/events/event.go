package events

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// EventType represents a domain event type with format "module.AggregateVerb".
// Examples: "users.UserCreated", "orders.OrderSubmitted"
type EventType string

// eventTypePattern validates the format: lowercase_module.PascalCaseEvent
var eventTypePattern = regexp.MustCompile(`^[a-z]+\.[A-Z][a-zA-Z]+$`)

// Validate checks if the event type follows the required format.
func (t EventType) Validate() error {
	if !eventTypePattern.MatchString(string(t)) {
		return fmt.Errorf("invalid event type format %q: must match 'module.AggregateVerb' (e.g., 'orders.OrderCreated')", t)
	}
	return nil
}

func (t EventType) String() string {
	return string(t)
}

// Event represents a domain event.
type Event interface {
	// EventID returns the unique identifier for this event instance.
	EventID() string
	// EventType returns the type name of the event (e.g., "UserCreated").
	EventType() EventType
	// OccurredAt returns when the event occurred.
	OccurredAt() time.Time
	// AggregateID returns the ID of the aggregate that produced this event.
	AggregateID() string
}

// BaseEvent provides common event fields that implements Event interface. Embed this in concrete event types.
type BaseEvent struct {
	id          string
	eventType   EventType
	timestamp   time.Time
	aggregateId string
}

// NewBaseEvent creates a new BaseEvent. Panics if eventType format is invalid.
func NewBaseEvent(eventType EventType, aggregateID string) BaseEvent {
	if err := eventType.Validate(); err != nil {
		panic(err)
	}
	return BaseEvent{
		id:          uuid.New().String(), // TODO
		eventType:   eventType,
		timestamp:   time.Now().UTC(), // TODO
		aggregateId: aggregateID,
	}
}

func (e BaseEvent) EventID() string       { return e.id }
func (e BaseEvent) EventType() EventType  { return e.eventType }
func (e BaseEvent) OccurredAt() time.Time { return e.timestamp }
func (e BaseEvent) AggregateID() string   { return e.aggregateId }

// Publisher publishes domain events for other modules to consume.
type Publisher interface {
	Publish(ctx context.Context, events ...Event) error
}

// Handler handles a specific type of domain event.
type Handler interface {
	Handle(ctx context.Context, event Event) error
}

// Subscriber subscribes to domain events.
type Subscriber interface {
	Subscribe(eventType EventType, handler Handler) error
}
