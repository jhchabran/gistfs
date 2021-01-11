package gistfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"sync"
	"time"

	"github.com/google/go-github/v33/github"
)

// Ensure FS implements fs.FS and fs.ReadFileFS interface.
var (
	_ fs.ReadFileFS = (*FS)(nil)

	_ fs.FileInfo = (*file)(nil)
)

// Ensure that *file is always a ReadDirFile.
// Non directories returns an error if ReadDir(int) is called on them.

var ErrNotLoaded = fmt.Errorf("gist not loaded: %w", fs.ErrInvalid)

type FS struct {
	id     string
	client *github.Client
	gist   *github.Gist
	mu     sync.RWMutex
}

func New(id string) *FS {
	return &FS{
		client: github.NewClient(nil),
		id:     id,
	}
}

func NewWithClient(client *github.Client, id string) *FS {
	return &FS{
		client: client,
		id:     id,
	}
}

// GetID returns the Github Gist ID that the filesystem was created with
func (fsys *FS) GetID() string {
	return fsys.id
}

// Load fetches the gist content from github, making the file system ready
// for use. If the underlying Github API call fails, it will return an error.
func (fsys *FS) Load(ctx context.Context) error {
	fsys.mu.Lock()
	defer fsys.mu.Unlock()

	gist, _, err := fsys.client.Gists.Get(ctx, fsys.id)
	if err != nil {
		return err
	}

	fsys.gist = gist

	return nil
}

// file represents a file stored in a Gist and implements fs.File methods.
// It is built out of a github.GistFile.
type file struct {
	gistFile *github.GistFile
	modtime  time.Time
	reader   io.Reader
	mu       sync.Mutex
}

func (fsys *FS) Open(name string) (fs.File, error) {
	fsys.mu.RLock()
	defer fsys.mu.RUnlock()

	if fsys.gist == nil {
		return nil, ErrNotLoaded
	}

	f, ok := fsys.gist.Files[github.GistFilename(name)]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return &file{
		gistFile: &f,
		reader:   bytes.NewReader([]byte(f.GetContent())),
		modtime:  fsys.gist.GetUpdatedAt(),
	}, nil
}

func (f *FS) ReadFile(name string) ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.gist == nil {
		return nil, ErrNotLoaded
	}

	gistFile, ok := f.gist.Files[github.GistFilename(name)]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return []byte(gistFile.GetContent()), nil
}

func (f *file) isClosed() bool {
	return f.reader == nil
}

func (f *file) Read(b []byte) (int, error) {
	if f == nil {
		return 0, fs.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.isClosed() {
		return 0, fs.ErrClosed
	}

	return f.reader.Read(b)
}

func (f *file) Close() error {
	if f == nil {
		return fs.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.gistFile = nil
	f.reader = nil

	return nil
}

func (f *file) Stat() (fs.FileInfo, error) {
	if f == nil {
		return nil, fs.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.isClosed() {
		return nil, fs.ErrClosed
	}

	return f, nil
}

func (f *file) ReadDir(n int) ([]fs.DirEntry, error) {
	return nil, fs.ErrInvalid
}

func (f *file) Name() string       { return f.gistFile.GetFilename() }
func (f *file) Size() int64        { return int64(f.gistFile.GetSize()) }
func (f *file) Mode() fs.FileMode  { return 0444 }
func (f *file) ModTime() time.Time { return f.modtime }
func (f *file) IsDir() bool        { return false }
func (f *file) Sys() interface{}   { return f }
