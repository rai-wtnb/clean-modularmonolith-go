package eventhandlers

import (
	"context"
	"log/slog"

	"github.com/rai/clean-modularmonolith-go/internal/platform/elasticsearch"
	"github.com/rai/clean-modularmonolith-go/modules/shared/idempotent"
)

const usersIndex = "users"

// UserIndexer indexes/deletes user documents in Elasticsearch.
// It embeds idempotent.OutboundCache so each outbound call is deduplicated on Spanner retry.
type UserIndexer struct {
	*idempotent.OutboundCache
	esClient elasticsearch.Client
	logger   *slog.Logger
}

func NewUserIndexer(esClient elasticsearch.Client, logger *slog.Logger) (_ *UserIndexer, cleanup func()) {
	cache, cleanup := idempotent.NewOutboundCache()
	return &UserIndexer{
		OutboundCache: cache,
		esClient:      esClient,
		logger:        logger,
	}, cleanup
}

func (i *UserIndexer) IndexUser(ctx context.Context, userID, email, firstName, lastName string) error {
	return i.Once("index-user", userID, func() error {
		doc := elasticsearch.Document{
			ID: userID,
			Body: map[string]any{
				"user_id":    userID,
				"email":      email,
				"first_name": firstName,
				"last_name":  lastName,
			},
		}
		i.logger.Info("indexing user in elasticsearch", slog.String("user_id", userID))
		return i.esClient.Index(ctx, usersIndex, doc)
	}, i.HashInput(userID, email, firstName, lastName))
}

func (i *UserIndexer) DeleteUser(ctx context.Context, userID string) error {
	return i.Once("delete-user", userID, func() error {
		i.logger.Info("deleting user from elasticsearch",
			slog.String("user_id", userID),
		)
		return i.esClient.Delete(ctx, usersIndex, userID)
	})
}
