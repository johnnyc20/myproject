.PHONY: build run test lint vet clean

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

vet:
	go vet ./...

lint: vet

clean:
	rm -rf bin data.db
