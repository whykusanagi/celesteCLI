FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Copy all source files
COPY *.go ./

# Build the Celeste binary (with embedded assets)
RUN go build -o Celeste main.go scaffolding.go animation.go ui.go assets.go interactive.go terminal_display.go ascii_art.go gif_animator.go embedded_assets.go

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install script utility for PTY support and grep
RUN apk add --no-cache util-linux grep bash

# Copy the compiled binary from builder
COPY --from=builder /app/Celeste .

# Copy the test script
COPY validation/test_animation.sh .
RUN chmod +x test_animation.sh

# Run the test
CMD ["./test_animation.sh"]
