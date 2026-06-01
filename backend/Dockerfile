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

# Build the application using -mod=mod to automatically update missing go.sum entries
RUN CGO_ENABLED=1 GOOS=linux go build -mod=mod -a -installsuffix cgo -o main ./cmd/main.go

FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates postgresql-client

# Copy binary from builder
COPY --from=builder /app/main .

# Copy config files if needed (using wildcard to prevent failure if it doesn't exist)
COPY --from=builder /app/.env.exampl[e] ./

EXPOSE 8000

CMD ["./main"]
