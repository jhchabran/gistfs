package gistfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"time"

	"github.com/google/go-github/v33/github"
)

// Ensure FS implements fs.FS and fs.ReadFileFS interface.
var _ fs.FS = (*FS)(nil)
var _ fs.ReadFileFS = (*FS)(nil)

var ErrNotLoaded = fmt.Errorf("gist not loaded: %w", fs.ErrInvalid)

type FS struct {
	ID     string
	client *github.Client
	gist   *github.Gist
}

func New(id string) *FS {
	return &FS{
		client: github.NewClient(nil),
		ID:     id,
	}
}

func NewWithClient(client *github.Client, id string) *FS {
	return &FS{
		client: client,
		ID:     id,
	}
}

// Load fetches the gist content from github, making the file system ready
// for use. If the underlying Github API call fails, it will return an error.
func (f *FS) Load(ctx context.Context) error {
	gist, _, err := f.client.Gists.Get(ctx, f.ID)
	if err != nil {
		return err
	}

	f.gist = gist
	return nil
}

// file represents a file stored in a Gist and implements fs.File methods.
// It is built out of a github.GistFile.
type file struct {
	name    string
	reader  io.Reader
	modTime time.Time
	size    int64
}

func (f *FS) Open(name string) (fs.File, error) {
	if f.gist == nil {
		return nil, ErrNotLoaded
	}

	gistFile, ok := f.gist.Files[github.GistFilename(name)]
	if !ok {
		return nil, fs.ErrNotExist
	}

	// This should not happen, but as it comes from the API, we never know.
	// Also, it's more accurate to test against the pointers from the response
	// than the accessors such as gistFile.GetContent() as an empty string
	// could be a valid value.
	if gistFile.Filename == nil || gistFile.Content == nil || f.gist.UpdatedAt == nil {
		return nil, fs.ErrNotExist
	}

	file := file{
		name:    *gistFile.Filename,
		reader:  bytes.NewReader([]byte(*gistFile.Content)),
		size:    int64(len(*gistFile.Content)),
		modTime: *f.gist.UpdatedAt,
	}

	return &file, nil
}

func (f *FS) ReadFile(name string) ([]byte, error) {
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

	if f.isClosed() {
		return 0, fs.ErrClosed
	}

	return f.reader.Read(b)
}

func (f *file) Close() error {
	if f == nil {
		return fs.ErrInvalid
	}

	f.reader = nil

	return nil
}

func (f *file) Stat() (fs.FileInfo, error) {
	if f == nil {
		return nil, fs.ErrInvalid
	}

	if f.isClosed() {
		return nil, fs.ErrClosed
	}

	info := fileInfo{
		name:    f.name,
		size:    f.size,
		mode:    fs.FileMode(0444),
		modTime: f.modTime,
		sys:     f,
	}

	return &info, nil
}

// fileInfo represents the file infos of a file stored in a gift.
type fileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	sys     *file
}

func (i *fileInfo) Name() string       { return i.name }
func (i *fileInfo) Size() int64        { return i.size }
func (i *fileInfo) Mode() fs.FileMode  { return i.mode }
func (i *fileInfo) ModTime() time.Time { return i.modTime }
func (i *fileInfo) IsDir() bool        { return false }
func (i *fileInfo) Sys() interface{}   { return i.sys }
