# Build stage
FROM golang:1.24.5-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o /techlag ./cmd/technicalLag.go

# Final stage
FROM alpine:latest

# Add ca-certificates for any HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /techlag .

# Create a directory for input/output files
RUN mkdir -p /data

# Set the working directory to /data to make file access easier
WORKDIR /data

# Set the entrypoint
ENTRYPOINT ["/root/techlag"]

# Default command (can be overridden)
CMD ["--help"]