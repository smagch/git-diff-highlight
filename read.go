package git_diff_highlight

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

const (
	highlightColor = "\x1b[7m"
	resetColor     = "\x1b[27m"
	colorRule      = `\x1b\[[0-9;]*m`
	minBufferSize  = 512
)

var (
	colorBlock    = regexp.MustCompile(colorRule)
	addedLine     = regexp.MustCompile("^(?:" + colorRule + `)*\+`)
	removedLine   = regexp.MustCompile("^(?:" + colorRule + `)*\-`)
	unchangedLine = regexp.MustCompile("^(?:" + colorRule + `)*\s`)
	hunkLine      = regexp.MustCompile("^(?:" + colorRule + `)*\@\@`)
)

type ttyPrinter struct {
	highlightColor, resetColor string
	buf                        *bytes.Buffer
}

func (p *ttyPrinter) Print(s string) {
	p.buf.WriteString(s)
}

func (p *ttyPrinter) PrintInsert(s string) {
	p.buf.WriteString(p.highlightColor)
	p.buf.WriteString(s)
	p.buf.WriteString(p.resetColor)
}

func (p *ttyPrinter) PrintDelete(s string) {
	p.buf.WriteString(p.highlightColor)
	p.buf.WriteString(s)
	p.buf.WriteString(p.resetColor)
}

func NewReader(r io.Reader) io.Reader {
	return &diffReader{
		scanner: bufio.NewScanner(r),
		printer: &ttyPrinter{
			highlightColor,
			resetColor,
			new(bytes.Buffer),
		},
	}
}

type diffReader struct {
	buf     bytes.Buffer
	inhunk  bool
	printer *ttyPrinter
	scanner *bufio.Scanner
	err     error
}

func (r *diffReader) Size() int {
	return r.printer.buf.Len()
}

// Read implement io.Reader interface.
func (r *diffReader) Read(p []byte) (n int, err error) {
	for r.Size() < minBufferSize && r.scanner.Scan() {
		r.hanldeLine(r.scanner.Text())
	}
	n, err = r.printer.buf.Read(p)
	if err == io.EOF && r.buf.Len() > 0 {
		r.dumpBuffer()
		return r.printer.buf.Read(p)
	}
	return
}

func (r *diffReader) hanldeLine(s string) {
	if !r.inhunk {
		if hunkLine.MatchString(s) {
			r.inhunk = true
		}
		r.printer.Print(s + "\n")
		return
	}
	if addedLine.MatchString(s) || removedLine.MatchString(s) {
		r.buf.WriteString(s)
		r.buf.WriteString("\n")
		return
	}
	if !unchangedLine.MatchString(s) && !hunkLine.MatchString(s) {
		r.inhunk = false
	}
	if r.buf.Len() > 0 {
		r.dumpBuffer()
	}
	r.printer.Print(s + "\n")
}

type diffPair struct {
	inserted, deleted bytes.Buffer
}

func (p *diffPair) oneSideEmpty() bool {
	return p.inserted.Len() == 0 || p.deleted.Len() == 0
}

func splitDiff(diff diffmatchpatch.Diff) (diffs []diffmatchpatch.Diff) {
	diff.Text = strings.TrimSpace(diff.Text)
	if !strings.Contains(diff.Text, "\n") {
		diffs = append(diffs, diff)
		return
	}

	text := diff.Text
	for text != "" {
		i := strings.Index(text, "\n")
		d := diffmatchpatch.Diff{
			Type: diff.Type,
		}
		if i == -1 {
			d.Text = strings.TrimSpace(text)
			diffs = append(diffs, d)
			return
		}

		d.Text = strings.TrimSpace(text[:i])
		newline := d
		newline.Text = "\n"
		diffs = append(diffs, d, newline)
		text = text[i+1:]
	}
	return
}

func (p *diffPair) diffs() []diffmatchpatch.Diff {
	diffs := diffmatchpatch.New().DiffMain(p.deleted.String(),
		p.inserted.String(), false)
	var safeDiffs []diffmatchpatch.Diff
	for _, diff := range diffs {
		diffs := splitDiff(diff)
		safeDiffs = append(safeDiffs, diffs...)
	}
	return safeDiffs
}

func getPair(b []byte) *diffPair {
	// sanitize color
	b = colorBlock.ReplaceAll(b, []byte(""))
	scanner := bufio.NewScanner(bytes.NewReader(b))
	var pair diffPair

	for scanner.Scan() {
		t := scanner.Text()
		if strings.HasPrefix(t, "-") {
			pair.deleted.WriteString(t[1:])
			pair.deleted.WriteString("\n")
		} else if strings.HasPrefix(t, "+") {
			pair.inserted.WriteString(t[1:])
			pair.inserted.WriteString("\n")
		} else {
			panic("Unexpected input: " + t)
		}
	}
	return &pair
}

func (r *diffReader) dumpBuffer() {
	pair := getPair(r.buf.Bytes())
	defer r.buf.Reset()

	// If one side is empty, then there is nothing to compare or highlight
	if pair.oneSideEmpty() {
		r.printer.Print(r.buf.String())
		r.printer.Print("\n")
		return
	}

	diffs := pair.diffs()

	s := r.buf.String()
	it := &diffIterator{
		diffs: diffs,
	}

	for s != "" {
		diff := it.Next()
		if diff == nil {
			r.printer.Print(s)
			s = ""
			break
		}

		i := strings.Index(s, diff.Text)
		// TODO don't panic
		if i == -1 {
			fmt.Printf("text is contains(%b): \"%s\"\n", strings.Contains(s, "\n"), s)
			fmt.Printf("buf is: \"%s\"\n", r.buf.String())
			panic("unknown string: \"" + diff.Text + "\"")
		}
		if diff.Type == diffmatchpatch.DiffEqual {
			r.printer.Print(s[:i+len(diff.Text)])
			s = s[i+len(diff.Text):]
			continue
		}

		if i != 0 {
			r.printer.Print(s[:i])
		}
		if diff.Type == diffmatchpatch.DiffDelete {
			r.printer.PrintDelete(diff.Text)
		} else {
			r.printer.PrintInsert(diff.Text)
		}
		s = s[i+len(diff.Text):]
	}
}

type diffIterator struct {
	i       int
	flipped bool
	diffs   []diffmatchpatch.Diff
}

func (it *diffIterator) Next() *diffmatchpatch.Diff {
	diff := it.next()
	if diff == nil && !it.flipped {
		it.flipped = true
		it.i = 0
		return it.next()
	}
	return diff
}

func (it *diffIterator) next() *diffmatchpatch.Diff {
	skipType := it.skipType()
	for it.i < len(it.diffs) {
		if it.diffs[it.i].Type == skipType {
			it.i++
			continue
		}
		i := it.i
		it.i++
		return &it.diffs[i]
	}
	return nil
}

func (it *diffIterator) skipType() diffmatchpatch.Operation {
	if it.flipped {
		return diffmatchpatch.DiffDelete
	}
	return diffmatchpatch.DiffInsert
}
