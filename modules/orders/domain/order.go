// Package domain contains business entities and rules for orders.
package domain

import (
	"time"

	userdomain "github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// Order is the aggregate root for the order bounded context.
type Order struct {
	id        OrderID
	userID    userdomain.UserID
	items     []OrderItem
	status    Status
	total     Money
	createdAt time.Time
	updatedAt time.Time
}

// OrderItem represents a line item in an order.
type OrderItem struct {
	ProductID   string
	ProductName string
	Quantity    int
	UnitPrice   Money
}

func (i OrderItem) Subtotal() Money {
	return i.UnitPrice.Multiply(int64(i.Quantity))
}

// NewOrder creates a new order for a user.
func NewOrder(userID userdomain.UserID) *Order {
	return &Order{
		id:        NewOrderID(),
		userID:    userID,
		items:     make([]OrderItem, 0),
		status:    StatusDraft,
		total:     MustNewMoney(0, "USD"),
		createdAt: time.Now().UTC(),
		updatedAt: time.Now().UTC(),
	}
}

// Reconstitute rebuilds an order from persistence.
func Reconstitute(
	id OrderID,
	userID userdomain.UserID,
	items []OrderItem,
	status Status,
	total Money,
	createdAt, updatedAt time.Time,
) *Order {
	return &Order{
		id:        id,
		userID:    userID,
		items:     items,
		status:    status,
		total:     total,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

// Getters

func (o *Order) ID() OrderID        { return o.id }
func (o *Order) UserID() userdomain.UserID     { return o.userID }
func (o *Order) Items() []OrderItem       { return o.items }
func (o *Order) Status() Status           { return o.status }
func (o *Order) Total() Money       { return o.total }
func (o *Order) CreatedAt() time.Time     { return o.createdAt }
func (o *Order) UpdatedAt() time.Time     { return o.updatedAt }

// Business methods

// AddItem adds an item to the order.
func (o *Order) AddItem(productID, productName string, quantity int, unitPrice Money) error {
	if o.status != StatusDraft {
		return ErrOrderNotDraft
	}
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	// Check if product already exists, update quantity
	for i, item := range o.items {
		if item.ProductID == productID {
			o.items[i].Quantity += quantity
			o.recalculateTotal()
			o.updatedAt = time.Now().UTC()
			return nil
		}
	}

	// Add new item
	o.items = append(o.items, OrderItem{
		ProductID:   productID,
		ProductName: productName,
		Quantity:    quantity,
		UnitPrice:   unitPrice,
	})
	o.recalculateTotal()
	o.updatedAt = time.Now().UTC()
	return nil
}

// RemoveItem removes an item from the order.
func (o *Order) RemoveItem(productID string) error {
	if o.status != StatusDraft {
		return ErrOrderNotDraft
	}

	for i, item := range o.items {
		if item.ProductID == productID {
			o.items = append(o.items[:i], o.items[i+1:]...)
			o.recalculateTotal()
			o.updatedAt = time.Now().UTC()
			return nil
		}
	}
	return ErrItemNotFound
}

// Submit submits the order for processing.
func (o *Order) Submit() error {
	if o.status != StatusDraft {
		return ErrOrderNotDraft
	}
	if len(o.items) == 0 {
		return ErrOrderEmpty
	}

	o.status = StatusPending
	o.updatedAt = time.Now().UTC()
	return nil
}

// Confirm confirms the order.
func (o *Order) Confirm() error {
	if o.status != StatusPending {
		return ErrOrderNotPending
	}

	o.status = StatusConfirmed
	o.updatedAt = time.Now().UTC()
	return nil
}

// Cancel cancels the order.
func (o *Order) Cancel() error {
	if o.status == StatusCancelled {
		return ErrOrderAlreadyCancelled
	}
	if o.status == StatusCompleted {
		return ErrOrderCompleted
	}

	o.status = StatusCancelled
	o.updatedAt = time.Now().UTC()
	return nil
}

// Complete marks the order as completed.
func (o *Order) Complete() error {
	if o.status != StatusConfirmed {
		return ErrOrderNotConfirmed
	}

	o.status = StatusCompleted
	o.updatedAt = time.Now().UTC()
	return nil
}

func (o *Order) recalculateTotal() {
	var total int64
	currency := "USD"
	for _, item := range o.items {
		subtotal := item.Subtotal()
		total += subtotal.Amount()
		currency = subtotal.Currency()
	}
	o.total = MustNewMoney(total, currency)
}
