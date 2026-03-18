package eventhandlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rai/clean-modularmonolith-go/internal/platform/elasticsearch"
	"github.com/rai/clean-modularmonolith-go/modules/shared/idempotent"
)

const usersIndex = "users"

// UserIndexer indexes/deletes user documents in Elasticsearch.
// It embeds idempotent.Base so each outbound call is deduplicated on Spanner retry.
type UserIndexer struct {
	idempotent.Base
	esClient elasticsearch.Client
	logger   *slog.Logger
}

func NewUserIndexer(esClient elasticsearch.Client, logger *slog.Logger) *UserIndexer {
	return &UserIndexer{esClient: esClient, logger: logger}
}

func (i *UserIndexer) IndexUser(ctx context.Context, userID, email, firstName, lastName string) error {
	return i.Once(fmt.Sprintf("index-user:%s", userID), func() error {
		doc := elasticsearch.Document{
			ID: userID,
			Body: map[string]any{
				"user_id":    userID,
				"email":      email,
				"first_name": firstName,
				"last_name":  lastName,
			},
		}
		i.logger.Info("indexing user in elasticsearch",
			slog.String("user_id", userID),
		)
		return i.esClient.Index(ctx, usersIndex, doc)
	})
}

func (i *UserIndexer) DeleteUser(ctx context.Context, userID string) error {
	return i.Once(fmt.Sprintf("delete-user:%s", userID), func() error {
		i.logger.Info("deleting user from elasticsearch",
			slog.String("user_id", userID),
		)
		return i.esClient.Delete(ctx, usersIndex, userID)
	})
}
