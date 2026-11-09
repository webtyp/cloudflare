//go:build wasm

package d1

import (
	"syscall/js"

	. "webtyp.com/fmt"
	"webtyp.com/jsvalue"
	"webtyp.com/storage"
)

type d1Rows struct {
	arr  js.Value
	cols []string
	cur  int
	len  int
	once bool
}

func (r *d1Rows) rowsLen() int {
	if !r.once {
		if r.arr.Truthy() {
			r.len = r.arr.Length()
		}
		r.once = true
	}
	return r.len
}

func (r *d1Rows) Columns() ([]string, error) { return r.cols, nil }

func (r *d1Rows) Next() bool {
	if r.cur < r.rowsLen() {
		r.cur++
		return true
	}
	return false
}

func (r *d1Rows) Scan(dest ...any) error {
	if r.cur == 0 || r.cur > r.rowsLen() {
		return Err(errPrefix + "invalid row cursor")
	}
	row := r.arr.Index(r.cur - 1)
	if len(dest) != len(r.cols) {
		return Err(errPrefix + "scan destination count mismatch")
	}
	for i, ptr := range dest {
		if err := scanCell(row.Index(i), ptr); err != nil {
			return err
		}
	}
	return nil
}

func (r *d1Rows) Close() error { return nil }
func (r *d1Rows) Err() error   { return nil }

// scanCell copies one D1 cell into dest through storage.ScanAny, the one definition of what a database value means in Go
// (NULL is the zero value, SQLite's 0/1 is a bool, ...).
func scanCell(cell js.Value, dest any) error {
	return storage.ScanAny(cellValue(cell), dest)
}

// cellValue is the Go value of one D1 cell. A BLOB arrives as a Uint8Array, an ArrayBuffer or (Miniflare) an array of byte
// numbers; every other cell is a plain JS value.
func cellValue(cell js.Value) any {
	if cell.Type() != js.TypeObject {
		return jsvalue.ToAny(cell)
	}
	if cell.InstanceOf(jsvalue.Uint8ArrayClass) {
		return copyBytes(cell)
	}
	if cell.InstanceOf(js.Global().Get("ArrayBuffer")) {
		return copyBytes(jsvalue.Uint8ArrayClass.New(cell))
	}
	if cell.InstanceOf(js.Global().Get("Array")) {
		out := make([]byte, cell.Length())
		for i := range out {
			out[i] = byte(cell.Index(i).Int())
		}
		return out
	}
	return jsvalue.ToAny(cell)
}

func copyBytes(u8 js.Value) []byte {
	b := make([]byte, u8.Length())
	js.CopyBytesToGo(b, u8)
	return b
}

type errScanner struct{ err error }

func (e *errScanner) Scan(...any) error { return e.err }

type rowScanner struct{ row js.Value }

func (s *rowScanner) Scan(dest ...any) error {
	if s.row.IsUndefined() || s.row.IsNull() {
		return Err(errPrefix + "no results to scan")
	}
	if len(dest) > s.row.Length() {
		return Err(errPrefix + "scan destination count mismatch")
	}
	for i, ptr := range dest {
		if err := scanCell(s.row.Index(i), ptr); err != nil {
			return err
		}
	}
	return nil
}

// compile-time check
var _ storage.Rows = (*d1Rows)(nil)
