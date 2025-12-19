package sample

import "errors"

// Common errors
var (
	ErrInvalidEmail = errors.New("invalid email address")
	ErrUserNotFound = errors.New("user not found")
	ErrAccessDenied = errors.New("access denied")
)
