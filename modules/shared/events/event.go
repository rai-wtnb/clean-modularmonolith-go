package events

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// Event represents a domain event.
type Event interface {
	// EventID returns the unique identifier for this event instance.
	EventID() string
	// EventType returns the type name of the event (e.g., "UserCreated").
	EventType() EventType
	// OccurredAt returns when the event occurred.
	OccurredAt() time.Time
}

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

// BaseEvent provides common event fields that implements Event interface. Embed this in concrete event types.
type BaseEvent struct {
	id        string
	eventType EventType
	timestamp time.Time
}

// NewBaseEvent creates a new BaseEvent. Panics if eventType format is invalid.
func NewBaseEvent(eventType EventType) BaseEvent {
	if err := eventType.Validate(); err != nil {
		panic(err)
	}
	return BaseEvent{
		id:        uuid.New().String(), // TODO
		eventType: eventType,
		timestamp: time.Now().UTC(), // TODO
	}
}

func (e BaseEvent) EventID() string       { return e.id }
func (e BaseEvent) EventType() EventType  { return e.eventType }
func (e BaseEvent) OccurredAt() time.Time { return e.timestamp }

// Publisher dispatches domain events to registered handlers.
type Publisher interface {
	Publish(ctx context.Context, events []Event) error
}

// Handler handles a specific type of domain event.
type Handler interface {
	Handle(ctx context.Context, event Event) error
	// HandlerName returns the name of this handler (e.g., "UserDeletedHandler").
	HandlerName() string
	// Subdomain returns the subdomain this handler belongs to (e.g., "orders").
	Subdomain() string
	// EventType returns the domain event type this handler subscribes to.
	EventType() EventType
}

// Subscriber subscribes handlers that run INSIDE the transaction boundary (pre-commit).
type Subscriber interface {
	Subscribe(eventType EventType, handler Handler) error
}

// PostCommitSubscriber subscribes handlers that run AFTER the transaction commits successfully.
// Post-commit handler failures are logged but do not affect the caller.
type PostCommitSubscriber interface {
	SubscribePostCommit(eventType EventType, handler Handler) error
}

// PostCommitPublisher dispatches events to post-commit handlers after the transaction commits.
// Errors from handlers are logged but not returned (best-effort delivery).
type PostCommitPublisher interface {
	PublishPostCommit(ctx context.Context, events []Event)
}
