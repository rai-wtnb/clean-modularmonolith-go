package types

import "errors"

// Sentinel errors for common validation failures.
// Define errors in the types package where the validated types live.
var (
	ErrInvalidID = errors.New("invalid identifier format")
)
