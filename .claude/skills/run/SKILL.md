---
name: run
description: Launches this project's Go API server and verifies it's serving. Use when asked to run, start, or check that the server works.
---

# Run myproject

## Start the server

```sh
make run     # go run ./cmd/server, listens on :8080 by default
```

Override config via env vars if needed (see `internal/config/config.go`):
- `ADDR` — listen address (default `:8080`)
- `DB_PATH` — SQLite file path (default `data.db`)

Run in the background (e.g. `run_in_background` or `&`) since it blocks. Give it
a second to bind, then hit an endpoint to confirm it's alive, e.g.:

```sh
curl -s -X POST localhost:8080/items -d '{"name":"test"}'
curl -s localhost:8080/items
```

Check routes in `internal/api` (`API.Routes()`) for the current set of endpoints
before guessing paths.

## Build instead of run

```sh
make build   # builds bin/server
./bin/server
```

## Stop it

Kill the backgrounded process when done poking at it — don't leave a stray
server holding `:8080` or the sqlite file locked for the next run.
