package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events/contracts"
	"github.com/rai/clean-modularmonolith-go/modules/users/application/commands"
	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// --- Mocks ---

type mockUserRepository struct {
	findByIDFn func(ctx context.Context, id domain.UserID) (*domain.User, error)
	saveFn     func(ctx context.Context, user *domain.User) error
}

func (m *mockUserRepository) FindByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
	return m.findByIDFn(ctx, id)
}

func (m *mockUserRepository) Save(ctx context.Context, user *domain.User) error {
	return m.saveFn(ctx, user)
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	return nil, nil
}

func (m *mockUserRepository) Exists(ctx context.Context, email domain.Email) (bool, error) {
	return false, nil
}

func (m *mockUserRepository) FindAll(ctx context.Context, offset, limit int) ([]*domain.User, int, error) {
	return nil, 0, nil
}

type mockTransactionScope struct {
	executeFn func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *mockTransactionScope) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.executeFn(ctx, fn)
}

type mockPublisher struct {
	publishFn func(ctx context.Context, evts ...events.Event) error
}

func (m *mockPublisher) Publish(ctx context.Context, evts ...events.Event) error {
	return m.publishFn(ctx, evts...)
}

// --- Tests ---

func TestDeleteUserHandler_Handle_Success(t *testing.T) {
	// Arrange
	userID := domain.NewUserID()
	user := createTestUser(t, userID)

	var savedUser *domain.User
	var publishedEvents []events.Event

	repo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id domain.UserID) (*domain.User, error) {
			if id.String() != userID.String() {
				t.Errorf("expected userID %s, got %s", userID, id)
			}
			return user, nil
		},
		saveFn: func(ctx context.Context, u *domain.User) error {
			savedUser = u
			return nil
		},
	}

	publisher := &mockPublisher{
		publishFn: func(ctx context.Context, evts ...events.Event) error {
			publishedEvents = evts
			return nil
		},
	}

	txScope := &mockTransactionScope{
		executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	}

	handler := commands.NewDeleteUserHandler(repo, txScope, publisher)

	// Act
	err := handler.Handle(context.Background(), commands.DeleteUserCommand{
		UserID: userID.String(),
	})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if savedUser == nil {
		t.Fatal("expected user to be saved")
	}
	if savedUser.Status() != domain.StatusDeleted {
		t.Errorf("expected user status to be deleted, got %s", savedUser.Status())
	}

	if len(publishedEvents) != 1 {
		t.Fatalf("expected 1 event, got %d", len(publishedEvents))
	}
	deletedEvent, ok := publishedEvents[0].(contracts.UserDeletedEvent)
	if !ok {
		t.Fatalf("expected UserDeletedEvent, got %T", publishedEvents[0])
	}
	if deletedEvent.UserID != userID.String() {
		t.Errorf("expected event userID %s, got %s", userID, deletedEvent.UserID)
	}
}

func TestDeleteUserHandler_Handle_InvalidUserID(t *testing.T) {
	handler := commands.NewDeleteUserHandler(nil, nil, nil)

	err := handler.Handle(context.Background(), commands.DeleteUserCommand{
		UserID: "invalid-uuid",
	})

	if err == nil {
		t.Fatal("expected error for invalid user ID")
	}
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Errorf("expected ErrInvalidUserID, got %v", err)
	}
}

func TestDeleteUserHandler_Handle_UserNotFound(t *testing.T) {
	userID := domain.NewUserID()
	errNotFound := errors.New("user not found")

	repo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id domain.UserID) (*domain.User, error) {
			return nil, errNotFound
		},
		saveFn: func(ctx context.Context, u *domain.User) error {
			t.Fatal("Save should not be called when user is not found")
			return nil
		},
	}

	publisher := &mockPublisher{
		publishFn: func(ctx context.Context, evts ...events.Event) error {
			t.Fatal("Publish should not be called when user is not found")
			return nil
		},
	}

	txScope := &mockTransactionScope{
		executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	}

	handler := commands.NewDeleteUserHandler(repo, txScope, publisher)

	err := handler.Handle(context.Background(), commands.DeleteUserCommand{
		UserID: userID.String(),
	})

	if err == nil {
		t.Fatal("expected error when user not found")
	}
	if !errors.Is(err, errNotFound) {
		t.Errorf("expected errNotFound, got %v", err)
	}
}

func TestDeleteUserHandler_Handle_SaveError(t *testing.T) {
	userID := domain.NewUserID()
	user := createTestUser(t, userID)
	errSave := errors.New("save failed")

	repo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id domain.UserID) (*domain.User, error) {
			return user, nil
		},
		saveFn: func(ctx context.Context, u *domain.User) error {
			return errSave
		},
	}

	publisher := &mockPublisher{
		publishFn: func(ctx context.Context, evts ...events.Event) error {
			t.Fatal("Publish should not be called when save fails")
			return nil
		},
	}

	txScope := &mockTransactionScope{
		executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	}

	handler := commands.NewDeleteUserHandler(repo, txScope, publisher)

	err := handler.Handle(context.Background(), commands.DeleteUserCommand{
		UserID: userID.String(),
	})

	if err == nil {
		t.Fatal("expected error when save fails")
	}
	if !errors.Is(err, errSave) {
		t.Errorf("expected errSave, got %v", err)
	}
}

func TestDeleteUserHandler_Handle_PublishError(t *testing.T) {
	userID := domain.NewUserID()
	user := createTestUser(t, userID)
	errPublish := errors.New("publish failed")

	repo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id domain.UserID) (*domain.User, error) {
			return user, nil
		},
		saveFn: func(ctx context.Context, u *domain.User) error {
			return nil
		},
	}

	publisher := &mockPublisher{
		publishFn: func(ctx context.Context, evts ...events.Event) error {
			return errPublish
		},
	}

	txScope := &mockTransactionScope{
		executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	}

	handler := commands.NewDeleteUserHandler(repo, txScope, publisher)

	err := handler.Handle(context.Background(), commands.DeleteUserCommand{
		UserID: userID.String(),
	})

	if err == nil {
		t.Fatal("expected error when publish fails")
	}
	if !errors.Is(err, errPublish) {
		t.Errorf("expected errPublish, got %v", err)
	}
}

func TestDeleteUserHandler_Handle_TransactionError(t *testing.T) {
	userID := domain.NewUserID()
	errTx := errors.New("transaction failed")

	txScope := &mockTransactionScope{
		executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return errTx // トランザクション自体が失敗
		},
	}

	handler := commands.NewDeleteUserHandler(nil, txScope, nil)

	err := handler.Handle(context.Background(), commands.DeleteUserCommand{
		UserID: userID.String(),
	})

	if err == nil {
		t.Fatal("expected error when transaction fails")
	}
	if !errors.Is(err, errTx) {
		t.Errorf("expected errTx, got %v", err)
	}
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

	user := domain.Reconstitute(id, email, name, domain.StatusActive, time.Now(), time.Now())
	return user
}
