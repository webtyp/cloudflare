---
PLAN: "feat!: edge Context.RawQuery() and the router v0.5.0 contract"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 2721403967464863423
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work. Commit after every stage.
> Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.
> **Publishing:** you publish only by OPENING THE PULL REQUEST (a `git push` from your sandbox does not reach GitHub). Open it after Stage 2 with whatever compiles and keep committing to it.
> **Never ask a yes/no question about progress**; ask only when a signature in this plan is wrong, and name the symbol.

# Plan: `edge` follows the router Context contract

## Why (read once)

`webtyp.com/router` v0.5.0 changed the `router.Context` contract: `Path()` is the path only and the new `RawQuery() string` is the query (`"a=1&b=two"`, `""` when none), read with `router.QueryReader(ctx.RawQuery())`. This repo implements `Context`, so it stops compiling until it does the same. The contract is held by `router/conformance` (`Run` for transports served by a `Router`); a transport that disagrees fails it.

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go get webtyp.com/router@v0.5.0
go mod tidy
```
No `replace` in `go.mod`. If a module cannot be fetched, stop and say so in the PR; never write a hash by hand.

## APIs used

| symbol | exact signature | tag | file |
|---|---|---|---|
| `router.Context.RawQuery` | `RawQuery() string` | router v0.5.0 | `router.go` |
| `router.PathOf` | `func PathOf(target string) string` | router v0.5.0 | `query_codec.go` |
| `router.QueryOf` | `func QueryOf(path string) string` | router v0.5.0 | `query_codec.go` |
| `router.DecodeArgs` | `func DecodeArgs(method, query string, body []byte, into model.Decodable, fromBody func(body []byte, into model.Decodable) error) error` | router v0.5.0 | `query_codec.go` |
If a signature in the published tag differs from this table, stop and name the symbol in the PR; never adapt it by guessing.

## Stage 1 — `edge/edge.go`

`Path()` already returns the pathname only (line 78): keep it. Add next to it:

```go
// RawQuery returns the query of the request without the "?", "" when there is none.
func (c *wasmContext) RawQuery() string { return router.QueryOf(c.req.URL) }
```

and in `Decode` (line 135) replace `router.QueryOf(c.req.URL)` by `c.RawQuery()`:

```go
return router.DecodeArgs(c.Method(), c.RawQuery(), c.Body(), into, func(body []byte, d model.Decodable) error {
```

(keep the rest of the call as it is). The file is `//go:build wasm`: its tests run in the wasm destination of `gotest`.

## Stage 2 — tests and docs

| Test | setup → call → expected |
|---|---|
| `TestEdgeContext_RawQuery` (`tests/edge_params_test.go`, the file that already builds a `wasmContext`) | a request for `https://example.com/api/q?a=1&b=two`: `Path()` is `/api/q`, `RawQuery()` is `a=1&b=two`; for `https://example.com/api/q`, `RawQuery()` is `""` |
| `TestEdgeConformance` (exists) | passes with the five new clauses of router v0.5.0 (no edit) |

Add each new test's row to `docs/TESTS.md`. If `docs/ARCHITECTURE.md` or the README describe the edge `Context`, say that `Path()` is the pathname only and `RawQuery()` the query; otherwise add one sentence. Do not touch `tests/d1docker` (needs Docker).

## Delivery check

`gotest` passes (it builds, vets, races and runs every destination); the pull request is open. A tree that does not compile or whose tests did not run is a failed delivery.

## Executor notes
- Added `RawQuery()` to `wasmContext` in `edge/edge.go` and mock contexts (`testCtx`, `conformanceCtx`, `fakeCtx`) across tests.
- Modified `Path()` implementations in test mocks to correctly strip queries using `router.PathOf()`.
- Created `docs/TESTS.md` with the full test index as required by `gotest`.
- Updated dependencies (`webtyp.com/router` to `v0.5.0` and `webtyp.com/crud` to `v0.0.15` which implements `RawQuery()`).
