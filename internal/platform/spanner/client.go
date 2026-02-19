// Package spanner provides Cloud Spanner client initialization.
package spanner

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
)

// Config holds Spanner connection configuration.
type Config struct {
	ProjectID  string
	InstanceID string
	DatabaseID string
}

// DSN returns the Spanner database connection string.
func (c Config) DSN() string {
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		c.ProjectID, c.InstanceID, c.DatabaseID)
}

// NewClient creates a new Spanner client from config.
// The caller is responsible for closing the client when done.
func NewClient(ctx context.Context, cfg Config) (*spanner.Client, error) {
	client, err := spanner.NewClient(ctx, cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to create spanner client: %w", err)
	}
	return client, nil
}
