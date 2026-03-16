// Package eventstest provides test helpers for transaction.ScopeWithDomainEvent.
package eventstest

import (
	"context"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/transaction"
	txmocks "github.com/rai/clean-modularmonolith-go/modules/shared/transaction/mocks"
	"go.uber.org/mock/gomock"
)

// Capture holds domain events collected during a test execution.
type Capture struct {
	Events []events.Event
}

// NewScopeCaptureEvents creates a mock ScopeWithDomainEvent and a Capture that will
// hold the domain events emitted during execution. Use this in all command handler tests
// — it enforces that event emission is always observable.
//
//	scope, capture := eventstest.NewScopeCaptureEvents(ctrl)
//	handler.Handle(ctx, cmd)
//	// assert on capture.Events
func NewScopeCaptureEvents(ctrl *gomock.Controller) (transaction.ScopeWithDomainEvent, *Capture) {
	capture := &Capture{}
	m := txmocks.NewMockScopeWithDomainEvent(ctrl)

	fn := func(ctx context.Context, fn func(ctx context.Context) error) error {
		evts, err := events.CaptureEvents(ctx, fn)
		if err != nil {
			return err
		}
		capture.Events = evts
		return nil
	}
	m.EXPECT().ExecuteWithPublish(gomock.Any(), gomock.Any()).DoAndReturn(fn)
	return m, capture
}

// NewScopeError creates a mock ScopeWithDomainEvent pre-configured to return err
// without executing fn. Use when testing transaction-level failure.
func NewScopeError(ctrl *gomock.Controller, err error) transaction.ScopeWithDomainEvent {
	m := txmocks.NewMockScopeWithDomainEvent(ctrl)
	m.EXPECT().ExecuteWithPublish(gomock.Any(), gomock.Any()).Return(err)
	return m
}
