package commands_test

import (
	"errors"
	"testing"
	"time"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events/eventstest"
	userevents "github.com/rai/clean-modularmonolith-go/modules/users/domain/events"
	"github.com/rai/clean-modularmonolith-go/modules/users/application/commands"
	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
	domainmocks "github.com/rai/clean-modularmonolith-go/modules/users/domain/mocks"
	"go.uber.org/mock/gomock"
)

func TestDeleteUserHandler_Handle_Success(t *testing.T) {
	ctrl := gomock.NewController(t)

	userID := domain.NewUserID()
	user := createTestUser(t, userID)

	repo := domainmocks.NewMockUserRepository(ctrl)
	gomock.InOrder(
		repo.EXPECT().FindByID(gomock.Any(), userID).Return(user, nil),
		repo.EXPECT().Save(gomock.Any(), deletedUser(userID)).Return(nil),
	)

	scope, capture := eventstest.NewScopeCaptureEvents(ctrl)
	handler := commands.NewDeleteUserHandler(repo, scope)

	err := handler.Handle(t.Context(), commands.DeleteUserCommand{UserID: userID.String()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capture.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(capture.Events))
	}
	deletedEvent, ok := capture.Events[0].(userevents.UserDeletedEvent)
	if !ok {
		t.Fatalf("expected contracts.UserDeletedEvent, got %T", capture.Events[0])
	}
	if deletedEvent.UserID != userID.String() {
		t.Errorf("expected event userID %s, got %s", userID, deletedEvent.UserID)
	}
}

func TestDeleteUserHandler_Handle_InvalidUserID(t *testing.T) {
	handler := commands.NewDeleteUserHandler(nil, nil)

	err := handler.Handle(t.Context(), commands.DeleteUserCommand{UserID: "invalid-uuid"})

	if err == nil {
		t.Fatal("expected error for invalid user ID")
	}
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Errorf("expected ErrInvalidUserID, got %v", err)
	}
}

func TestDeleteUserHandler_Handle_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)

	userID := domain.NewUserID()
	errNotFound := errors.New("user not found")

	repo := domainmocks.NewMockUserRepository(ctrl)
	repo.EXPECT().FindByID(gomock.Any(), userID).Return(nil, errNotFound)

	scope, capture := eventstest.NewScopeCaptureEvents(ctrl)
	handler := commands.NewDeleteUserHandler(repo, scope)

	err := handler.Handle(t.Context(), commands.DeleteUserCommand{UserID: userID.String()})

	if err == nil {
		t.Fatal("expected error when user not found")
	}
	if !errors.Is(err, errNotFound) {
		t.Errorf("expected errNotFound, got %v", err)
	}
	if len(capture.Events) != 0 {
		t.Errorf("expected no events on failure, got %d", len(capture.Events))
	}
}

func TestDeleteUserHandler_Handle_SaveError(t *testing.T) {
	ctrl := gomock.NewController(t)

	userID := domain.NewUserID()
	user := createTestUser(t, userID)
	errSave := errors.New("save failed")

	repo := domainmocks.NewMockUserRepository(ctrl)
	gomock.InOrder(
		repo.EXPECT().FindByID(gomock.Any(), userID).Return(user, nil),
		repo.EXPECT().Save(gomock.Any(), deletedUser(userID)).Return(errSave),
	)

	scope, capture := eventstest.NewScopeCaptureEvents(ctrl)
	handler := commands.NewDeleteUserHandler(repo, scope)

	err := handler.Handle(t.Context(), commands.DeleteUserCommand{UserID: userID.String()})

	if err == nil {
		t.Fatal("expected error when save fails")
	}
	if !errors.Is(err, errSave) {
		t.Errorf("expected errSave, got %v", err)
	}
	if len(capture.Events) != 0 {
		t.Errorf("expected no events on failure, got %d", len(capture.Events))
	}
}

func TestDeleteUserHandler_Handle_TransactionError(t *testing.T) {
	ctrl := gomock.NewController(t)

	userID := domain.NewUserID()
	errTx := errors.New("transaction failed")

	handler := commands.NewDeleteUserHandler(nil, eventstest.NewScopeError(ctrl, errTx))

	err := handler.Handle(t.Context(), commands.DeleteUserCommand{UserID: userID.String()})

	if err == nil {
		t.Fatal("expected error when transaction fails")
	}
	if !errors.Is(err, errTx) {
		t.Errorf("expected errTx, got %v", err)
	}
}

// --- Matchers ---

// deletedUser matches a *domain.User with the given ID that has been soft-deleted.
func deletedUser(id domain.UserID) gomock.Matcher {
	return gomock.Cond(func(x any) bool {
		u, ok := x.(*domain.User)
		return ok && u.ID() == id && u.Status() == domain.StatusDeleted
	})
}

// --- Helper ---

func createTestUser(t *testing.T, id domain.UserID) *domain.User {
	t.Helper()

	email, err := domain.NewEmail("test@example.com")
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	name, err := domain.NewName("John", "Doe")
	if err != nil {
		t.Fatalf("failed to create name: %v", err)
	}

	return domain.Reconstitute(id, email, name, domain.StatusActive, time.Now(), time.Now())
}
