---
PLAN: "feat: r2.Files — the R2 bucket as a files.Store, proved by the files conformance suite"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 15743901672225066326
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work. Commit after every stage.
> Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.
> **Publishing:** you publish only by OPENING THE PULL REQUEST (a `git push` from your sandbox does not reach GitHub). Open it after Stage 2 with whatever compiles and keep committing to it.
> **Never ask a yes/no question about progress**; ask only when a signature in this plan is wrong, and name the symbol.
> **Before you submit:** `git status` must be clean and `git diff --stat main` must list the files of the stages. A pull request that carries only go.mod, go.sum or docs/PLAN.md is a failed delivery: `git add -A`, commit and submit again.

# Plan: `r2.Files`

## Why (read once)

File upload (design D46) hands each entity with a file a `files.Store` chosen by the app: `disk` on a server, `opfs` in a browser. On Cloudflare the destination is an R2 bucket, but `r2.Bucket` speaks `Put/Get/Delete/List`, not the `files.Store` contract (`ReadFile`, `WriteFile`, `AppendFile`, `RemoveFile`), so `crud` could not use it. This plan adds the adapter and proves it with the conformance suite that already proves `mem`, `disk` and `opfs`.

## Ecosystem search

Searched `files.Store`, `ReadWriter`, `conformance.Run` in `webtyp`: `files` owns the contract and `files/conformance` its executable proof (`conformance.Run(t, conformance.Factory{Name, New})`); `files/mem` is the reference. **Reuse** both; **new** here only the adapter, in the repo that owns R2 (`cloudflare/r2`).

## Design gate

