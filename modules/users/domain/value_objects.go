package domain

import (
	"regexp"
	"strings"
)

// Email is a value object representing a validated email address.
// Value objects are immutable and compared by value.
type Email struct {
	value string
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// NewEmail creates a validated Email value object.
func NewEmail(value string) (Email, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return Email{}, ErrEmailRequired
	}
	if !emailRegex.MatchString(value) {
		return Email{}, ErrEmailInvalid
	}
	return Email{value: value}, nil
}

func (e Email) String() string { return e.value }
func (e Email) IsZero() bool   { return e.value == "" }

func (e Email) Equals(other Email) bool {
	return e.value == other.value
}

// Name is a value object representing a user's name.
type Name struct {
	firstName string
	lastName  string
}

// NewName creates a validated Name value object.
func NewName(firstName, lastName string) (Name, error) {
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)

	if firstName == "" {
		return Name{}, ErrFirstNameRequired
	}
	if len(firstName) < 2 || len(firstName) > 50 {
		return Name{}, ErrFirstNameLength
	}
	if lastName == "" {
		return Name{}, ErrLastNameRequired
	}
	if len(lastName) < 2 || len(lastName) > 50 {
		return Name{}, ErrLastNameLength
	}
	return Name{firstName: firstName, lastName: lastName}, nil
}

func (n Name) FirstName() string { return n.firstName }
func (n Name) LastName() string  { return n.lastName }
func (n Name) FullName() string  { return n.firstName + " " + n.lastName }
func (n Name) IsZero() bool      { return n.firstName == "" && n.lastName == "" }

func (n Name) Equals(other Name) bool {
	return n.firstName == other.firstName && n.lastName == other.lastName
}

// Status represents the user account status.
type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusDeleted  Status = "deleted"
)

func (s Status) String() string { return string(s) }

func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusInactive, StatusDeleted:
		return true
	default:
		return false
	}
}
