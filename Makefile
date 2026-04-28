BIN     := aiw
VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test lint fmt fmt-check tidy snapshot clean

build:
	go build $(LDFLAGS) -o $(BIN) .

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

fmt-check:
	@gofmt -l . | grep -q . && echo "Files need formatting" && exit 1 || echo "All files properly formatted"

tidy:
	go mod tidy && go mod verify

snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -f $(BIN)
	rm -rf dist/
