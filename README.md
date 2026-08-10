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
- `GET /widgets` — list widgets
- `POST /widgets` — create widget, body `{"name": "...", "price": 1999}`
- `GET /widgets/{id}` — get widget
- `PUT /widgets/{id}` — update widget, body `{"name": "...", "price": 1999}`
- `DELETE /widgets/{id}` — delete widget
- `GET /notes` — list notes
- `POST /notes` — create note, body `{"body": "..."}`
- `GET /notes/{id}` — get note
- `GET /memories` — list memories, optionally filtered with `?type=`
  (`user`, `feedback`, `project`, or `reference`)
- `POST /memories` — create memory, body `{"name": "...", "type": "...", "description": "...", "content": "..."}`
- `GET /memories/{id}` — get memory
- `DELETE /memories/{id}` — delete memory
- `GET /memories/search?q=...` — full-text search over memories
- `GET /memories/{id}/relationships` — list graph edges touching this memory
  (either direction)
- `POST /memories/{id}/relationships` — create a directed edge, body
  `{"to_memory_id": 2, "type": "..."}` (`references`, `contradicts`,
  `related_to`, or `supersedes`)
- `DELETE /memories/{id}/relationships/{relId}` — delete an edge

Memories form a small knowledge graph this way — nodes are memories, edges
are typed relationships between them. Deleting a memory cascades to delete
its relationships (SQLite foreign keys are enforced via `PRAGMA foreign_keys
= ON`).

## MCP internet-fetch server

`cmd/mcp-fetch` is a separate, local (stdio) MCP server that exposes one
tool, `fetch_url`, letting an LLM retrieve HTTPS content through an
SSRF-hardened client (`internal/secfetch`). It has no default allowlist —
every fetch is rejected until you configure one.

```sh
make mcp-fetch-run    # go run ./cmd/mcp-fetch
make mcp-fetch-build  # build binary to bin/mcp-fetch
```

Environment variables:

- `MCP_FETCH_ALLOWED_HOSTS` — comma-separated hostnames the tool may fetch,
  e.g. `docs.example.com,*.example.com`. Required; empty means nothing is
  reachable.
- `MCP_FETCH_DENIED_HOSTS` — comma-separated hostnames to explicitly block,
  even if they'd otherwise match the allowlist.
- `MCP_FETCH_TIMEOUT` — per-request timeout (default `10s`)
- `MCP_FETCH_MAX_REDIRECTS` — max redirects to follow (default `3`)
- `MCP_FETCH_MAX_BODY_BYTES` — response body cap in bytes (default `2097152`, 2 MiB)
- `MCP_FETCH_USER_AGENT` — outbound `User-Agent` header (default `myproject-mcp-fetch/1.0`)

Regardless of the allowlist, requests are always rejected if they resolve to
a private, loopback, link-local, multicast, or cloud-metadata address, or if
the URL scheme isn't `https`.

To use it from a Claude Code / MCP-compatible client, point the client at
`go run ./cmd/mcp-fetch` (or the built `bin/mcp-fetch` binary) over stdio,
with the environment variables above set in its MCP server config.

## MCP DeepSeek server

`cmd/mcp-deepseek` is another separate, local (stdio) MCP server that exposes
one tool, `deepseek_chat`, letting an LLM delegate a prompt to DeepSeek's
chat-completions API (`internal/deepseek`). Unlike `mcp-fetch`, it always
talks to a single fixed, operator-configured host rather than an arbitrary
caller-supplied URL, so there's no allowlist to configure — just an API key.

```sh
make mcp-deepseek-run    # go run ./cmd/mcp-deepseek
make mcp-deepseek-build  # build binary to bin/mcp-deepseek
```

Environment variables:

- `DEEPSEEK_API_KEY` — your DeepSeek API key from
  [platform.deepseek.com](https://platform.deepseek.com). Required; every
  `deepseek_chat` call is rejected with a clear error until it's set (the
  server itself still starts, matching `mcp-fetch`'s fail-closed-at-call-time
  behavior rather than crashing at startup).
- `DEEPSEEK_BASE_URL` — API base URL (default `https://api.deepseek.com`).
  Must be `https://`.
- `DEEPSEEK_MODEL` — default model ID (default `deepseek-chat`; override to
  e.g. `deepseek-reasoner`, or whatever your account's current model IDs are
  if DeepSeek has renamed them since this was written).
- `DEEPSEEK_TIMEOUT` — per-request timeout (default `60s`)
- `DEEPSEEK_MAX_BODY_BYTES` — response body cap in bytes (default `4194304`, 4 MiB)

To use it from a Claude Code / MCP-compatible client, point the client at
`go run ./cmd/mcp-deepseek` (or the built `bin/mcp-deepseek` binary) over
stdio, with `DEEPSEEK_API_KEY` set in its MCP server config.

**Not verified end-to-end**: `api.deepseek.com` is blocked by this
development environment's own network policy (proxy-level `403` on
`CONNECT`), so this was built and unit-tested (`internal/deepseek/client_test.go`,
against a local `httptest` server standing in for the real API) but never
exercised against the actual DeepSeek API. The request/response shapes match
DeepSeek's published API docs as of when this was written — verify with a
real API key before relying on it, and open an issue if DeepSeek has changed
their contract since.
