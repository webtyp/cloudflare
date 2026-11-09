---
PLAN: "feat: the edge decodes a GET from its query (router.DecodeArgs); the D1 adapter reports rows affected and a duplicate key (storage.Result, storage.Duplicate)"
EXECUTOR: local
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.
> This plan runs **locally** (the owner decided it); publish with `gopush`, never `codejob`.
> Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.

## Goal

Two changes, three stages, one delivery: (stages 1–2) the Worker decodes a `GET` from its query; (stage 3) the D1 adapter follows the new `storage` executor contract.

Operations become REST: a read is `GET /api/<module>/<entity>?search=…`. The Worker (`edge/edge.go`, `//go:build wasm`) must decode the arguments of a `GET` from its query like `httpd` does.
Today `wasmContext.Decode` always reads the body (`edge.go:131`) and the context never keeps the query: the request path is taken from the URL's `pathname` only (`edge.go:555`).

## Ecosystem search

Searched `Decode(`, `pathname`, `DecodeArgs` over `webtyp/*`: `router.DecodeArgs` (router plan) holds the rule "a GET reads its query, every other method its body" and `server/httpd` calls it (server plan). **Reuse** it here; no second copy of the rule.

## Design gate

1. **Prior art**: none new; the rule is the one `httpd` follows.
2. **Novice-name test**: no new name.
3. **Ledger**: concepts 0; copies of the "GET reads its query" rule 1 → 1 (shared in `router`).
4. **Where**: the Worker adapter builds the context; the rule is in `router`.
5. **Deletes**: nothing.

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
```

Dependencies (all published, no `replace`): `go get webtyp.com/router@v0.4.21 webtyp.com/model@v0.2.18 webtyp.com/storage@v0.2.1 webtyp.com/ddl@v0.0.49 webtyp.com/orm@v0.12.30 webtyp.com/sqlt@v0.0.17 && go mod tidy`.

## APIs used

| symbol | exact signature | file |
|---|---|---|
| `router.DecodeArgs` | `func DecodeArgs(method, query string, body []byte, into model.Decodable, fromBody func(body []byte, into model.Decodable) error) error` | `router/query_codec.go` (router plan) |
| `router.QueryOf` | `func QueryOf(path string) string` (the query of a path or of a full URL, without `?` or fragment, `""` when none) | `router/query_codec.go` |
| `wasmContext` | fields `req *workers.Request`, `res`, `path string`; `Method() string`, `Body() []byte` | `edge/edge.go:75-77` |
| `workers.Request.URL` | `URL string` — the full request URL, query included | `workers/request.go:14` |
| `json.Decode` | `func Decode(input any, into model.Decodable) error` (`[]byte` or `string`) — read `json/decode.go` for the exact form | `json/` |

If a signature differs from this table, stop and name the symbol in the PR; never adapt it by guessing.

## Stage 1 — the code (`edge/edge.go`)

The edge cannot be unit-tested without a JS runtime, and the rule itself is tested in `router` (`TestDecodeArgs_*`); this stage is glue, proved by stage 2's compile test.

No new field and no JS `URL` call: the context already holds the full URL (`c.req.URL`) and `router.QueryOf` extracts its query; the construction at `edge.go:554-557` stays as it is. Replace `Decode` (`edge.go:131`):

```go
// Decode reads the arguments of the request: a GET carries them in its query, every other method in its body.
func (c *wasmContext) Decode(into model.Decodable) error {
	return router.DecodeArgs(c.Method(), router.QueryOf(c.req.URL), c.Body(), into, func(body []byte, d model.Decodable) error {
		return json.Decode(body, d)
	})
}
```

`Path()` is **not** changed: `Dispatch` matches routes on it (`edge.go:581`) and must keep receiving the bare path. Note this in the doc comment of `Path()`:
"// Path returns the pathname only (no query): the router matches on it. The query is read by Decode."

## Stage 2 — compile proof and documentation

- `GOOS=js GOARCH=wasm go vet ./...` (or `gotest`, which builds both targets) must pass: the `wasm` file compiles with the new call.
- `docs/ARCHITECTURE.md` (four sections): in the context section state that `Decode` follows `router.DecodeArgs` (GET → query, else body) and that `Path()` excludes the query on the edge.
- `README.md` unchanged unless it documents `Decode`; keep ≤ 150 lines.
- Do not link `docs/PLAN.md` from permanent docs.

Commit: `feat: the edge decodes a GET from its query`.

## Stage 3 — D1 `Exec` reports rows affected and a duplicate key (decision D48)

### Goal

`webtyp/storage` v0.2.0 (now v0.2.1) changed the executor contract (decision D48 of the architecture master, so that `crud` can do a conditional update and report a duplicate key):

| Before | After |
|---|---|
| `Exec(query string, args ...any) error` | `Exec(query string, args ...any) (storage.Result, error)` |
| a unique violation = the driver's own error | a unique violation = `storage.Duplicate(driverErr)`, tested with `storage.IsDuplicate(err)` |

`storage.Result` has one method, `RowsAffected() int64`; `storage.Affected(n)` is the value every implementation returns. Nothing else changes in this repo: no new feature, no rename.
Wave 3 of the migration; the repos it depends on are published (ddl, orm, sqlt).

### Ecosystem search

Searched `Exec(` over `webtyp/*` and `veltylabs/*`: the implementers and callers of the old signature are the ones listed here; the compiler reports any site missed. No new library.

### Design gate

1. **Prior art**: `database/sql.Result` (`RowsAffected`), `sql.DB.Exec`.
2. **Novice-name test**: `result.RowsAffected()` = how many rows changed; `storage.IsDuplicate(err)` = the key already exists.
3. **Ledger**: concepts 0 here (the contract is storage's); ways to learn that a write changed nothing: 0 → 1.
4. **Where**: the contract is `webtyp/storage`; this repo adapts to it.
5. **Deletes**: the old single-value `Exec` shape.

### Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go get webtyp.com/storage@v0.2.1 webtyp.com/ddl@v0.0.49 webtyp.com/orm@v0.12.30 webtyp.com/sqlt@v0.0.17
go mod tidy
```

### What to change

`cloudflare/d1/adapter.go`: `adapter.Exec` returns `(storage.Result, error)`. The `run` promise's result carries `meta.changes`: read it (the JS number) and return `storage.Affected(changes)`.
D1 is SQLite: an error whose message contains `UNIQUE constraint failed` is returned as `storage.Duplicate(err)` (use `fmt.Contains` from `webtyp.com/fmt`).
Name the SQLite message once as a constant (`const uniqueViolation = "UNIQUE constraint failed"`), never a literal in the logic.
`d1/session_test.go` (package `d1`, `wasm`) calls `adapterObj.Exec(...)` at lines 82 and 209: adapt them to `_, err =`.
Add to that file two tests over a fake `run` promise (the file's `fakePromise`/`setupMockDB` helpers): `TestAdapterExec_ReportsRowsChanged` (the resolved value carries `meta.changes = 3` → `RowsAffected() == 3`) and `TestAdapterExec_UniqueViolationIsDuplicate` (a rejected promise whose message contains `UNIQUE constraint failed: t.id` → `storage.IsDuplicate(err)`; any other message is returned as it came).

Rules for every call site (the compiler lists them: follow its errors, then `grep -rn "\.Exec(" --include=*.go .`):

| Old | New |
|---|---|
| `err := x.Exec(…)` | `_, err := x.Exec(…)` |
| `if err := x.Exec(…); err != nil {` | `if _, err := x.Exec(…); err != nil {` |
| `return x.Exec(…)` in a function that returns `error` | `_, err := x.Exec(…)` then `return err` |
| a type with `Exec(query string, args ...any) error` | `Exec(query string, args ...any) (storage.Result, error)`; return `storage.Affected(n)` with the real count, or `storage.Affected(0)` when it executes nothing (a test double) |

Never ignore the count where the caller can use it, and never invent a count: a driver that cannot know it fails the `storage/conformance` suite and the delivery.

Commit: `feat: D1 Exec reports rows affected and a duplicate key`.

## Stage 4 — conformance against a real D1 (Docker)

The browser tests fake `context.env`, so they cannot tell how D1 really answers. `tests/d1docker/` (own `go.mod`, so `gotest` needs no Docker) holds `d1_conformance_test.go` (`//go:build wasm`), which runs `storage/conformance` and `ddl/conformance` over `d1.NewEdge("DB")`, a `Dockerfile` (node 22 glibc + a pinned Miniflare) and `run.cjs` (glue that puts Miniflare's D1 binding in `context.env` and runs the WASM test binary). `run.sh` builds and runs it. Fixing what it reports is part of the stage.

## Delivery check

1. `gotest` (builds, vets, runs both targets; a PR that does not compile, or whose tests could not run, is a failed delivery).
2. `grep -n "json.Decode(c.Body()" edge/edge.go` → empty.
3. `grep -rnE "\) Exec\(([a-z]+) string, ([a-z]+) \.\.\.any\) error" --include=*.go .` → empty (no implementer with the old shape).
4. `git diff --stat main` lists only `edge/edge.go`, `d1/adapter.go`, `d1/session_test.go`, `go.mod`, `go.sum`, `docs/ARCHITECTURE.md`, `docs/D1.md`.

| Stage | Files | Commit |
|---|---|---|
| 1 | `edge/edge.go` | `feat: the edge decodes a GET from its query` |
| 2 | `docs/ARCHITECTURE.md` | `docs: Decode source on the edge` |
| 3 | `d1/adapter.go`, `d1/errors.go`, `d1/session_test.go`, `docs/D1.md` (the `Exec` contract), `go.mod`, `go.sum` | `feat: D1 Exec reports rows affected and a duplicate key` |
| 4 | `tests/d1docker/*`, `d1/rows.go`, `d1/adapter.go`, `docs/D1.md`, `docs/ARCHITECTURE.md` | `test: D1 conformance against a real Miniflare D1 in Docker` |

## Executed — what the real D1 revealed

`tests/d1docker/run.sh` found four defects of the adapter that the fakes could not show; all fixed in `d1/rows.go` and `d1/adapter.go`: a `bool` column panicked on read (D1 returns `0/1`), a missing row answered `orm.ErrNotFound` instead of `storage.ErrNoRows`, NULL scanned as the text `<null>`, and a `BLOB` read back empty (Miniflare returns an array of bytes).
