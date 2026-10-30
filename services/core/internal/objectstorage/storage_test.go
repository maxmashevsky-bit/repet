package objectstorage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesystem(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "objects")
	s, err := NewFilesystem(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	object, err := s.Put(t.Context(), strings.NewReader("private source"), 100)
	if err != nil {
		t.Fatal(err)
	}
	if object.Size != 14 || len(object.SHA256) != 64 || !objectKey.MatchString(object.Key) {
		t.Fatal(object)
	}
	f, err := s.Open(t.Context(), object.Key)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(f)
	f.Close()
	if err != nil || string(data) != "private source" {
		t.Fatal(string(data), err)
	}
	for _, key := range []string{"../secret", "/etc/passwd", "", "a/../../etc/passwd", strings.Repeat("a", 63)} {
		if _, err = s.Open(t.Context(), key); !errors.Is(err, ErrInvalidKey) {
			t.Fatal("unsafe key", key, err)
		}
		if err = s.Delete(t.Context(), key); !errors.Is(err, ErrInvalidKey) {
			t.Fatal("unsafe delete", key, err)
		}
	}
	if _, err = s.Put(t.Context(), bytes.NewReader(make([]byte, 101)), 100); !errors.Is(err, ErrTooLarge) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err = s.Put(ctx, strings.NewReader("cancelled"), 100); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatal("partial objects leaked", entries, err)
	}
	if err = s.Delete(t.Context(), object.Key); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Open(t.Context(), object.Key); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
}
func TestFilesystemRejectsPublicRoot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "public")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if s, err := NewFilesystem(dir); err == nil {
		s.Close()
		t.Fatal("accepted public root")
	}
}
