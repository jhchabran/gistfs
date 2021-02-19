# GistFS

GistFS is an `io/fs` implementation that enables to read files stored in a given Gist.

## Requirements

This module depends on `io/fs` which is only available since [go 1.16](https://tip.golang.org/doc/go1.16).

## Usage

GistFS is threadsafe.

```go
package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jhchabran/gistfs"
)

func main() {
	// create a FS based on https://gist.github.com/jhchabran/ded2f6727d98e6b0095e62a7813aa7cf
	gfs := gistfs.New("ded2f6727d98e6b0095e62a7813aa7cf")

	// load the remote content once for all,
	// ie, no more API calls toward Github will be made.
	err := gfs.Load(context.Background())
	if err != nil {
		panic(err)
	}

	// --- base API
	// open the "test1.txt" file
	f, err := gfs.Open("test1.txt")
	if err != nil {
		panic(err)
	}

	// read its content
	b := make([]byte, 1024)
	_, err = f.Read(b)

	if err != nil {
		panic(err)
	}

	fmt.Println(string(b))

	// --- ReadFile API
	// directly read the "test1.txt" file
	b, err = gfs.ReadFile("test1.txt")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(b))

	// --- ReadDir API
	// there is only one directory in a gistfile, the root dir "."
	files, err := gfs.ReadDir(".")
	if err != nil {
		panic(err)
	}

	for _, entry := range files {
		fmt.Println(entry.Name())
	}

	// --- Serve the files from the gists over http
	http.ListenAndServe(":8080", http.FileServer(http.FS(gfs)))
}
```

## See also

- [io/fs godoc](https://pkg.go.dev/io/fs)
