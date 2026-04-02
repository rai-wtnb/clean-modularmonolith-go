// Package persistence implements repository interfaces for orders.
package persistence

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"

	platformspanner "github.com/rai/clean-modularmonolith-go/internal/platform/spanner"
	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
)

type SpannerRepository struct {
	client *spanner.Client
	logger *slog.Logger
}

func NewSpannerRepository(client *spanner.Client, logger *slog.Logger) *SpannerRepository {
	return &SpannerRepository{client: client, logger: logger}
}

// Save persists an order using DML for read-your-writes consistency.
// It uses an existing transaction if available, otherwise creates a new one.
// All statements are executed in a single BatchUpdate RPC.
func (r *SpannerRepository) Save(ctx context.Context, order *domain.Order) error {
	orderID := order.ID().String()

	stmts := make([]spanner.Statement, 0, 2+len(order.Items()))

	// Delete existing items first
	stmts = append(stmts, spanner.Statement{
		SQL:    `DELETE FROM OrderItems WHERE OrderID = @orderID`,
		Params: map[string]interface{}{"orderID": orderID},
	})

	// Upsert order
	stmts = append(stmts, spanner.Statement{
		SQL: `INSERT OR UPDATE INTO Orders (OrderID, UserID, Status, TotalAmount, TotalCurrency, CreatedAt, UpdatedAt)
		      VALUES (@orderID, @userID, @status, @totalAmount, @totalCurrency, @createdAt, @updatedAt)`,
		Params: map[string]interface{}{
			"orderID":       orderID,
			"userID":        order.UserRef().String(),
			"status":        order.Status().String(),
			"totalAmount":   order.Total().Amount(),
			"totalCurrency": order.Total().Currency(),
			"createdAt":     order.CreatedAt(),
			"updatedAt":     order.UpdatedAt(),
		},
	})

	// Insert items
	for i, item := range order.Items() {
		stmts = append(stmts, spanner.Statement{
			SQL: `INSERT OR UPDATE INTO OrderItems (OrderID, ItemIndex, ProductID, ProductName, Quantity, UnitAmount, Currency)
			      VALUES (@orderID, @itemIndex, @productID, @productName, @quantity, @unitAmount, @currency)`,
			Params: map[string]interface{}{
				"orderID":     orderID,
				"itemIndex":   int64(i),
				"productID":   item.ProductID,
				"productName": item.ProductName,
				"quantity":    int64(item.Quantity),
				"unitAmount":  item.UnitPrice.Amount(),
				"currency":    item.UnitPrice.Currency(),
			},
		})
	}

	if err := platformspanner.Write(ctx, stmts...); err != nil {
		return fmt.Errorf("failed to save order: %w", err)
	}
	return nil
}

func (r *SpannerRepository) FindByID(ctx context.Context, id domain.OrderID) (*domain.Order, error) {
	return platformspanner.ConsistentRead(ctx, r.client, r.logger, func(ctx context.Context, reader platformspanner.ReadTransaction) (*domain.Order, error) {
		row, err := reader.ReadRow(ctx, "Orders",
			spanner.Key{id.String()},
			[]string{"OrderID", "UserID", "Status", "TotalAmount", "TotalCurrency", "CreatedAt", "UpdatedAt"},
		)
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return nil, domain.ErrOrderNotFound
			}
			return nil, fmt.Errorf("failed to read order: %w", err)
		}

		var orderID, userID, status, totalCurrency string
		var totalAmount int64
		var createdAt, updatedAt time.Time

		if err := row.Columns(&orderID, &userID, &status, &totalAmount, &totalCurrency, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		items, err := r.readOrderItems(ctx, reader, orderID)
		if err != nil {
			return nil, err
		}

		parsedOrderID, err := domain.ParseOrderID(orderID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse order id: %w", err)
		}

		userRef := domain.MustNewUserRef(userID)
		total := domain.MustNewMoney(totalAmount, totalCurrency)

		return domain.Reconstitute(
			parsedOrderID,
			userRef,
			items,
			domain.Status(status),
			total,
			createdAt,
			updatedAt,
		), nil
	})
}

