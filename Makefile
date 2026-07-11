BINARY_NAME=radar
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

.PHONY: build test lint run clean fmt vet

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/radar/

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

run:
	go run ./cmd/radar/

fmt:
	gofmt -s -w .

vet:
	go vet ./...

clean:
	rm -f $(BINARY_NAME)

install:
	go install $(LDFLAGS) ./cmd/radar/
