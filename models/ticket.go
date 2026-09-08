// models/ticket - Defines the Ticket data structure, status constants, and transition rules.
//
// Status flow: open → in_progress → closed (one-way, closed tickets cannot be reopened).
package models

import "time"

const (
	StatusOpen       = "open"
	StatusInProgress = "in_progress"
	StatusClosed     = "closed"
)

// ValidStatusTransitions maps each status to its only allowed next status.
var ValidStatusTransitions = map[string]string{
	StatusOpen:       StatusInProgress,
	StatusInProgress: StatusClosed,
}

type Ticket struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedBy   int64     `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateTicketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// IsValidStatus checks if a status string is one of the allowed values.
func IsValidStatus(status string) bool {
	return status == StatusOpen || status == StatusInProgress || status == StatusClosed
}

// CanTransition checks if moving from one status to another is allowed.
func CanTransition(from, to string) bool {
	allowedNext, exists := ValidStatusTransitions[from]
	return exists && allowedNext == to
}
