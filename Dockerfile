# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
# Copy gen directory for local modules
COPY interverse-report/gen/ ./gen/

COPY interverse-report/go.mod interverse-report/go.sum ./
RUN go mod download

# Copy source code
COPY interverse-report/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o report-service .

# Final stage
FROM alpine:3.19

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN adduser -D -s /bin/sh appuser
USER appuser

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/report-service .

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:50054 || exit 1

EXPOSE 50054

CMD ["./report-service"]
