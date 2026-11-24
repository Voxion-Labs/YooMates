# syntax=docker/dockerfile:1

# Stage 1: Build the static Go binary
FROM golang:alpine AS builder

ENV GOTOOLCHAIN=auto

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Download dependencies (if any)
COPY go.mod ./
RUN if [ -f go.sum ]; then go mod download; fi

# Copy source tree
COPY . .

# Build statically linked binary with optimizations (stripped symbols)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /app/yoomates ./cmd/server/main.go

# Stage 2: Minimalist, hardened scratch/alpine runtime
FROM alpine:3.20 AS runner

RUN apk add --no-cache ca-certificates tzdata

# Create unprivileged runtime user
RUN addgroup -S voxion && adduser -S voxion -G voxion

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/yoomates /app/yoomates

# Set ownership
RUN chown -R voxion:voxion /app

USER voxion

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/app/yoomates"]
