package objectstorage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"regexp"
)

const MaxObjectBytes int64 = 8 << 20

var ErrInvalidKey = errors.New("invalid object key")
var ErrTooLarge = errors.New("object exceeds size limit")
var objectKey = regexp.MustCompile(`^[a-f0-9]{64}$`)

type Object struct {
	Key    string
	Size   int64
	SHA256 string
}
type ObjectStorage interface {
	Put(context.Context, io.Reader, int64) (Object, error)
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}

type Filesystem struct{ root *os.Root }

func NewFilesystem(directory string) (*Filesystem, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("object storage root must be a private non-symlink directory (0700)")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	return &Filesystem{root: root}, nil
}
func (s *Filesystem) Close() error { return s.root.Close() }
func (s *Filesystem) Put(ctx context.Context, body io.Reader, limit int64) (Object, error) {
	var out Object
	if limit < 1 || limit > MaxObjectBytes {
		return out, ErrTooLarge
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return out, err
	}
	out.Key = hex.EncodeToString(key)
	f, err := s.root.OpenFile(out.Key, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return Object{}, err
	}
	committed := false
	defer func() {
		_ = f.Close()
		if !committed {
			_ = s.root.Remove(out.Key)
		}
	}()
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, hash), io.LimitReader(contextReader{ctx, body}, limit+1))
	if err != nil {
		return Object{}, err
	}
	if n > limit {
		return Object{}, ErrTooLarge
	}
	if err = ctx.Err(); err != nil {
		return Object{}, err
	}
	if err = f.Sync(); err != nil {
		return Object{}, err
	}
	if err = f.Close(); err != nil {
		return Object{}, err
	}
	out.Size = n
	out.SHA256 = hex.EncodeToString(hash.Sum(nil))
	committed = true
	return out, nil
}
func (s *Filesystem) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !objectKey.MatchString(key) {
		return nil, ErrInvalidKey
	}
	f, err := s.root.Open(key)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > MaxObjectBytes || info.Mode().Perm()&0077 != 0 {
		f.Close()
		return nil, errors.New("invalid private object")
	}
	return f, nil
}
func (s *Filesystem) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !objectKey.MatchString(key) {
		return ErrInvalidKey
	}
	return s.root.Remove(key)
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
