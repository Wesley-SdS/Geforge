# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build both binaries
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" -o /bin/gateforge ./cmd/gateforge
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/upstream ./cmd/upstream

# Final stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/gateforge /usr/local/bin/gateforge
COPY --from=builder /bin/upstream /usr/local/bin/upstream
COPY --from=builder /app/configs/gateway.yaml /etc/gateforge/gateway.yaml

EXPOSE 8080

ENTRYPOINT ["gateforge"]
CMD ["--config", "/etc/gateforge/gateway.yaml"]
