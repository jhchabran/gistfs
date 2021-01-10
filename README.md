# GistFS

GistFS is an `io/fs` implementation that enables to read files stored in a given Gist.

## Requirements

This module depends on `io/fs` which is only available in [go 1.16 beta1](https://tip.golang.org/doc/go1.16).
To install it, run the following commands:

```sh
go get golang.org/dl/go1.16beta1
go1.16beta1 download

# From there, just use go1.16beta1 instead of the go command
go1.16beta1 run ...
```

## Usage

GistFS is threadsafe.

```go
package main

import (
	"context"
	"fmt"

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

	// --- Serve the files from the gists over http
	http.ListenAndServe(":8080", http.FileServer(http.FS(gfs)))
}
```

## See also

- [Draft design for the file system interface.](https://go.googlesource.com/proposal/+/master/design/draft-iofs.md)
- [Prototype code of io/fs](https://go-review.googlesource.com/c/go/+/243939)
- [Managing Go installations](https://golang.org/doc/manage-install)
