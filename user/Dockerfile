# User Service: Hardened Go multi-stage build
# Minimizes attack surface with non-root execution

# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod ./

# Download dependencies
RUN go mod tidy

# Copy source code
COPY main.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /user main.go

# Production stage
FROM alpine:3.19

# Security: create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the compiled binary
COPY --from=builder /user .

# Security: run as non-root
USER appuser

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./user"]
