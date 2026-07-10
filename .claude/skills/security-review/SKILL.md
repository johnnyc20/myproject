---
name: security-review
description: Reviews pending changes in this Go API server for security issues specific to this stack (net/http + database/sql + modernc.org/sqlite). Use when asked to security-review a diff, PR, or new resource in this repo, or before merging changes that touch internal/api or internal/store.
---

# Security Review (myproject)

Review the diff (or working tree, if no diff is specified) against the
checklist below. This repo has no auth layer and is a small internal-style
API, so focus on the concrete classes of bug that actually occur in this
codebase's patterns — don't flag generic advice that doesn't apply here.

## 1. SQL injection — `internal/store`

- Every query must use `?` placeholders with values passed as `Exec`/`Query`/
  `QueryRow` args — never `fmt.Sprintf`/string concatenation into SQL text,
  including for table/column names, `ORDER BY`, or `LIMIT`.
- Check `migrate()` too: schema changes are plain `Exec` strings but must
  never interpolate anything caller-controlled.

## 2. Error message leakage — `internal/api`

- `writeError` calls `err.Error()` directly and serializes it to the client.
  Any store error (including raw SQL driver errors) reaches the HTTP
  response verbatim. When reviewing a new handler, check whether the error
  passed to `writeError` could ever be a low-level `database/sql` /
  `modernc.org/sqlite` error rather than a sentinel like `sql.ErrNoRows` —
  if so, flag it; the fix is a generic message with the real error only in
  `log.Printf`.

## 3. Input validation — request decoding

- `json.NewDecoder(r.Body).Decode(&body)` has no `MaxBytesReader` — a large
  request body is read into memory unbounded. Flag new handlers that decode
  a body without a size limit, and flag if this is being introduced/removed
  inconsistently across handlers.
- Required-field checks (`body.Name == ""`, `body.Price == 0`) must exist for
  every new field on create/update handlers, matching the existing
  `handleCreateItem`/`handleCreateWidget` pattern. Note zero-value checks
  (e.g. `Price == 0`) can't distinguish "omitted" from "legitimately zero" —
  flag this if a field's valid range includes zero or negative values matter
  (e.g. a resource where 0 or negative price should be rejected but isn't).
- `strconv.ParseInt(r.PathValue("id"), 10, 64)` is the only path-param
  pattern in use — any new handler parsing a path value must use it (or
  equivalent bounds-checked parsing), not manual string handling.

## 4. AuthN/AuthZ

- There is currently no authentication or authorization middleware anywhere
  in `Routes()`. This is expected for this project's current scope — don't
  flag its *absence* as a finding unless the diff adds something that assumes
  auth exists (e.g. a handler that trusts a header/claim without
  establishing where it's verified).

## 5. Resource exhaustion

- No rate limiting or request size limits exist at the `http.ListenAndServe`
  level (`cmd/server/main.go`) or in `Routes()`. Flag any new endpoint that's
  meaningfully more expensive than existing ones (e.g. unbounded `List*`
  queries, N+1 patterns, or anything that reads an entire table without
  pagination) since there's no other backstop.

## 6. Dependency surface

- `internal/store` only imports `database/sql` and `modernc.org/sqlite`.
  Flag any new third-party dependency added to touch stored data or parse
  untrusted input — check `go.mod`/`go.sum` diff for unexpected additions.

## Verify

```sh
make vet
make test
```

Report findings as: file:line, the concrete failure scenario (not just "best
practice"), and severity. Don't report a finding if the pattern is identical
to existing, unchanged code elsewhere in the same file — that's pre-existing
risk to note separately, not a defect in the diff being reviewed.
