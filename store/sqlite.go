// store/sqlite - Handles all database operations using SQLite.
// Sets up tables, and provides CRUD functions for users and tickets.
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/gauravjain0377/ticket-system/models"
)

type DB struct {
	conn *sql.DB
}

// NewDB opens the SQLite database and creates tables if they don't exist.
func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db := &DB{conn: conn}

	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

// createTables sets up the users and tickets tables.
func (db *DB) createTables() error {
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		username      TEXT    NOT NULL UNIQUE,
		email         TEXT    NOT NULL UNIQUE,
		password_hash TEXT    NOT NULL,
		created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	ticketsTable := `
	CREATE TABLE IF NOT EXISTS tickets (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		title       TEXT    NOT NULL,
		description TEXT    NOT NULL DEFAULT '',
		status      TEXT    NOT NULL DEFAULT 'open',
		created_by  INTEGER NOT NULL,
		created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (created_by) REFERENCES users(id)
	);`

	if _, err := db.conn.Exec(usersTable); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}
	if _, err := db.conn.Exec(ticketsTable); err != nil {
		return fmt.Errorf("failed to create tickets table: %w", err)
	}

	return nil
}

// Close shuts down the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// CreateUser inserts a new user and returns the created user object.
func (db *DB) CreateUser(username, email, passwordHash string) (*models.User, error) {
	now := time.Now()
	result, err := db.conn.Exec(
		"INSERT INTO users (username, email, password_hash, created_at) VALUES (?, ?, ?, ?)",
		username, email, passwordHash, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user (username or email may already exist): %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get new user ID: %w", err)
	}

	return &models.User{
		ID:        id,
		Username:  username,
		Email:     email,
		CreatedAt: now,
	}, nil
}

// GetUserByUsername looks up a user by username. Returns nil if not found.
func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	user := &models.User{}

	err := db.conn.QueryRow(
		"SELECT id, username, email, password_hash, created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}

// CreateTicket inserts a new ticket with status "open" and returns it.
func (db *DB) CreateTicket(title, description string, userID int64) (*models.Ticket, error) {
	now := time.Now()

	result, err := db.conn.Exec(
		"INSERT INTO tickets (title, description, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		title, description, models.StatusOpen, userID, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get new ticket ID: %w", err)
	}

	return &models.Ticket{
		ID:          id,
		Title:       title,
		Description: description,
		Status:      models.StatusOpen,
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// GetTicketsByUserID returns all tickets belonging to a specific user.
func (db *DB) GetTicketsByUserID(userID int64) ([]models.Ticket, error) {
	rows, err := db.conn.Query(
		"SELECT id, title, description, status, created_by, created_at, updated_at FROM tickets WHERE created_by = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list tickets: %w", err)
	}
	defer rows.Close()

	var tickets []models.Ticket
	for rows.Next() {
		var t models.Ticket
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to read ticket: %w", err)
		}
		tickets = append(tickets, t)
	}

	// Return empty slice instead of nil so JSON encodes as [] not null
	if tickets == nil {
		tickets = []models.Ticket{}
	}

	return tickets, nil
}

// GetTicketByID returns a single ticket by its ID. Returns nil if not found.
func (db *DB) GetTicketByID(ticketID int64) (*models.Ticket, error) {
	ticket := &models.Ticket{}

	err := db.conn.QueryRow(
		"SELECT id, title, description, status, created_by, created_at, updated_at FROM tickets WHERE id = ?",
		ticketID,
	).Scan(&ticket.ID, &ticket.Title, &ticket.Description, &ticket.Status, &ticket.CreatedBy, &ticket.CreatedAt, &ticket.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find ticket: %w", err)
	}

	return ticket, nil
}

// UpdateTicketStatus changes the status of a ticket and updates the timestamp.
func (db *DB) UpdateTicketStatus(ticketID int64, status string) error {
	_, err := db.conn.Exec(
		"UPDATE tickets SET status = ?, updated_at = ? WHERE id = ?",
		status, time.Now(), ticketID,
	)
	if err != nil {
		return fmt.Errorf("failed to update ticket status: %w", err)
	}

	return nil
}

// Ping checks if the database connection is alive.
func (db *DB) Ping() error {
	return db.conn.Ping()
}

// GetCounts returns total count of users and tickets.
func (db *DB) GetCounts() (int, int, error) {
	var userCount, ticketCount int
	if err := db.conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount); err != nil {
		return 0, 0, fmt.Errorf("failed to count users: %w", err)
	}
	if err := db.conn.QueryRow("SELECT COUNT(*) FROM tickets").Scan(&ticketCount); err != nil {
		return 0, 0, fmt.Errorf("failed to count tickets: %w", err)
	}
	return userCount, ticketCount, nil
}

