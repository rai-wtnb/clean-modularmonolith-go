package domain

// Status represents the order status.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPending   Status = "pending"
	StatusConfirmed Status = "confirmed"
	StatusCompleted Status = "completed"
	StatusCancelled Status = "cancelled"
)

func (s Status) String() string { return string(s) }

func (s Status) IsValid() bool {
	switch s {
	case StatusDraft, StatusPending, StatusConfirmed, StatusCompleted, StatusCancelled:
		return true
	default:
		return false
	}
}
