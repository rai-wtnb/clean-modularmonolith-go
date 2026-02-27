// Package persistence implements repository interfaces for orders.
package persistence

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"

	platformspanner "github.com/rai/clean-modularmonolith-go/internal/platform/spanner"
	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
)

type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

// Save persists an order.
// It uses an existing transaction if available, otherwise creates a new one.
func (r *SpannerRepository) Save(ctx context.Context, order *domain.Order) error {
	if txn, ok := platformspanner.ReadWriteTxFromContext(ctx); ok {
		return r.saveWithTx(txn, order)
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return r.saveWithTx(txn, order)
	})
	if err != nil {
		return fmt.Errorf("failed to save order: %w", err)
	}
	return nil
}

func (r *SpannerRepository) saveWithTx(tx *spanner.ReadWriteTransaction, order *domain.Order) error {
	orderID := order.ID().String()

	// Delete existing items first (handles item removal on update)
	if err := tx.BufferWrite([]*spanner.Mutation{
		spanner.Delete("OrderItems", spanner.KeyRange{
			Start: spanner.Key{orderID},
			End:   spanner.Key{orderID},
			Kind:  spanner.ClosedClosed,
		}),
	}); err != nil {
		return fmt.Errorf("failed to delete existing items: %w", err)
	}

	// Build mutations for order and items
	mutations := []*spanner.Mutation{
		spanner.InsertOrUpdate("Orders",
			[]string{"OrderID", "UserID", "Status", "TotalAmount", "TotalCurrency", "CreatedAt", "UpdatedAt"},
			[]interface{}{
				orderID,
				order.UserRef().String(),
				order.Status().String(),
				order.Total().Amount(),
				order.Total().Currency(),
				order.CreatedAt(),
				order.UpdatedAt(),
			},
		),
	}

	for i, item := range order.Items() {
		mutations = append(mutations, spanner.InsertOrUpdate("OrderItems",
			[]string{"OrderID", "ItemIndex", "ProductID", "ProductName", "Quantity", "UnitAmount", "Currency"},
			[]interface{}{
				orderID,
				int64(i),
				item.ProductID,
				item.ProductName,
				int64(item.Quantity),
				item.UnitPrice.Amount(),
				item.UnitPrice.Currency(),
			},
		))
	}

	return tx.BufferWrite(mutations)
}

func (r *SpannerRepository) FindByID(ctx context.Context, id domain.OrderID) (*domain.Order, error) {
	reader, ok := platformspanner.ReadTransactionFromContext(ctx)
	if !ok {
		// Reads from Orders + OrderItems require ReadOnlyTransaction
		// for point-in-time consistency. Single() is only for one read.
		roTx := r.client.ReadOnlyTransaction()
		defer roTx.Close()
		reader = roTx
	}

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
}

func (r *SpannerRepository) FindByUserRef(ctx context.Context, userRef domain.UserRef, offset, limit int) ([]*domain.Order, int, error) {
	reader, ok := platformspanner.ReadTransactionFromContext(ctx)
	if !ok {
		// Multiple queries (COUNT + SELECT + items) require ReadOnlyTransaction
		// for point-in-time consistency. Single() is only for one read.
		roTx := r.client.ReadOnlyTransaction()
		defer roTx.Close()
		reader = roTx
	}

	// Get total count
	countStmt := spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM Orders WHERE UserID = @userID`,
		Params: map[string]interface{}{"userID": userRef.String()},
	}

	countIter := reader.Query(ctx, countStmt)
	defer countIter.Stop()

	var total int64
	countRow, err := countIter.Next()
	if err != nil && err != iterator.Done {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}
	if countRow != nil {
		if err := countRow.Columns(&total); err != nil {
			return nil, 0, fmt.Errorf("failed to scan count: %w", err)
		}
	}

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
			return nil, 0, fmt.Errorf("failed to query orders: %w", err)
		}

		var orderID, userIDStr, status, totalCurrency string
		var totalAmount int64
		var createdAt, updatedAt time.Time

		if err := row.Columns(&orderID, &userIDStr, &status, &totalAmount, &totalCurrency, &createdAt, &updatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan order: %w", err)
		}

		items, err := r.readOrderItems(ctx, reader, orderID)
		if err != nil {
			return nil, 0, err
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

	return orders, int(total), nil
}

func (r *SpannerRepository) Delete(ctx context.Context, id domain.OrderID) error {
	mutations := []*spanner.Mutation{
		// ON DELETE CASCADE handles OrderItems automatically
		spanner.Delete("Orders", spanner.Key{id.String()}),
	}

	// Use existing transaction if available
	if txn, ok := platformspanner.ReadWriteTxFromContext(ctx); ok {
		return txn.BufferWrite(mutations)
	}

	// Fallback: standalone mutation (backward compatible)
	_, err := r.client.Apply(ctx, mutations)
	if err != nil {
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
