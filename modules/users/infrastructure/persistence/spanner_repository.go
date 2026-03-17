// Package persistence implements repository interfaces for users.
package persistence

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"

	platformspanner "github.com/rai/clean-modularmonolith-go/internal/platform/spanner"
	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// SpannerRepository implements UserRepository using Cloud Spanner.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository creates a new Spanner-backed user repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

// Compile-time interface check.
var _ domain.UserRepository = (*SpannerRepository)(nil)

func (r *SpannerRepository) Save(ctx context.Context, user *domain.User) error {
	stmt := spanner.Statement{
		SQL: `INSERT OR UPDATE INTO Users (UserID, Email, FirstName, LastName, Status, CreatedAt, UpdatedAt)
		      VALUES (@userID, @email, @firstName, @lastName, @status, @createdAt, @updatedAt)`,
		Params: map[string]interface{}{
			"userID":    user.ID().String(),
			"email":     user.Email().String(),
			"firstName": user.Name().FirstName(),
			"lastName":  user.Name().LastName(),
			"status":    user.Status().String(),
			"createdAt": user.CreatedAt(),
			"updatedAt": user.UpdatedAt(),
		},
	}

	if err := platformspanner.Write(ctx, r.client, stmt); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}
	return nil
}

func (r *SpannerRepository) FindByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
	return platformspanner.SingleRead(ctx, r.client, func(ctx context.Context, rtx platformspanner.ReadTransaction) (*domain.User, error) {
		row, err := rtx.ReadRow(ctx, "Users",
			spanner.Key{id.String()},
			[]string{"UserID", "Email", "FirstName", "LastName", "Status", "CreatedAt", "UpdatedAt"},
		)
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return nil, domain.ErrUserNotFound
			}
			return nil, fmt.Errorf("failed to read user: %w", err)
		}

		return r.scanUser(row)
	})
}

func (r *SpannerRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	return platformspanner.SingleRead(ctx, r.client, func(ctx context.Context, rtx platformspanner.ReadTransaction) (*domain.User, error) {
		stmt := spanner.Statement{
			SQL: `SELECT UserID, Email, FirstName, LastName, Status, CreatedAt, UpdatedAt
			      FROM Users@{FORCE_INDEX=UsersByEmail}
			      WHERE Email = @email
			      LIMIT 1`,
			Params: map[string]interface{}{"email": email.String()},
		}

		iter := rtx.Query(ctx, stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err == iterator.Done {
			return nil, domain.ErrUserNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query user by email: %w", err)
		}

		return r.scanUser(row)
	})
}

func (r *SpannerRepository) Exists(ctx context.Context, email domain.Email) (bool, error) {
	return platformspanner.SingleRead(ctx, r.client, func(ctx context.Context, rtx platformspanner.ReadTransaction) (bool, error) {
		stmt := spanner.Statement{
			SQL:    `SELECT 1 FROM Users@{FORCE_INDEX=UsersByEmail} WHERE Email = @email LIMIT 1`,
			Params: map[string]interface{}{"email": email.String()},
		}

		iter := rtx.Query(ctx, stmt)
		defer iter.Stop()

		_, err := iter.Next()
		if err == iterator.Done {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("failed to check user existence: %w", err)
		}
		return true, nil
	})
}

func (r *SpannerRepository) FindAll(ctx context.Context, offset, limit int) ([]*domain.User, int, error) {
	var total int
	users, err := platformspanner.ConsistentRead(ctx, r.client, func(ctx context.Context, rtx platformspanner.ReadTransaction) ([]*domain.User, error) {
		// Get total count
		countStmt := spanner.Statement{
			SQL: `SELECT COUNT(*) FROM Users WHERE Status != 'deleted'`,
		}
		countIter := rtx.Query(ctx, countStmt)
		defer countIter.Stop()

		var totalCount int64
		countRow, err := countIter.Next()
		if err != nil && err != iterator.Done {
			return nil, fmt.Errorf("failed to count users: %w", err)
		}
		if countRow != nil {
			if err := countRow.Columns(&totalCount); err != nil {
				return nil, fmt.Errorf("failed to scan count: %w", err)
			}
		}
		total = int(totalCount)

		// Query with pagination
		stmt := spanner.Statement{
			SQL: `SELECT UserID, Email, FirstName, LastName, Status, CreatedAt, UpdatedAt
			      FROM Users
			      WHERE Status != 'deleted'
			      ORDER BY CreatedAt DESC
			      LIMIT @limit OFFSET @offset`,
			Params: map[string]interface{}{
				"limit":  int64(limit),
				"offset": int64(offset),
			},
		}

		iter := rtx.Query(ctx, stmt)
		defer iter.Stop()

		var users []*domain.User
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("failed to query users: %w", err)
			}

			user, err := r.scanUser(row)
			if err != nil {
				return nil, err
			}
			users = append(users, user)
		}

		return users, nil
	})
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *SpannerRepository) scanUser(row *spanner.Row) (*domain.User, error) {
	var userID, emailStr, firstName, lastName, status string
	var createdAt, updatedAt time.Time

	if err := row.Columns(&userID, &emailStr, &firstName, &lastName, &status, &createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	id, err := domain.ParseUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user id: %w", err)
	}

	email, err := domain.NewEmail(emailStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse email: %w", err)
	}

	name, err := domain.NewName(firstName, lastName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse name: %w", err)
	}

	return domain.Reconstitute(id, email, name, domain.Status(status), createdAt, updatedAt), nil
}
