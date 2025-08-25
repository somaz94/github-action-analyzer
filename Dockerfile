# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -o /analyzer ./cmd/analyzer

# Final stage
FROM alpine:latest

# Install required packages
RUN apk add --no-cache \
    git \
    curl

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /analyzer /analyzer

ENTRYPOINT ["/analyzer"]