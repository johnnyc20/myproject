# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
make run     # go run ./cmd/server — starts the API on :8080
make build   # builds binary to bin/server
make test    # go test ./...
make vet     # go vet ./...
```

Run a single test: `go test ./internal/api -run TestCreateAndGetItem -v`

The `go` toolchain is pinned via `go.mod` (`go 1.25.0`) and will auto-download
a matching toolchain (`GOTOOLCHAIN=auto`) if the system Go is older — no manual
version management needed.

## Configuration

Environment variables (see `internal/config/config.go`):
- `ADDR` — listen address (default `:8080`)
- `DB_PATH` — SQLite file path (default `data.db`)

## Architecture

Three-layer structure, one direction of dependency: `cmd/server` → `internal/api` → `internal/store`.

- **`cmd/server/main.go`** — composition root. Loads config, opens the store,
  builds the API, starts `http.ListenAndServe`. No logic lives here beyond wiring.
- **`internal/api`** — HTTP layer. `API.Routes()` builds an `http.ServeMux` using
  Go 1.22+ method+path patterns (e.g. `mux.HandleFunc("GET /items/{id}", ...)`,
  `r.PathValue("id")`). Handlers decode/encode JSON and translate store errors
  into HTTP status codes; they contain no SQL and don't touch `database/sql` directly.
- **`internal/store`** — persistence layer. Wraps `database/sql` with the
  `modernc.org/sqlite` driver (pure Go, no cgo — this is why the module requires
  a newer Go toolchain). `Store.migrate()` runs `CREATE TABLE IF NOT EXISTS` on
  `Open()`; there is no separate migration tool/directory. Add new tables/columns
  by extending `migrate()`.
- **`internal/config`** — env-var loading with defaults, nothing else.

Adding a new resource means: add table + CRUD methods in `store`, add a handler
+ route in `api`, no changes needed elsewhere.

Tests use `store.Open(":memory:")` for an isolated in-memory SQLite DB per test
and drive handlers directly via `httptest` against `API.Routes()` — no real
network or file I/O involved.
