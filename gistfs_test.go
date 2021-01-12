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
var approxModTime, _ = time.Parse("2000-12-31", "2020-01-02") // when the gist was last edited

// Avoid burning rate limit by using a caching http client.
func cachingClient() *github.Client {
	c := &http.Client{
		Transport: httpcache.NewMemoryCacheTransport(),
	}

	return github.NewClient(c)
}

var cacheClient = cachingClient()

func TestErrorNotLoaded(t *testing.T) {
	if !errors.Is(ErrNotLoaded, fs.ErrInvalid) {
		t.Fatal("Err not loaded is not a wrapped ErrInvalid")
	}
}

func TestNew(t *testing.T) {
	t.Run("New OK", func(t *testing.T) {
		gfs := New(referenceGistID)
		if got, want := gfs.GetID(), referenceGistID; got != want {
			t.Fatalf("New returned a FS with ID=%#v, want %#v", got, want)
		}
	})

	t.Run("NewWithClient OK", func(t *testing.T) {
		gfs := NewWithClient(cacheClient, referenceGistID)
		if got, want := gfs.GetID(), referenceGistID; got != want {
			t.Fatalf("NewWithClient returned a FS with ID=%#v, want %#v", got, want)
		}
	})
}

func TestOpen(t *testing.T) {
	t.Run("Open OK", func(t *testing.T) {
		gfs := NewWithClient(cacheClient, referenceGistID)
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

			if test.err != nil && !errors.Is(err, test.err) {
				t.Fatalf("Opened %#v, got error %#v, want %#v", test.name, err, test.err)
			}

			if test.err == nil && f == nil {
				t.Fatalf("Opened %#v, got nil", test.name)
			}
		}
	})

	t.Run("Open NOK not loaded", func(t *testing.T) {
		gfs := NewWithClient(cacheClient, referenceGistID)
		_, err := gfs.Open("test1.txt")

		if err == nil {
			t.Fatalf("Opened without loading, got no error, want %#v", ErrNotLoaded)
		} else if err != ErrNotLoaded {
			t.Fatalf("Opened without loading, got an error %#v, want %#v", err, ErrNotLoaded)
		}
	})
}

func TestReadFile(t *testing.T) {
	t.Run("ReadFile OK", func(t *testing.T) {
		gfs := NewWithClient(cacheClient, referenceGistID)
		gfs.Load(context.Background())

		tests := []struct {
			name    string
			content string
			err     error
		}{
			{"test1.txt", "foobar\nbarfoo", nil},
			{"test2.txt", "olala\n12345\nabcde", nil},
			{"non-existing-file.txt", "", fs.ErrNotExist},
		}

		for _, test := range tests {
			b, err := gfs.ReadFile(test.name)

			if test.err != nil && !errors.Is(err, test.err) {
				t.Fatalf("Read file %#v, got error %#v, want %#v", test.name, err, test.err)
			}

			if test.err == nil && b == nil {
				t.Fatalf("Read file %#v, got nil", test.name)
			}

			if test.content != string(b) {
				t.Fatalf("Read file %#v, expected %#v but got %#v", test.name, test.content, string(b))
			}
		}
	})

}

