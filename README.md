# myproject

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A Go HTTP API server backed by SQLite.

## Requirements

- Go 1.22+ (toolchain auto-manages to 1.25 via `go.mod`)

## Development

```sh
make run     # start the server on :8080
make build   # build binary to bin/server
make test    # run tests
make vet     # go vet
```

## Configuration

Environment variables:

- `ADDR` — listen address (default `:8080`)
- `DB_PATH` — SQLite file path (default `data.db`)

## API

- `GET /healthz` — health check
- `GET /items` — list items
- `POST /items` — create item, body `{"name": "..."}`
- `GET /items/{id}` — get item
- `DELETE /items/{id}` — delete item
