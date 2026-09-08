# Ticket System

A REST API backend for a ticket management system built with **Go**. Users can register, login, create tickets, view their own tickets, and update ticket statuses.

## Screenshots

![Sign In Page](assets/ticket.png)

## Tech Stack

- **Language:** Go
- **Router:** gorilla/mux
- **Database:** SQLite (via modernc.org/sqlite — pure Go, no CGO)
- **Auth:** JWT (golang-jwt/jwt/v5)
- **Password Hashing:** bcrypt

---

## Local Development

### Prerequisites
- Go 1.21+

### Run locally

```bash
# 1. Clone the repo
git clone https://github.com/gauravjain0377/ticket-system
cd ticket-system

# 2. Install dependencies
go mod tidy

# 3. (Optional) Create a .env file
cp .env.example .env
# Edit .env if you want a custom JWT_SECRET

# 4. Start the server
go run main.go
```

Server starts at **http://localhost:8080**

---

## Docker

```bash
# Build the image
docker build -t ticket-system .

# Run the container
docker run -p 8080:8080 ticket-system

# With custom JWT secret
docker run -p 8080:8080 -e JWT_SECRET=your-secret ticket-system
```

---

## Environment Variables

| Variable     | Default                              | Description                    |
|-------------|--------------------------------------|-------------------------------|
| `PORT`       | `8080`                               | Port the server listens on     |
| `JWT_SECRET` | `default-secret-change-me-in-production` | Secret key for JWT signing |
| `DB_PATH`    | `tickets.db`                         | Path to the SQLite database    |

---

## API Reference

### Health Check

```
GET /health
```
Response: `{"status": "ok"}`

---

### Auth

#### Register
```
POST /auth/register
Content-Type: application/json

{
  "username": "john",
  "email": "john@example.com",
  "password": "password123"
}
```

#### Login
```
POST /auth/login
Content-Type: application/json

{
  "username": "john",
  "password": "password123"
}
```
Response: `{"token": "<jwt-token>"}`

All protected routes require the header:
```
Authorization: Bearer <token>
```

---

### Tickets

#### Create Ticket
```
POST /tickets
Authorization: Bearer <token>

{
  "title": "Fix login bug",
  "description": "The login button doesn't work on mobile"
}
```

#### List My Tickets
```
GET /tickets
Authorization: Bearer <token>
```

#### Get Ticket by ID
```
GET /tickets/{id}
Authorization: Bearer <token>
```

#### Update Ticket Status
```
PATCH /tickets/{id}/status
Authorization: Bearer <token>

{
  "status": "in_progress"
}
```

**Status Flow:**
```
open → in_progress → closed
```
- `closed` tickets cannot be reopened
- Users can only view/update their own tickets

---

## Status Codes

| Code | Meaning |
|------|---------|
| 200  | OK |
| 201  | Created |
| 400  | Bad Request (invalid input or status transition) |
| 401  | Unauthorized (missing or invalid token) |
| 403  | Forbidden (ticket belongs to another user) |
| 404  | Not Found |
| 409  | Conflict (duplicate username/email) |
| 500  | Internal Server Error |

---

## Project Structure

```
ticket-system/
├── main.go              # Entry point, router setup
├── config/config.go     # Environment variable loading
├── models/              # Data structs (User, Ticket)
├── store/sqlite.go      # Database layer
├── handlers/            # HTTP handlers (health, auth, ticket)
├── middleware/auth.go   # JWT validation middleware
├── utils/               # JWT, bcrypt, JSON response helpers
├── frontend/            # HTML/CSS/JS served by the Go server
├── Dockerfile           # Multi-stage Docker build
└── .env.example         # Environment variable template
```

---

## Assumptions

- No admin role — all users are equal
- Tickets are private — each user sees only their own
- SQLite is used for simplicity (single-file persistent database, no external DB needed)
- Passwords are hashed with bcrypt before storage (never stored in plain text)
- JWT tokens expire after 24 hours
- The frontend is served directly by the Go server at `/`
