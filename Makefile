.PHONY: build test lint clean

build:
	go build -o bin/convoy ./cmd/convoy

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/