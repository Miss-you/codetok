package provider

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// maxJSONLLineSize bounds a single JSONL line. Real session lines reach a few
// MB (large tool outputs, replayed histories); 64MB is generous headroom that
// still prevents a pathological line from causing unbounded allocations.
const maxJSONLLineSize = 64 * 1024 * 1024

// EachJSONLLine calls fn for every line read from r, without the trailing
// line terminator. Unlike bufio.Scanner it handles multi-megabyte lines, but
// returns an error once a line exceeds maxJSONLLineSize.
func EachJSONLLine(r io.Reader, fn func(line []byte)) error {
	br := bufio.NewReaderSize(r, 64*1024)
	var line []byte
	for {
		frag, err := br.ReadSlice('\n')
		line = append(line, frag...)
		if len(line) > maxJSONLLineSize {
			return fmt.Errorf("jsonl: line exceeds %d byte limit", maxJSONLLineSize)
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		if len(line) > 0 {
			fn(bytes.TrimRight(line, "\r\n"))
		}
		line = line[:0]
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}
