package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
)

// Config holds the Elasticsearch client configuration.
type Config struct {
	Addresses []string
	Username  string
	Password  string
	APIKey    string
}

// ElasticsearchClient wraps the official go-elasticsearch typed client.
type ElasticsearchClient struct {
	client *elasticsearch.TypedClient
}

// NewElasticsearchClient creates a new Elasticsearch client.
func NewElasticsearchClient(cfg Config) (*ElasticsearchClient, error) {
	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
	}
	if cfg.Username != "" {
		esCfg.Username = cfg.Username
		esCfg.Password = cfg.Password
	}
	if cfg.APIKey != "" {
		esCfg.APIKey = cfg.APIKey
	}

	client, err := elasticsearch.NewTypedClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("creating elasticsearch client: %w", err)
	}

	return &ElasticsearchClient{client: client}, nil
}

var _ Client = (*ElasticsearchClient)(nil)

func (c *ElasticsearchClient) Index(ctx context.Context, index string, doc Document) error {
	_, err := c.client.Index(index).
		Id(doc.ID).
		Request(doc.Body).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("indexing document %s: %w", doc.ID, err)
	}
	return nil
}

func (c *ElasticsearchClient) Delete(ctx context.Context, index string, id string) error {
	_, err := c.client.Delete(index, id).Do(ctx)
	if err != nil {
		return fmt.Errorf("deleting document %s: %w", id, err)
	}
	return nil
}

func (c *ElasticsearchClient) Search(ctx context.Context, index string, query string, offset, limit int) (*SearchResponse, error) {
	res, err := c.client.Search().
		Index(index).
		Query(&types.Query{
			MultiMatch: &types.MultiMatchQuery{
				Query:  query,
				Fields: []string{"email", "first_name", "last_name"},
			},
		}).
		From(offset).
		Size(limit).
		Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("searching index %s: %w", index, err)
	}

	hits := make([]SearchResult, len(res.Hits.Hits))
	for i, hit := range res.Hits.Hits {
		var source map[string]any
		if err := json.Unmarshal(hit.Source_, &source); err != nil {
			return nil, fmt.Errorf("unmarshaling hit: %w", err)
		}

		var score float64
		if hit.Score_ != nil {
			score = float64(*hit.Score_)
		}

		var id string
		if hit.Id_ != nil {
			id = *hit.Id_
		}

		hits[i] = SearchResult{
			ID:     id,
			Score:  score,
			Source: source,
		}
	}

	total := len(hits)
	if res.Hits.Total != nil {
		total = int(res.Hits.Total.Value)
	}

	return &SearchResponse{
		Hits:  hits,
		Total: total,
	}, nil
}
