package gistfs

import (
	"fmt"
	"testing"
)

func TestOK(t *testing.T) {
	gfs := New("e4d1a1b18aeabd7d15427f4afb34cd8e")
	f, err := gfs.Open("usernames.json")

	if err != nil {
		t.Fatal(err)
	}

	b := make([]byte, 30000)

	_, err = f.Read(b)

	fmt.Println(string(b))
}
