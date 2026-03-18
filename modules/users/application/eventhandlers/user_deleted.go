package eventhandlers

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	userevents "github.com/rai/clean-modularmonolith-go/modules/users/domain/events"
)

// UserDeletedHandler handles UserDeleted events by removing the user from Elasticsearch.
// Performs external side effects; must not run within a database transaction.
type UserDeletedHandler struct {
	indexer *UserIndexer
}

func NewUserDeletedHandler(indexer *UserIndexer) *UserDeletedHandler {
	return &UserDeletedHandler{indexer: indexer}
}

func (h *UserDeletedHandler) HandlerName() string         { return "UserDeletedHandler" }
func (h *UserDeletedHandler) Subdomain() string           { return "users" }
func (h *UserDeletedHandler) EventType() events.EventType { return userevents.UserDeletedEventType }

func (h *UserDeletedHandler) Handle(ctx context.Context, event events.Event) error {
	e, ok := event.(userevents.UserDeletedEvent)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", event)
	}
	return h.indexer.DeleteUser(ctx, e.UserID)
}
