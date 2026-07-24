# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
make run     # go run ./cmd/server — starts the API on :8080
make build   # builds binary to bin/server
make test    # go test ./...
make vet     # go vet ./...

make mcp-fetch-run    # go run ./cmd/mcp-fetch — starts the internet-fetch MCP server (stdio)
make mcp-fetch-build  # builds binary to bin/mcp-fetch
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
- **`internal/accesslog`** — buffered HTTP access logging (`Logger.Middleware`,
  wrapped around `a.Routes()` in `cmd/server/main.go`). One log line per
  request is a genuine case for `bufio`-style buffering (many small,
  independent writes under load); this is a deliberate contrast with
  `cmd/mcp-fetch`'s stdio transport, where each JSON-RPC response is
  already a single atomic write and an explicit buffer would add nothing.
  Flushes every second on a background goroutine and on `Close()`; a
  crash between flushes loses at most ~1s of log lines, the accepted
  trade-off of buffering (see the package doc comment).
- **`cmd/mcp-fetch`** — a separate composition root for a local (stdio) MCP
  server, unrelated to the REST API above. It exposes one tool, `fetch_url`,
  backed by `internal/secfetch`.
- **`internal/secfetch`** — an SSRF-hardened outbound HTTP client for MCP
  tools that fetch arbitrary internet URLs. It enforces an explicit host
  allowlist (fail-closed: nothing is reachable until `MCP_FETCH_ALLOWED_HOSTS`
  is set), rejects everything but `https://`, and independently blocks
  connections to private/loopback/link-local/multicast/cloud-metadata IPs at
  dial time — checking the *resolved* address rather than the hostname is
  what stops DNS-rebinding bypassing the allowlist. See
  `internal/secfetch/config.go` for all `MCP_FETCH_*` env vars.

Adding a new resource means: add table + CRUD methods in `store`, add a handler
+ route in `api`, no changes needed elsewhere.

Tests use `store.Open(":memory:")` for an isolated in-memory SQLite DB per test
and drive handlers directly via `httptest` against `API.Routes()` — no real
network or file I/O involved.
