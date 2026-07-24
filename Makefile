.PHONY: build run test lint vet clean mcp-fetch-build mcp-fetch-run mcp-deepseek-build mcp-deepseek-run

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

mcp-fetch-build:
	go build -o bin/mcp-fetch ./cmd/mcp-fetch

mcp-fetch-run:
	go run ./cmd/mcp-fetch

mcp-deepseek-build:
	go build -o bin/mcp-deepseek ./cmd/mcp-deepseek

mcp-deepseek-run:
	go run ./cmd/mcp-deepseek

test:
	go test ./...

vet:
	go vet ./...

lint: vet

clean:
	rm -rf bin data.db
