package gistfs

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/v33/github"
	"github.com/gregjones/httpcache"
)

var referenceGistID = "ded2f6727d98e6b0095e62a7813aa7cf"

// Avoid burning rate limit by using a caching http client.
func cachingClient() *github.Client {
	c := &http.Client{
		Transport: httpcache.NewMemoryCacheTransport(),
	}

	return github.NewClient(c)
}

func TestErrorNotLoaded(t *testing.T) {
	if !errors.Is(ErrNotLoaded, fs.ErrInvalid) {
		t.Fatal("Err not loaded is not a wrapped ErrInvalid")
	}
}

func TestNew(t *testing.T) {
	t.Run("New OK", func(t *testing.T) {
		gfs := New(referenceGistID)
		if got, want := gfs.ID, referenceGistID; got != want {
			t.Fatalf("New returned a FS with ID=%#v, want %#v", got, want)
		}
	})

	t.Run("NewWithClient OK", func(t *testing.T) {
		gfs := NewWithClient(cachingClient(), referenceGistID)
		if got, want := gfs.ID, referenceGistID; got != want {
			t.Fatalf("NewWithClient returned a FS with ID=%#v, want %#v", got, want)
		}
	})
}

func TestOpen(t *testing.T) {
	t.Run("Open OK", func(t *testing.T) {
		gfs := NewWithClient(cachingClient(), referenceGistID)
		gfs.Load(context.Background())

		tests := []struct {
			name string
			err  error
		}{
			{"test1.txt", nil},
			{"test2.txt", nil},
			{"non-existing-file.txt", fs.ErrNotExist},
		}

		for _, test := range tests {
			f, err := gfs.Open(test.name)

			if err != test.err {
				t.Fatalf("Opened %#v, got error %#v, want %#v", test.name, err, test.err)
			}

			if test.err == nil && f == nil {
				t.Fatalf("Opened %#v, got nil", test.name)
			}
		}
	})

	t.Run("Open NOK not loaded", func(t *testing.T) {
		gfs := NewWithClient(cachingClient(), referenceGistID)
		_, err := gfs.Open("test1.txt")

		if err == nil {
			t.Fatalf("Opened without loading, got no error, want %#v", ErrNotLoaded)
		} else if err != ErrNotLoaded {
			t.Fatalf("Opened without loading, got an error %#v, want %#v", err, ErrNotLoaded)
		}
	})
}

func TestRead(t *testing.T) {
	gfs := NewWithClient(cachingClient(), referenceGistID)
	gfs.Load(context.Background())

	t.Run("Read OK", func(t *testing.T) {
		f, err := gfs.Open("test1.txt")
		if err != nil {
			t.Fatalf("Opened file and got an error %#v, want no error", err)
		}

		b := make([]byte, len("foobar\n"))
		n, err := f.Read(b)

		if err != nil {
			t.Fatalf("Read and got an error %#v, want no error", err)
		}

		if got, want := n, len("foobar\n"); got != want {
			t.Fatalf("Read %d bytes in b, want %d bytes", got, want)
		}

		if got, want := string(b), "foobar\n"; got != want {
			t.Fatalf("Read %d bytes in b (%#v), want %#v", n, got, want)
		}
	})

	t.Run("Read NOK nil file", func(t *testing.T) {
		b := make([]byte, len("foobar\n"))
		var f *file = nil
		_, err := f.Read(b)

		if got, want := err, fs.ErrInvalid; got != want {
			t.Fatalf("Read on a nil file and got %#v, want %#v", got, want)
		}
	})

	t.Run("Read NOK closed file", func(t *testing.T) {
		f, err := gfs.Open("test1.txt")
		if err != nil {
			t.Fatalf("Opened file and got an error %#v, want no error", err)
		}

		b := make([]byte, len("foobar\n"))
		_ = f.Close()
		_, err = f.Read(b)

		if got, want := err, fs.ErrClosed; got != want {
			t.Fatalf("Read on a closed file and got %#v, want %#v", got, want)
		}
	})
}

func TestStat(t *testing.T) {
	gfs := NewWithClient(cachingClient(), referenceGistID)
	gfs.Load(context.Background())

	t.Run("Stat OK", func(t *testing.T) {
		approxModTime, _ := time.Parse("2000-12-31", "2020-01-02") // when the gist was last edited

		tests := []struct {
			filename      string
			err           error
			size          int64
			mode          fs.FileMode
			approxModTime time.Time
			isDir         bool
		}{
			{
				filename:      "test1.txt",
				err:           nil,
				size:          int64(len("foobar\nbarfoo")),
				mode:          fs.FileMode(0444),
				approxModTime: approxModTime,
				isDir:         false,
			},
			{
				filename:      "test2.txt",
				err:           nil,
				size:          int64(len("olala\n12345\nabcde")),
				mode:          fs.FileMode(0444),
				approxModTime: approxModTime,
				isDir:         false,
			},
		}

		for _, test := range tests {
			f, err := gfs.Open(test.filename)
			if err != nil {
				t.Fatalf("Opened file and got an error %#v, want no error", err)
			}

			stat, err := f.Stat()

			if err != test.err {
				t.Fatalf("Stat and got an error %#v, want %#v", err, test.err)
			}

			if test.err == nil {
				if got, want := stat.Name(), test.filename; got != want {
					t.Fatalf("got filename %#v, want %#v", got, want)
				}

				if got, want := stat.Size(), test.size; got != want {
					t.Fatalf("got size %#v, want %#v", got, want)
				}

				if got, want := stat.Mode(), test.mode; got != want {
					t.Fatalf("got mode %#v, want %#v", got, want)
				}

				if got, want := stat.ModTime(), test.approxModTime; got.After(approxModTime) &&
					got.Before(test.approxModTime.Add(24*time.Hour)) {
					t.Fatalf("got modTime %#v, want approx %#v", got, want)
				}

				if got, want := stat.IsDir(), test.isDir; got != want {
					t.Fatalf("got isDir %#v, want %#v", got, want)
				}

				if got, want := stat.Sys(), f; got != want {
					t.Fatalf("got Sys %#v, want %#v", got, want)
				}
			}
		}
	})

	t.Run("Stat NOK nil file", func(t *testing.T) {
		var f *file = nil
		_, err := f.Stat()

		if got, want := err, fs.ErrInvalid; got != want {
			t.Fatalf("Read on a nil file and got %#v, want %#v", got, want)
		}
	})

	t.Run("Stat NOK closed file", func(t *testing.T) {
		f, err := gfs.Open("test1.txt")
		if err != nil {
			t.Fatalf("Opened file and got an error %#v, want no error", err)
		}

		_ = f.Close()
		_, err = f.Stat()

		if got, want := err, fs.ErrClosed; got != want {
			t.Fatalf("Read on a closed file and got %#v, want %#v", got, want)
		}
	})
}
