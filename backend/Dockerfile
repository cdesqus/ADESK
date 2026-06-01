FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Copy go mod files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Remove the corrupted go.sum copied from the host and regenerate it
RUN rm -f go.sum && go mod download

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

EXPOSE 8000

CMD ["./main"]
