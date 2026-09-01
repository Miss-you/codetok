package provider

import (
	"bufio"
	"bytes"
	"io"
)

// EachJSONLLine calls fn for every line read from r, without the trailing
// line terminator. Unlike bufio.Scanner it imposes no line-length limit,
// so sessions containing very long lines (large tool outputs, replays) are
// parsed instead of being dropped entirely.
func EachJSONLLine(r io.Reader, fn func(line []byte)) error {
	br := bufio.NewReaderSize(r, 64*1024)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			fn(bytes.TrimRight(line, "\r\n"))
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}
