package gistfs

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"time"

	"github.com/google/go-github/v33/github"
)

// Ensure FS implements fs.FS interface.
var _ fs.FS = (*FS)(nil)

var ErrNotLoaded = errors.New("gist not loaded")

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
	content string
	buf     *bytes.Buffer
	modTime time.Time
}

func (g *FS) Open(name string) (fs.File, error) {
	if g.gist == nil {
		return nil, ErrNotLoaded
	}

	gistFile, ok := g.gist.Files[github.GistFilename(name)]
	if !ok {
		return nil, fs.ErrNotExist
	}

	// this should not happen, but as it comes from the API, we never know.
	if gistFile.Filename == nil || gistFile.Content == nil || g.gist.UpdatedAt == nil {
		return nil, fs.ErrNotExist
	}

	file := file{
		name:    *gistFile.Filename,
		content: *gistFile.Content,
		buf:     bytes.NewBufferString(*gistFile.Content),
		modTime: *g.gist.UpdatedAt,
	}

	return &file, nil
}

func (f *file) isClosed() bool {
	return f.buf == nil
}

func (f *file) Read(b []byte) (int, error) {
	if f == nil {
		return 0, fs.ErrInvalid
	}

	if f.isClosed() {
		return 0, fs.ErrClosed
	}

	return f.buf.Read(b)
}

func (f *file) Close() error {
	if f == nil {
		return fs.ErrInvalid
	}

	f.content = ""
	f.buf = nil

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
		size:    int64(len(f.content)),
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
