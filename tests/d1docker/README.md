# D1 conformance against a real local D1

Runs the conformance suites of `webtyp/storage` and `webtyp/ddl` over `d1.NewEdge("DB")`, with a **real** D1 database (Miniflare/workerd) inside Docker.

## Run

```bash
tests/d1docker/run.sh          # all clauses, verbose
tests/d1docker/run.sh -test.run TestD1_DDLConformance
```

Needs Docker and Go. `run.sh` compiles the test for `GOOS=js GOARCH=wasm`, builds the image (`Dockerfile`: Node 22 on glibc + a pinned Miniflare) and runs it; `run.cjs` is only the glue that places the D1 binding in `context.env.DB`.

The `batch_insert_is_atomic` clause is skipped: D1 has no transaction opened from a Worker (see [../../docs/D1.md](../../docs/D1.md)).
