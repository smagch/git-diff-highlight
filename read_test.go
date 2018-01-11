package git_diff_highlight

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func loadTestData(t *testing.T, filename string) io.Reader {
	f, err := os.Open(path.Join("testdata", filename) + ".txt")
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func readTestData(t *testing.T, filename string) []byte {
	r := loadTestData(t, filename)
	b, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestReader(t *testing.T) {
	testCases := []struct {
		input, expected string
	}{
		{"input-diff-with-color", "expected-with-color"},
		{"input-diff-without-color", "expected-without-color"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			r := NewReader(loadTestData(t, tc.input))
			got, err := ioutil.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			want := readTestData(t, tc.expected)
			if !bytes.Equal(got, want) {
				t.Fatalf("want\n%s\ngot\n%s", want, got)
			}
		})
	}
}
