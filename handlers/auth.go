// handlers/auth - Registration and login endpoints.
// Handles user signup with password hashing and login with JWT token generation.
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gauravjain0377/ticket-system/models"
	"github.com/gauravjain0377/ticket-system/store"
	"github.com/gauravjain0377/ticket-system/utils"
)

// AuthHandler holds dependencies needed by auth endpoints.
type AuthHandler struct {
	DB        *store.DB
	JWTSecret string
}

// NewAuthHandler creates a new AuthHandler with the given database and JWT secret.
func NewAuthHandler(db *store.DB, jwtSecret string) *AuthHandler {
	return &AuthHandler{DB: db, JWTSecret: jwtSecret}
}

// Register handles POST /auth/register
// Expects JSON body: {"username": "...", "email": "...", "password": "..."}
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest

	// Parse the JSON request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Email == "" || req.Password == "" {
		utils.Error(w, http.StatusBadRequest, "username, email, and password are required")
		return
	}

	// Check if username already exists
	existingUser, err := h.DB.GetUserByUsername(req.Username)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to check existing user")
		return
	}
	if existingUser != nil {
		utils.Error(w, http.StatusConflict, "username already exists")
		return
	}

	// Hash the password before storing
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	// Create the user in the database
	user, err := h.DB.CreateUser(req.Username, req.Email, hashedPassword)
	if err != nil {
		utils.Error(w, http.StatusConflict, "username or email already exists")
		return
	}

	utils.JSON(w, http.StatusCreated, user)
}

// Login handles POST /auth/login
// Expects JSON body: {"username": "...", "password": "..."}
// Returns a JWT token on success.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Password == "" {
		utils.Error(w, http.StatusBadRequest, "username and password are required")
		return
	}

	// Look up the user by username
	user, err := h.DB.GetUserByUsername(req.Username)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to find user")
		return
	}
	if user == nil {
		utils.Error(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	// Verify the password against the stored hash
	if !utils.CheckPassword(user.PasswordHash, req.Password) {
		utils.Error(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	// Generate a JWT token for the authenticated user
	token, err := utils.GenerateToken(user.ID, user.Username, h.JWTSecret)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	utils.JSON(w, http.StatusOK, models.LoginResponse{Token: token})
}
