package toolmeta

import (
	"fmt"
	"strconv"
	"strings"
)

// Blocked is the shared denial header: [blocked class=… path=…]
type Blocked struct {
	Class string
	Path  string
}

func (b Blocked) String() string {
	return fmt.Sprintf("[blocked class=%s path=%s]", b.Class, b.Path)
}

// CatHeader is the success metadata line for cat.
type CatHeader struct {
	Path      string
	Lines     int
	Truncated bool
	NextByte  int64 // set when Truncated; omitted from String when 0 and !Truncated
}

func (h CatHeader) String() string {
	var b strings.Builder
	b.WriteString("[cat path=")
	b.WriteString(h.Path)
	b.WriteString(" lines=")
	b.WriteString(strconv.Itoa(h.Lines))
	b.WriteString(" truncated=")
	b.WriteString(strconv.FormatBool(h.Truncated))
	if h.Truncated {
		b.WriteString(" next=")
		b.WriteString(strconv.FormatInt(h.NextByte, 10))
	}
	b.WriteByte(']')
	return b.String()
}

// ListHeader is the success metadata line for list.
type ListHeader struct {
	Path      string
	Returned  int
	Total     int  // 0 means unknown
	Truncated bool
	Next      int    // entry offset for next page; omitted when !Truncated
	Columns   string // comma-separated effective columns; only set for detailed (list=true) responses
}

func (h ListHeader) String() string {
	var b strings.Builder
	b.WriteString("[list path=")
	b.WriteString(h.Path)
	b.WriteString(" entries=")
	b.WriteString(strconv.Itoa(h.Returned))
	if h.Total > 0 {
		b.WriteByte('/')
		b.WriteString(strconv.Itoa(h.Total))
	}
	b.WriteString(" truncated=")
	b.WriteString(strconv.FormatBool(h.Truncated))
	if h.Truncated && h.Next > 0 {
		b.WriteString(" next=")
		b.WriteString(strconv.Itoa(h.Next))
	}
	if h.Columns != "" {
		b.WriteString(" columns=")
		b.WriteString(h.Columns)
	}
	b.WriteByte(']')
	return b.String()
}

// Render concatenates header and optional body with a single allocation path.
func Render(h fmt.Stringer, body *strings.Builder) string {
	var out strings.Builder
	hs := h.String()
	if body == nil || body.Len() == 0 {
		out.Grow(len(hs))
		out.WriteString(hs)
		return out.String()
	}
	out.Grow(len(hs) + 1 + body.Len())
	out.WriteString(hs)
	out.WriteByte('\n')
	out.WriteString(body.String())
	return out.String()
}
