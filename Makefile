.PHONY: build test test-verbose lint run bench docker-up docker-down coverage-html clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags="-s -w -X main.Version=$(VERSION)"

build:
	@echo "Building gateforge..."
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/gateforge ./cmd/gateforge
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/upstream ./cmd/upstream
	@echo "Done. Binaries in bin/"

test:
	go test -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | grep total

test-verbose:
	go test -v -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

run: build
	./bin/gateforge --config configs/gateway.yaml

bench:
	go test -bench=. -benchmem ./...

docker-up:
	docker compose -f deployments/docker-compose.yml up --build

docker-down:
	docker compose -f deployments/docker-compose.yml down -v

coverage-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

clean:
	rm -rf bin/ coverage.out coverage.html
