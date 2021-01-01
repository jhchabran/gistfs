package gistfs

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"time"

	"github.com/google/go-github/v33/github"
)

// type File interface {
// 	Stat() (FileInfo, error)
// 	Read([]byte) (int, error)
// 	Close() error
// }
//
// type FileInfo interface {
// 	Name() string       // base name of the file
// 	Size() int64        // length in bytes for regular files; system-dependent for others
// 	Mode() FileMode     // file mode bits
// 	ModTime() time.Time // modification time
// 	IsDir() bool        // abbreviation for Mode().IsDir()
// 	Sys() interface{}   // underlying data source (can return nil)
// }
//
// type FileMode uint32

type GistFS struct {
	client *github.Client
	ID     string
}

func New(id string) *GistFS {
	return &GistFS{
		client: github.NewClient(nil),
		ID:     id,
	}
}

func NewWithClient(client *github.Client, id string) *GistFS {
	return &GistFS{
		client: client,
		ID:     id,
	}
}

type wrapper struct {
	name    string
	content string
	buf     *bytes.Buffer
}

func newWrapper(name string, content string) *wrapper {
	buf := bytes.NewBufferString(content)

	return &wrapper{
		name:    name,
		content: content,
		buf:     buf,
	}
}

func (w *wrapper) Read(b []byte) (int, error) { return w.buf.Read(b) }
func (w *wrapper) Stat() (fs.FileInfo, error) { return w, nil }
func (w *wrapper) Close() error               { return nil }

func (w *wrapper) Name() string       { return w.name }
func (w *wrapper) Size() int64        { return int64(len(w.content)) }
func (w *wrapper) Mode() fs.FileMode  { return fs.FileMode(0444) }
func (w *wrapper) ModTime() time.Time { return time.Now() } // TODO
func (w *wrapper) IsDir() bool        { return false }
func (w *wrapper) Sys() interface{}   { return w }

func (g *GistFS) Open(name string) (fs.File, error) {
	if name == "/" {
		return g.openRoot()
	}

	// TODO _ = resp
	gist, _, err := g.client.Gists.Get(context.Background(), g.ID)
	if err != nil {
		return nil, err
	}

	gistFile, ok := gist.Files[github.GistFilename(name)]
	if !ok {
		return nil, fmt.Errorf("not found") // TODO there may be some stuff for that
	}

	if gistFile.Content == nil {
		return nil, fmt.Errorf("null file") // TODO there may be some stuff for that
	}

	return newWrapper(name, *gistFile.Content), nil
}

func (g *GistFS) openRoot() (fs.File, error) {
	// gist, resp, err := g.client.Gists.Get(context.Background(), g.ID)
	// if err != nil {
	// 	return nil, err
	// }

	panic("not implemented")

	return nil, nil
}
