package domain_test

import (
	"context"
	"testing"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

func TestNewUser(t *testing.T) {
	ctx := events.NewContext(context.Background())

	email, err := domain.NewEmail("test@example.com")
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	name, err := domain.NewName("John", "Doe")
	if err != nil {
		t.Fatalf("failed to create name: %v", err)
	}

	user := domain.NewUser(ctx, email, name)

	if user.ID().IsZero() {
		t.Error("expected user to have an ID")
	}
	if user.Email().String() != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", user.Email().String())
	}
	if user.Name().FullName() != "John Doe" {
		t.Errorf("expected name 'John Doe', got '%s'", user.Name().FullName())
	}
	if user.Status() != domain.StatusActive {
		t.Errorf("expected status 'active', got '%s'", user.Status())
	}

	collected := events.Collect(ctx)
	if len(collected) != 1 {
		t.Fatalf("expected 1 event, got %d", len(collected))
	}
	if collected[0].EventType() != domain.UserCreatedEventType {
		t.Errorf("expected UserCreatedEvent, got %s", collected[0].EventType())
	}
}

func TestUser_UpdateProfile(t *testing.T) {
	ctx := events.NewContext(context.Background())
	user := createTestUser(t, ctx)

	newName, err := domain.NewName("Jane", "Smith")
	if err != nil {
		t.Fatalf("failed to create name: %v", err)
	}

	err = user.UpdateProfile(ctx, newName)
	if err != nil {
		t.Fatalf("failed to update profile: %v", err)
	}

	if user.Name().FullName() != "Jane Smith" {
		t.Errorf("expected name 'Jane Smith', got '%s'", user.Name().FullName())
	}
}

func TestUser_Delete(t *testing.T) {
	ctx := events.NewContext(context.Background())
	user := createTestUser(t, ctx)

	err := user.Delete(ctx)
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	if user.Status() != domain.StatusDeleted {
		t.Errorf("expected status 'deleted', got '%s'", user.Status())
	}
}

func TestUser_UpdateProfile_Deleted(t *testing.T) {
	ctx := events.NewContext(context.Background())
	user := createTestUser(t, ctx)
	user.Delete(ctx)

	newName, _ := domain.NewName("Jane", "Smith")
	err := user.UpdateProfile(ctx, newName)

	if err != domain.ErrUserDeleted {
		t.Errorf("expected ErrUserDeleted, got %v", err)
	}
}

func TestEmail_Validation(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{"valid email", "test@example.com", nil},
		{"valid email with subdomain", "test@mail.example.com", nil},
		{"empty email", "", domain.ErrEmailRequired},
		{"invalid format", "not-an-email", domain.ErrEmailInvalid},
		{"missing @", "testexample.com", domain.ErrEmailInvalid},
		{"missing domain", "test@", domain.ErrEmailInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewEmail(tt.email)
			if err != tt.wantErr {
				t.Errorf("NewEmail(%q) error = %v, want %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestName_Validation(t *testing.T) {
	tests := []struct {
		name      string
		firstName string
		lastName  string
		wantErr   error
	}{
		{"valid name", "John", "Doe", nil},
		{"empty first name", "", "Doe", domain.ErrFirstNameRequired},
		{"empty last name", "John", "", domain.ErrLastNameRequired},
		{"short first name", "J", "Doe", domain.ErrFirstNameLength},
		{"short last name", "John", "D", domain.ErrLastNameLength},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewName(tt.firstName, tt.lastName)
			if err != tt.wantErr {
				t.Errorf("NewName(%q, %q) error = %v, want %v", tt.firstName, tt.lastName, err, tt.wantErr)
			}
		})
	}
}

func createTestUser(t *testing.T, ctx context.Context) *domain.User {
	t.Helper()

	email, err := domain.NewEmail("test@example.com")
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	name, err := domain.NewName("John", "Doe")
	if err != nil {
		t.Fatalf("failed to create name: %v", err)
	}

	return domain.NewUser(ctx, email, name)
}
