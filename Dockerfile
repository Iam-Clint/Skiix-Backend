# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/skiix/main.go

# Final stage
FROM alpine:latest

# Install CA certificates and tzdata for SSL/Timezone support
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/main .

# Expose port 8080 (the port our Gin server runs on)
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
