package eventhandlers

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// UserUpdatedHandler handles UserUpdated events by re-indexing the user in Elasticsearch.
// Performs external side effects; must not run within a database transaction.
type UserUpdatedHandler struct {
	indexer *UserIndexer
}

func NewUserUpdatedHandler(indexer *UserIndexer) *UserUpdatedHandler {
	return &UserUpdatedHandler{indexer: indexer}
}

func (h *UserUpdatedHandler) HandlerName() string         { return "UserUpdatedHandler" }
func (h *UserUpdatedHandler) Subdomain() string           { return "users" }
func (h *UserUpdatedHandler) EventType() events.EventType { return domain.UserUpdatedEventType }

func (h *UserUpdatedHandler) Handle(ctx context.Context, event events.Event) error {
	e, ok := event.(domain.UserUpdatedEvent)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", event)
	}
	return h.indexer.IndexUser(ctx, e.UserID, e.Email, e.FirstName, e.LastName)
}
