package main

import (
	"fmt"
	"io"
	"os"

	highlight "github.com/smagch/git-diff-highlight"
)

func main() {
	r := highlight.NewReader(os.Stdin)
	if _, err := io.Copy(os.Stdout, r); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}
