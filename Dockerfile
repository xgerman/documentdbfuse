# Build stage
FROM golang:1.23-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/documentdbfuse ./cmd/documentdbfuse

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends fuse3 ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /bin/documentdbfuse /usr/local/bin/documentdbfuse

RUN mkdir -p /mnt/db

ENTRYPOINT ["documentdbfuse"]
