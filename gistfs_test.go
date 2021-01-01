package gistfs

import (
	"context"
	"fmt"
	"testing"
)

func TestOK(t *testing.T) {
	gfs := New("ded2f6727d98e6b0095e62a7813aa7cf")
	gfs.Load(context.Background())
	f, err := gfs.Open("test1.txt")

	if err != nil {
		t.Fatal(err)
	}

	b := make([]byte, 30000)

	_, err = f.Read(b)

	fmt.Println(string(b))
}