1. **Prior art:** Go `io/fs` + `afero.Fs` adapters over object stores (S3 `afero` backends), Apache OpenDAL's `Operator` over S3/R2, Cloudflare's own R2 binding API (`put/get/delete/list`). All map a path-like key to an object; none promise append, which is why `AppendFile` here is read-modify-write and says so. https://developers.cloudflare.com/r2/api/workers/workers-api-reference/ (unverified in this session: from memory).
2. **Novice name:** `r2.NewFiles(bucket)` returns the bucket as a set of files; the bucket seam is `r2.ObjectBucket`, what a bucket does with objects.
3. **Ledger:** concepts +1 (`Files`), files an app touches to store files on R2 −N (one constructor), ways to do the same: 0.
4. **Where:** the R2 adapter lives in `cloudflare/r2` (the repo that owns R2); the contract stays in `files`.
5. **Deletes:** nothing.

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go get webtyp.com/files@latest
go mod tidy
```
No `replace` in `go.mod`. If a module cannot be fetched, stop and say so in the PR; never write a hash by hand.

## APIs used

| symbol | exact signature | tag | file |
|---|---|---|---|
| `files.Store` | `ReadFile(path string) ([]byte, error)`; `WriteFile(path string, data []byte) error`; `AppendFile(path string, data []byte) error`; `RemoveFile(path string) error` | files latest | `files.go` |
| `files.ErrNotExist` | `var ErrNotExist error` (returned as the value itself, never wrapped: callers compare with `==`) | files latest | `files.go` |
| `conformance.Run` | `func Run(t *testing.T, f Factory)` with `Factory{Name string; New func(t *testing.T) files.ReadWriter}` | files latest | `conformance/conformance.go:19` |
| `r2.Bucket` | `Put(key string, data []byte, contentType string) error`; `Get(key string) ([]byte, string, error)`; `Delete(key string) error`; `List(prefix string) ([]ObjectInfo, error)` | this repo | `r2/bucket.go` |
| `r2.ObjectInfo` | `struct{ Key string; Size int64; ContentType string }` | this repo | `r2/bucket.go:93` |
If a signature in the published tag differs from this table, stop and name the symbol in the PR; never adapt it by guessing.

## Stage 1 — a typed «not found» in `r2`

In `r2/errors.go` add (house pattern of the file):

```go
var ErrObjectNotFound = Err(errPrefix + "object not found")
```

In `r2/bucket.go`, `Get`: where it returns `fmt.Errf("r2: object not found: %s", key)` return `ErrObjectNotFound` instead (the value itself). `cloudflare/files` treats any error from `Get` as 404, so nothing else changes. Check: `grep -n "object not found" r2/*.go` → only `errors.go`.

## Stage 2 — `r2/files.go` (new, `//go:build wasm` like `bucket.go`)

```go
//go:build wasm

package r2

import "webtyp.com/files"

// ObjectBucket is what Files needs from a bucket. *Bucket satisfies it; tests give an in-memory one.
type ObjectBucket interface {
	Put(key string, data []byte, contentType string) error
	Get(key string) ([]byte, string, error)
	Delete(key string) error
	List(prefix string) ([]ObjectInfo, error)
}

// contentTypeBytes is stored for every object: the type of a file is kept in the record that owns it (deduced from its bytes), not here.
const contentTypeBytes = "application/octet-stream"

// Files is an R2 bucket used as a files.Store: the path is the object key.
//
//	store := r2.NewFiles(bucket) // pass it to the entity that owns files
type Files struct{ bucket ObjectBucket }

// NewFiles wraps a bucket as a files.Store.
func NewFiles(b ObjectBucket) *Files { return &Files{bucket: b} }

var _ files.Store = (*Files)(nil)

// ReadFile returns the object, or files.ErrNotExist itself when there is none.
func (f *Files) ReadFile(path string) ([]byte, error) {
	data, _, err := f.bucket.Get(path)
	if err == ErrObjectNotFound {
		return nil, files.ErrNotExist
	}
	return data, err
}

// WriteFile replaces the object.
func (f *Files) WriteFile(path string, data []byte) error {
	return f.bucket.Put(path, data, contentTypeBytes)
}

// AppendFile adds data to the end of the object by reading it and writing it back: R2 has no append, so two
// concurrent appends to the same path can lose one of them. Use it for logs written by one writer.
func (f *Files) AppendFile(path string, data []byte) error {
	current, err := f.ReadFile(path)
	if err != nil && err != files.ErrNotExist {
		return err
	}
	return f.WriteFile(path, append(current, data...))
}

// RemoveFile deletes the object, or returns files.ErrNotExist itself when there is none (R2 deletes silently, so the
// object is looked up first).
func (f *Files) RemoveFile(path string) error {
	found, err := f.bucket.List(path)
	if err != nil {
		return err
	}
	exists := false
	for i := 0; i < len(found); i++ {
		if found[i].Key == path {
			exists = true
		}
	}
	if !exists {
		return files.ErrNotExist
	}
	return f.bucket.Delete(path)
}
```

(`List(prefix)` also returns longer keys that start with `path`; the loop keeps only the exact key. No `map`, no stdlib: this file is wasm.)

## Stage 3 — tests and documentation

`tests/r2_files_test.go` (`//go:build wasm`, package `cloudflare_test`, next to `files_test.go`):

```go
//go:build wasm

package cloudflare_test

import (
	"testing"

	"webtyp.com/cloudflare/r2"
	"webtyp.com/files"
	"webtyp.com/files/conformance"
)

// memBucket is an in-memory r2.ObjectBucket: the edge of the test; Files is the real code.
type memBucket struct {
	keys  []string
	datas [][]byte
}

func (b *memBucket) index(key string) int {
	for i := 0; i < len(b.keys); i++ {
		if b.keys[i] == key {
			return i
		}
	}
	return -1
}

func (b *memBucket) Put(key string, data []byte, contentType string) error {
	copied := append([]byte(nil), data...)
	if i := b.index(key); i >= 0 {
		b.datas[i] = copied
		return nil
	}
	b.keys = append(b.keys, key)
	b.datas = append(b.datas, copied)
	return nil
}

func (b *memBucket) Get(key string) ([]byte, string, error) {
	i := b.index(key)
	if i < 0 {
		return nil, "", r2.ErrObjectNotFound
	}
	return append([]byte(nil), b.datas[i]...), "", nil
}

func (b *memBucket) Delete(key string) error {
	if i := b.index(key); i >= 0 {
		b.keys = append(b.keys[:i], b.keys[i+1:]...)
		b.datas = append(b.datas[:i], b.datas[i+1:]...)
	}
	return nil // like R2: deleting what is not there is not an error
}

func (b *memBucket) List(prefix string) ([]r2.ObjectInfo, error) {
	var out []r2.ObjectInfo
	for i := 0; i < len(b.keys); i++ {
		if len(b.keys[i]) >= len(prefix) && b.keys[i][:len(prefix)] == prefix {
			out = append(out, r2.ObjectInfo{Key: b.keys[i], Size: int64(len(b.datas[i]))})
		}
	}
	return out, nil
}

func TestR2Files_Conformance(t *testing.T) {
	conformance.Run(t, conformance.Factory{
		Name: "r2",
		New:  func(t *testing.T) files.ReadWriter { return r2.NewFiles(&memBucket{}) },
	})
}
```

| Test | setup → call → expected |
|---|---|
| `TestR2Files_Conformance` | the suite of `files/conformance` (read, write, append, remove, not-exist as the value itself) passes over `r2.NewFiles(&memBucket{})` |
| `TestR2Files_RemoveMissingIsNotExist` | `RemoveFile("nope")` → `files.ErrNotExist` compared with `==`; `RemoveFile("a")` after `WriteFile("a", …)` → nil and `ReadFile("a")` → `files.ErrNotExist` |

Each test gets its row in `docs/TESTS.md` (`| [Name](../tests/r2_files_test.go) | rule |`; the file may be incomplete: `gotest` prints one line `missing row for TestX (path)` per test without a row — add one row per line, format `| [TestX](../path) | X |`, sorted by name, until «tests index» is green). Documentation: one paragraph in `docs/ARCHITECTURE.md` and the README of the repo: «`r2.NewFiles(bucket)` is the `files.Store` of an app deployed on Workers; append is read-modify-write».

## Delivery check

`gotest` passes on every destination (the wasm one runs the new tests); `grep -n "object not found" r2/*.go` shows only `errors.go`; the pull request is open. A tree that does not compile or whose tests did not run is a failed delivery.
