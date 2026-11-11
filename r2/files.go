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
