---
name: add-resource
description: Scaffolds a new CRUD resource in myproject (table + store methods + API handlers + routes). Use when asked to add a new resource, entity, or endpoint group to this Go API server.
---

# Add a Resource

This project's dependency direction is `cmd/server` → `internal/api` → `internal/store`.
A new resource touches only `internal/store` and `internal/api` — nothing in
`cmd/server` or `internal/config` needs to change. Follow the existing `items`
resource in both files as the template.

## 1. `internal/store/store.go`

- Add the table to `migrate()` as `CREATE TABLE IF NOT EXISTS ...` (no separate
  migration tool/directory exists — this is the only place schema is defined).
- Add a struct with `json` tags for the row shape.
- Add methods on `*Store` following the `Item` naming convention: `List<Plural>`,
  `Get<Singular>(id int64)`, `Create<Singular>(...)`, `Delete<Singular>(id int64)`,
  and `Update<Singular>(...)` if the resource is mutable.
- Return `sql.ErrNoRows` naturally from `Get`/`Update`/`Delete` (via `QueryRow.Scan`
  or checking `RowsAffected`) — don't swallow it, the API layer maps it to 404.
- No SQL in `internal/api` — all queries live here.

## 2. `internal/api/api.go`

- Add `mux.HandleFunc` lines in `Routes()` using Go 1.22+ method+path patterns,
  e.g. `"GET /widgets"`, `"POST /widgets"`, `"GET /widgets/{id}"`.
- Add one handler method per route, mirroring `handleListItems` / `handleCreateItem`
  / `handleGetItem` / `handleDeleteItem`:
  - Decode/validate the JSON body for writes; `writeError(w, http.StatusBadRequest, ...)`
    on bad input.
  - Parse `r.PathValue("id")` with `strconv.ParseInt` for path params.
  - Translate store errors to status codes (`sql.ErrNoRows` → 404, everything
    else → 500) — don't touch `database/sql` directly here.
  - Respond via the existing `writeJSON` / `writeError` helpers.

## 3. Tests

Add a test in `internal/api/api_test.go` using `store.Open(":memory:")` and
driving `API.Routes()` via `httptest`, matching the existing `TestCreateAndGetItem`
style — no real network or file I/O.

## 4. Verify

```sh
make vet
make test
go test ./internal/api -run TestYourNewTest -v
```
