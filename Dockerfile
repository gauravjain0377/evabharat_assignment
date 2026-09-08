# ─────────────────────────────────────────────
# Stage 1: Build the Go binary
# ─────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

# Set the working directory inside the builder container
WORKDIR /app

# Copy dependency files first (better Docker layer caching)
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary — CGO_ENABLED=0 ensures a fully static binary (no C deps)
RUN CGO_ENABLED=0 GOOS=linux go build -o ticket-system .

# ─────────────────────────────────────────────
# Stage 2: Minimal runtime image
# ─────────────────────────────────────────────
FROM alpine:3.20

WORKDIR /app

# Copy only the compiled binary from the builder stage
COPY --from=builder /app/ticket-system .

# Copy frontend static files so the server can serve them
COPY --from=builder /app/frontend ./frontend

# Expose port 8080
EXPOSE 8080

# Set default environment variables (can be overridden at docker run time)
ENV PORT=8080
ENV JWT_SECRET=change-me-in-production
ENV DB_PATH=/app/data/tickets.db

# Create directory for the SQLite database file
RUN mkdir -p /app/data

# Run the server
CMD ["./ticket-system"]
