// handlers/ticket - CRUD endpoints for tickets.
// All routes are protected by JWT middleware. Users can only access their own tickets.
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/gauravjain0377/ticket-system/middleware"
	"github.com/gauravjain0377/ticket-system/models"
	"github.com/gauravjain0377/ticket-system/store"
	"github.com/gauravjain0377/ticket-system/utils"
)

// TicketHandler holds dependencies needed by ticket endpoints.
type TicketHandler struct {
	DB *store.DB
}

// NewTicketHandler creates a new TicketHandler with the given database.
func NewTicketHandler(db *store.DB) *TicketHandler {
	return &TicketHandler{DB: db}
}

// CreateTicket handles POST /tickets
// Creates a new ticket with status "open" for the logged-in user.
func (h *TicketHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	var req models.CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)

	if req.Title == "" {
		utils.Error(w, http.StatusBadRequest, "title is required")
		return
	}

	ticket, err := h.DB.CreateTicket(req.Title, req.Description, userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to create ticket")
		return
	}

	utils.JSON(w, http.StatusCreated, ticket)
}

// ListTickets handles GET /tickets
// Returns all tickets belonging to the logged-in user.
func (h *TicketHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	tickets, err := h.DB.GetTicketsByUserID(userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to list tickets")
		return
	}

	utils.JSON(w, http.StatusOK, tickets)
}

// GetTicket handles GET /tickets/{id}
// Returns a single ticket only if it belongs to the logged-in user.
func (h *TicketHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	// Extract ticket ID from the URL path
	ticketID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid ticket ID")
		return
	}

	ticket, err := h.DB.GetTicketByID(ticketID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to get ticket")
		return
	}

	if ticket == nil {
		utils.Error(w, http.StatusNotFound, "ticket not found")
		return
	}

	// Ownership check — users can only view their own tickets
	if ticket.CreatedBy != userID {
		utils.Error(w, http.StatusForbidden, "you can only view your own tickets")
		return
	}

	utils.JSON(w, http.StatusOK, ticket)
}

// UpdateStatus handles PATCH /tickets/{id}/status
// Updates a ticket's status following the allowed transition rules:
//   open → in_progress → closed (one-way, closed cannot reopen)
func (h *TicketHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	ticketID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid ticket ID")
		return
	}

	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Status = strings.TrimSpace(req.Status)

	if !models.IsValidStatus(req.Status) {
		utils.Error(w, http.StatusBadRequest, "invalid status, must be: open, in_progress, or closed")
		return
	}

	// Fetch the ticket to check ownership and current status
	ticket, err := h.DB.GetTicketByID(ticketID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to get ticket")
		return
	}

	if ticket == nil {
		utils.Error(w, http.StatusNotFound, "ticket not found")
		return
	}

	// Ownership check
	if ticket.CreatedBy != userID {
		utils.Error(w, http.StatusForbidden, "you can only update your own tickets")
		return
	}

	// Validate the status transition
	if !models.CanTransition(ticket.Status, req.Status) {
		utils.Error(w, http.StatusBadRequest,
			"invalid status transition: cannot move from '"+ticket.Status+"' to '"+req.Status+"'")
		return
	}

	// Perform the update
	if err := h.DB.UpdateTicketStatus(ticketID, req.Status); err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to update ticket status")
		return
	}

	// Return the updated ticket
	updatedTicket, err := h.DB.GetTicketByID(ticketID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to get updated ticket")
		return
	}

	utils.JSON(w, http.StatusOK, updatedTicket)
}
