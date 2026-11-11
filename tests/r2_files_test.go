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

func TestR2Files_RemoveMissingIsNotExist(t *testing.T) {
	bucket := &memBucket{}
	store := r2.NewFiles(bucket)

	if err := store.RemoveFile("nope"); err != files.ErrNotExist {
		t.Errorf("RemoveFile(missing) = %v; want %v", err, files.ErrNotExist)
	}

	if err := store.WriteFile("a", []byte("hello")); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := store.RemoveFile("a"); err != nil {
		t.Errorf("RemoveFile(existing) failed: %v", err)
	}

	if _, err := store.ReadFile("a"); err != files.ErrNotExist {
		t.Errorf("ReadFile(removed) error = %v; want %v", err, files.ErrNotExist)
	}
}
