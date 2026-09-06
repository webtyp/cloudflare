# cloudflare
<img src="docs/img/badges.svg">

Go/WASM runtime for Cloudflare Workers — the JS↔Go bridge, edge router, and storage adapters that compile into every deployed Worker. Extracted from `webtyp/goflare`; `goflare` keeps the build/deploy CLI and imports this repo only for the runtime it ships.

## Install

```bash
go get webtyp.com/cloudflare/edge
go get webtyp.com/cloudflare/workers
go get webtyp.com/cloudflare/d1
go get webtyp.com/cloudflare/r2
go get webtyp.com/cloudflare/log
```

Requires Go 1.25+ and TinyGo for the `//go:build wasm` packages.

## Minimal `edge/main.go`

```go
//go:build wasm

package main

import (
	"webtyp.com/cloudflare/d1"
	"webtyp.com/cloudflare/edge"
	"webtyp.com/fmt"
	"webtyp.com/orm"
)

func main() {
	db, err := d1.NewEdge("DB")
	if err != nil {
		fmt.Println("d1:", err)
		return
	}
	// optional DDL sync: ddl.New(db.RawConn(), d1.DDLCompiler(db.RawConn())).Sync(&YourModel{})

	r := edge.NewRouter(edge.Config{
		// Authn establishes identity before the gate — see edge/edge.go
		// Authn: myAuthMiddleware,
		// Authorize: myAuthorizer,
	})
	r.Get("/api/ping", func(ctx edge.Context) {
		ctx.Write([]byte("pong"))
	}).Public()

	r.Get("/api/items/{id}", func(ctx edge.Context) {
		id := ctx.Param("id")
		ctx.Write([]byte("item " + id))
	}).Public()

	// register more routes...

	edge.Serve(r) // blocks forever, handles every request in this isolate
}
```

Built with `goflare` (or any bundler reading `webtyp.com/cloudflare/assets`):

```bash
go install webtyp.com/goflare/cmd/goflare@latest
goflare build   # → .build/edge.js + .build/edge.wasm
goflare deploy  # → Cloudflare Workers API
```

See `webtyp/goflare` for the CLI, and `docs/ARCHITECTURE.md` for the isolate lifecycle contract (`context.binding` vs shared globals, `runtime.ticks` ABI).

## Packages

| Import | Description |
|---|---|
| `webtyp.com/cloudflare/edge` | Router adapter (`router.Router` on `workers.Handle`). For mounting `/_routes`, see `webtyp/router`'s `docs/INTROSPECTION.md`. |
| `webtyp.com/cloudflare/workers` | JS↔Go bridge (`Request`/`Response`, `Handle`/`Ready`) |
| `webtyp.com/cloudflare/d1` | D1 adapter for `webtyp/orm` (`d1.NewEdge`, or `d1.NewEdgeSession` for read replicas via the Sessions API) — see [docs/D1.md](docs/D1.md) |
| `webtyp.com/cloudflare/r2` | R2 bucket (`r2.NewEdge`) |
| `webtyp.com/cloudflare/log` | Edge logging (`log.Reject`/`Fail`/`Panic`) |
| `webtyp.com/cloudflare/assets` | JS half of runtime (`WasmExecJS`, `RuntimeMJS`, `WorkerMJS`) for bundlers — `!wasm` |
| `webtyp.com/env` | Env access (`env.Get`/`Lookup` — `os`+`.env` vs `context.env` auto-tag) |

## Constraints

- `//go:build wasm` files never import stdlib (`fmt`, `strings`, `errors`). Use `webtyp.com/fmt`.
- The runtime lives alone — no `os/exec`, no `net/http` client. That stays in `goflare`.
- Isolate invariants documented in `AGENTS.md` — read before touching `workers/` or `assets/`.

## Testing

```bash
go install webtyp.com/devflow/cmd/gotest@latest
gotest
```

`gotest` runs `go vet`, `go test -race -cover`, and the WASM suite in a real browser (auto-detected via `//go:build wasm`, `wasmbrowsertest` under the hood). See `webtyp/devflow/docs/GOTEST.md`.

## License

See `LICENSE`.