func (r *SpannerRepository) FindByUserRef(ctx context.Context, userRef domain.UserRef, offset, limit int) ([]*domain.Order, int, error) {
	var total int
	orders, err := platformspanner.ConsistentRead(ctx, r.client, r.logger, func(ctx context.Context, reader platformspanner.ReadTransaction) ([]*domain.Order, error) {
		// Get total count
		countStmt := spanner.Statement{
			SQL:    `SELECT COUNT(*) FROM Orders WHERE UserID = @userID`,
			Params: map[string]interface{}{"userID": userRef.String()},
		}

		countIter := reader.Query(ctx, countStmt)
		defer countIter.Stop()

		var totalCount int64
		countRow, err := countIter.Next()
		if err != nil && err != iterator.Done {
			return nil, fmt.Errorf("failed to count orders: %w", err)
		}
		if countRow != nil {
			if err := countRow.Columns(&totalCount); err != nil {
				return nil, fmt.Errorf("failed to scan count: %w", err)
			}
		}
		total = int(totalCount)

		// Query orders with pagination
		stmt := spanner.Statement{
			SQL: `SELECT OrderID, UserID, Status, TotalAmount, TotalCurrency, CreatedAt, UpdatedAt
			      FROM Orders@{FORCE_INDEX=OrdersByUserID}
			      WHERE UserID = @userID
			      ORDER BY CreatedAt DESC
			      LIMIT @limit OFFSET @offset`,
			Params: map[string]interface{}{
				"userID": userRef.String(),
				"limit":  int64(limit),
				"offset": int64(offset),
			},
		}

		iter := reader.Query(ctx, stmt)
		defer iter.Stop()

		var orders []*domain.Order
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("failed to query orders: %w", err)
			}

			var orderID, userIDStr, status, totalCurrency string
			var totalAmount int64
			var createdAt, updatedAt time.Time

			if err := row.Columns(&orderID, &userIDStr, &status, &totalAmount, &totalCurrency, &createdAt, &updatedAt); err != nil {
				return nil, fmt.Errorf("failed to scan order: %w", err)
			}

			items, err := r.readOrderItems(ctx, reader, orderID)
			if err != nil {
				return nil, err
			}

			parsedOrderID, _ := domain.ParseOrderID(orderID)
			userRefFromDB := domain.MustNewUserRef(userIDStr)
			orderTotal := domain.MustNewMoney(totalAmount, totalCurrency)

			orders = append(orders, domain.Reconstitute(
				parsedOrderID,
				userRefFromDB,
				items,
				domain.Status(status),
				orderTotal,
				createdAt,
				updatedAt,
			))
		}

		return orders, nil
	})
	if err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

func (r *SpannerRepository) Delete(ctx context.Context, id domain.OrderID) error {
	if err := platformspanner.Write(ctx, spanner.Statement{
		SQL:    `DELETE FROM Orders WHERE OrderID = @orderID`,
		Params: map[string]interface{}{"orderID": id.String()},
	}); err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}
	return nil
}

func (r *SpannerRepository) readOrderItems(ctx context.Context, reader platformspanner.ReadTransaction, orderID string) ([]domain.OrderItem, error) {
	iter := reader.Read(ctx, "OrderItems",
		spanner.KeyRange{
			Start: spanner.Key{orderID},
			End:   spanner.Key{orderID},
			Kind:  spanner.ClosedClosed,
		},
		[]string{"ProductID", "ProductName", "Quantity", "UnitAmount", "Currency"},
	)
	defer iter.Stop()

	var items []domain.OrderItem
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read order items: %w", err)
		}

		var productID, productName, currency string
		var quantity, unitAmount int64

		if err := row.Columns(&productID, &productName, &quantity, &unitAmount, &currency); err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}

		items = append(items, domain.OrderItem{
			ProductID:   productID,
			ProductName: productName,
			Quantity:    int(quantity),
			UnitPrice:   domain.MustNewMoney(unitAmount, currency),
		})
	}

	return items, nil
}
