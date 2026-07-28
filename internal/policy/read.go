package policy

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Page is one bounded read from a file (no full-file cache).
type Page struct {
	Path      string
	Body      strings.Builder
	Lines     int
	Truncated bool
	NextByte  int64 // absolute file offset for resume; 0 if not truncated
}

// ReadPage opens path, enforces path/type policy, sniffs from the start of the
// file, Seeks to offset when offset > 0, then streams at most MaxLines ∩ MaxBytes.
func ReadPage(path string, offset int64, limits Limits) (*Page, error) {
	limits = limits.withDefaults()

	abs, err := CheckPath(path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(abs)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if err := CheckReadableType(abs, info); err != nil {
		return nil, err
	}

	// Always classify from file start so offset cannot skip a private-key header.
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek start: %w", err)
	}
	prefix := make([]byte, PrefixBytes)
	n, err := io.ReadFull(f, prefix)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		prefix = prefix[:n]
		err = nil
	}
	if err != nil && err != io.EOF {
		return nil, err
	}
	if class := ClassifyPrefix(prefix); class != "" {
		return nil, &Denied{Class: class, Path: abs}
	}

	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return nil, fmt.Errorf("resume not supported (seek to offset %d failed): %w", offset, err)
		}
	} else {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
	}

	page := &Page{Path: abs}
	br := bufio.NewReaderSize(f, 32*1024)
	bytesUsed := 0
	pos := offset

	for page.Lines < limits.MaxLines && bytesUsed < limits.MaxBytes {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			remain := limits.MaxBytes - bytesUsed
			chunk := line
			hitByteCap := false
			if len(chunk) > remain {
				chunk = chunk[:remain]
				hitByteCap = true
			}
			page.Body.Write(chunk)
			bytesUsed += len(chunk)
			pos += int64(len(chunk))
			page.Lines++
			if hitByteCap {
				page.Truncated = true
				page.NextByte = pos
				return page, nil
			}
			// Line without trailing newline at EOF still counts as a line.
		}
		if err == io.EOF {
			if page.Lines >= limits.MaxLines && len(line) > 0 && line[len(line)-1] == '\n' {
				// Exact line cap with complete last line: peek if more exists.
			}
			break
		}
		if err != nil {
			return nil, err
		}
	}

	if page.Lines >= limits.MaxLines {
		// If we stopped because of line cap, check whether more data remains.
		_, err := br.Peek(1)
		if err == nil {
			page.Truncated = true
			page.NextByte = pos
		} else if err != io.EOF {
			return nil, err
		}
	}
	return page, nil
}
