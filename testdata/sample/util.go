package sample

import (
	"strings"
)

// User represents a user in the system
type User struct {
	ID   int
	Name string
	Email string
}

// IsValid checks if the user has valid data
// This is a method with a receiver
func (u *User) IsValid() bool {
	if u.ID <= 0 {
		return false
	}
	if u.Name == "" {
		return false
	}
	if !strings.Contains(u.Email, "@") {
		return false
	}
	return true
}

// GetDisplayName returns a formatted display name
func (u *User) GetDisplayName() string {
	return strings.Title(u.Name)
}

// UpdateEmail updates the user's email address
func (u *User) UpdateEmail(newEmail string) error {
	if !strings.Contains(newEmail, "@") {
		return ErrInvalidEmail
	}
	u.Email = newEmail
	return nil
}

// MaxInt returns the maximum of two integers
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MinInt returns the minimum of two integers
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Contains checks if a slice contains a value
func Contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
