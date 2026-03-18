package elasticsearch

import "context"

// Document represents a document to be indexed.
type Document struct {
	ID   string
	Body map[string]any
}

// SearchResult represents a single search hit.
type SearchResult struct {
	ID     string
	Score  float64
	Source map[string]any
}

// SearchResponse contains the search results.
type SearchResponse struct {
	Hits  []SearchResult
	Total int
}

// Client is the interface for Elasticsearch operations.
type Client interface {
	// Index indexes or updates a document in the given index.
	Index(ctx context.Context, index string, doc Document) error
	// Delete removes a document from the given index.
	Delete(ctx context.Context, index string, id string) error
	// Search searches for documents in the given index matching the query string.
	Search(ctx context.Context, index string, query string, offset, limit int) (*SearchResponse, error)
}
