# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev libpcap-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# Note: CGO_ENABLED=0 for static binary, but libpcap requires CGO
# For production, consider using alpine base instead of scratch
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o netmeta ./cmd/netmeta

# Final stage - use alpine for libpcap support
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates libpcap

WORKDIR /app

# Copy the binary
COPY --from=builder /build/netmeta /app/netmeta
COPY --from=builder /build/templates /app/templates
COPY --from=builder /build/config.yaml /app/config.yaml

# Expose ports
EXPOSE 8080 9090

ENTRYPOINT ["/app/netmeta"]

