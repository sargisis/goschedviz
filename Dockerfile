# Stage 1: Build the application
FROM golang:alpine AS builder

# Install git for fetching dependencies
RUN apk add --no-cache git


WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
RUN go build -o goschedviz ./cmd/goschedviz

# Stage 2: Create a minimal runtime image
FROM alpine:latest

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/goschedviz .

# Allow running the tool
ENTRYPOINT ["./goschedviz"]
