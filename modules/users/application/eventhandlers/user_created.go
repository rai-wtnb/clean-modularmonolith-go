package eventhandlers

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// UserCreatedHandler handles UserCreated events by indexing the user in Elasticsearch.
// Performs external side effects; must not run within a database transaction.
type UserCreatedHandler struct {
	indexer *UserIndexer
}

func NewUserCreatedHandler(indexer *UserIndexer) *UserCreatedHandler {
	return &UserCreatedHandler{indexer: indexer}
}

func (h *UserCreatedHandler) HandlerName() string         { return "UserCreatedHandler" }
func (h *UserCreatedHandler) Subdomain() string           { return "users" }
func (h *UserCreatedHandler) EventType() events.EventType { return domain.UserCreatedEventType }

func (h *UserCreatedHandler) Handle(ctx context.Context, event events.Event) error {
	e, ok := event.(domain.UserCreatedEvent)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", event)
	}
	return h.indexer.IndexUser(ctx, e.UserID, e.Email, e.FirstName, e.LastName)
}
