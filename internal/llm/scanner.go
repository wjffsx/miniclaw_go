package llm

import (
	"bufio"
	"io"
)

type lineScanner struct {
	scanner *bufio.Scanner
}

func newLineScanner(r io.Reader) *lineScanner {
	return &lineScanner{
		scanner: bufio.NewScanner(r),
	}
}

func (ls *lineScanner) Scan() bool {
	return ls.scanner.Scan()
}

func (ls *lineScanner) Text() string {
	return ls.scanner.Text()
}

func (ls *lineScanner) Err() error {
	return ls.scanner.Err()
}