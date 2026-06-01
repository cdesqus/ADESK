FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Copy go mod files
COPY go.mod ./

# Copy source code
COPY . .

# Fix checksum issues and download dependencies
RUN rm -f go.sum && go clean -modcache && go mod tidy

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates postgresql-client

# Copy binary from builder
COPY --from=builder /app/main .

# Copy config files if needed
COPY --from=builder /app/.env.example ./.env.example || true

EXPOSE 8080

CMD ["./main"]