func TestRead(t *testing.T) {
	gfs := NewWithClient(cacheClient, referenceGistID)
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
	gfs := NewWithClient(cacheClient, referenceGistID)
	gfs.Load(context.Background())

	t.Run("Stat OK", func(t *testing.T) {

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

				_, ok := stat.Sys().(*github.GistFile)
				if got, want := ok, true; got != want {
					t.Fatal("got Sys with type different from *github.GistFile, want it to be the case")
				}
			}
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

func TestReadDir(t *testing.T) {
	gfs := NewWithClient(cacheClient, referenceGistID)
	gfs.Load(context.Background())

	t.Run("OK", func(t *testing.T) {
		file, err := gfs.Open(".")
		if err != nil {
			t.Fatalf("Opening root directory, expected no error but got %#v", err)
		}

		dir, ok := file.(fs.ReadDirFile)
		if !ok {
			t.Fatal("Reading root directory, expected a ReadDirFile but got something else")
		}

		files, err := dir.ReadDir(-1)
		if err != nil {
			t.Fatalf("Reading root directory, expected no error but got %#v", err)
		}

		if got, want := len(files), 2; got != want {
			t.Fatalf("Reading root directory, got %#v files, want %#v", got, want)
		}
	})

	t.Run("OK subsequent reads", func(t *testing.T) {
		file, err := gfs.Open(".")
		if err != nil {
			t.Fatalf("Opening root directory, expected no error but got %#v", err)
		}

		dir, ok := file.(fs.ReadDirFile)
		if !ok {
			t.Fatal("Reading root directory, expected a ReadDirFile but got something else")
		}

		// first read
		files, err := dir.ReadDir(1)
		if err != nil {
			t.Fatalf("Reading root directory, expected no error but got %#v", err)
		}

		if got, want := len(files), 1; got != want {
			t.Fatalf("Reading root directory, got %#v files, want %#v", got, want)
		}

		// second read
		files, err = dir.ReadDir(1)
		if err != nil {
			t.Fatalf("Reading root directory, expected no error but got %#v", err)
		}

		if got, want := len(files), 1; got != want {
			t.Fatalf("Reading root directory, got %#v files, want %#v", got, want)
		}

		// last read (no entries left)
		files, err = dir.ReadDir(1)
		if err != nil {
			t.Fatalf("Reading root directory, expected no error but got %#v", err)
		}

		if got, want := len(files), 0; got != want {
			t.Fatalf("Reading root directory, got %#v files, want %#v", got, want)
		}
	})

	t.Run("OK ReadDir", func(t *testing.T) {
		files, err := gfs.ReadDir(".")
		if err != nil {
			t.Fatalf("Reading root directory, expected no error but got %#v", err)
		}

		if got, want := len(files), 2; got != want {
			t.Fatalf("Reading root directory, got %#v files, want %#v", got, want)
		}
	})

	t.Run("NOK ReadDir on a file", func(t *testing.T) {
		file, err := gfs.Open("test1.txt")
		if err != nil {
			t.Fatalf("Opening root directory, expected no error but got %#v", err)
		}

		dir, ok := file.(fs.ReadDirFile)
		if !ok {
			t.Fatal("Reading root directory, expected a ReadDirFile but got something else")
		}

		_, err = dir.ReadDir(-1)

		if _, ok := err.(*fs.PathError); !ok {
			t.Fatalf("Reading directory on a file, got %#v error, want %#v", err, &fs.PathError{})
		}
	})

	t.Run("NOK Read on a directory", func(t *testing.T) {
		file, err := gfs.Open(".")
		if err != nil {
			t.Fatalf("Opening root directory, expected no error but got %#v", err)
		}

		dir, ok := file.(fs.ReadDirFile)
		if !ok {
			t.Fatal("Reading root directory, expected a ReadDirFile but got something else")
		}

		b := make([]byte, 1)
		_, err = dir.Read(b)

		if _, ok := err.(*fs.PathError); !ok {
			t.Fatalf("Reading bytes on a directory, got %#v error, want %#v", err, &fs.PathError{})
		}
	})

	t.Run("OK Stat", func(t *testing.T) {
		file, err := gfs.Open(".")
		if err != nil {
			t.Fatalf("Opening root directory, expected no error but got %#v", err)
		}

		dir, ok := file.(fs.ReadDirFile)
		if !ok {
			t.Fatal("Reading root directory, expected a ReadDirFile but got something else")
		}

		stat, err := dir.Stat()
		if err != nil {
			t.Fatalf("Getting stat of root directory, expected no error but got %#v", err)
		}

		if got, want := stat.Name(), "./"; got != want {
			t.Fatalf("Reading name of root directory, got %#v files, want %#v", got, want)
		}

		if got, want := stat.ModTime(), approxModTime; got.After(approxModTime) &&
			got.Before(approxModTime.Add(24*time.Hour)) {
			t.Fatalf("got modTime %#v, want approx %#v", got, want)
		}

		if got, want := stat.IsDir(), true; got != want {
			t.Fatalf("got isDir %#v, want %#v", got, want)
		}
	})
}
