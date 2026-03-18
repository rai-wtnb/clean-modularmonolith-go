package queries

import (
	"context"

	"github.com/rai/clean-modularmonolith-go/internal/platform/elasticsearch"
)

const usersIndex = "users"

// UserSearchResultDTO represents a single user in search results.
type UserSearchResultDTO struct {
	UserID    string  `json:"user_id"`
	Email     string  `json:"email"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Score     float64 `json:"score"`
}

// UserSearchResponseDTO contains the search response.
type UserSearchResponseDTO struct {
	Users  []UserSearchResultDTO `json:"users"`
	Total  int                   `json:"total"`
	Offset int                   `json:"offset"`
	Limit  int                   `json:"limit"`
}

// SearchUsersQuery represents a request to search users.
type SearchUsersQuery struct {
	Query  string
	Offset int
	Limit  int
}

// SearchUsersHandler handles SearchUsersQuery.
type SearchUsersHandler struct {
	esClient elasticsearch.Client
}

func NewSearchUsersHandler(esClient elasticsearch.Client) *SearchUsersHandler {
	return &SearchUsersHandler{esClient: esClient}
}

// Handle executes the search users query against Elasticsearch.
func (h *SearchUsersHandler) Handle(ctx context.Context, query SearchUsersQuery) (*UserSearchResponseDTO, error) {
	offset := query.Offset
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	resp, err := h.esClient.Search(ctx, usersIndex, query.Query, offset, limit)
	if err != nil {
		return nil, err
	}

	users := make([]UserSearchResultDTO, len(resp.Hits))
	for i, hit := range resp.Hits {
		users[i] = UserSearchResultDTO{
			UserID:    getStringField(hit.Source, "user_id"),
			Email:     getStringField(hit.Source, "email"),
			FirstName: getStringField(hit.Source, "first_name"),
			LastName:  getStringField(hit.Source, "last_name"),
			Score:     hit.Score,
		}
	}

	return &UserSearchResponseDTO{
		Users:  users,
		Total:  resp.Total,
		Offset: offset,
		Limit:  limit,
	}, nil
}

func getStringField(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
