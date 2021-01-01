package gistfs

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"time"

	"github.com/google/go-github/v33/github"
)

// Ensure FS implements fs.FS interface.
var _ fs.FS = (*FS)(nil)

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

func (g *FS) Load(ctx context.Context) error {
	gist, _, err := g.client.Gists.Get(ctx, g.ID)
	if err != nil {
		return err
	}

	g.gist = gist
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
		return nil, fmt.Errorf("not loaded") // TODO there may be some stuff for that
	}
	gistFile, ok := g.gist.Files[github.GistFilename(name)]
	if !ok {
		return nil, fmt.Errorf("not found") // TODO there may be some stuff for that
	}

	// this should not happen, but as it comes from the API, we never know.
	if gistFile.Filename == nil || gistFile.Content == nil || g.gist.CreatedAt == nil {
		return nil, fmt.Errorf("not found") // TODO there may be some stuff for that
	}

	file := file{
		name:    *gistFile.Filename,
		content: *gistFile.Content,
		buf:     bytes.NewBufferString(*gistFile.Content),
		modTime: *g.gist.UpdatedAt,
	}

	return &file, nil
}

func (f *file) Read(b []byte) (int, error) {
	return f.buf.Read(b)
}

func (f *file) Close() error { return nil }

func (f *file) Stat() (fs.FileInfo, error) {
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
