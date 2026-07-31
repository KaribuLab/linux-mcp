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

// FindHeader is the success metadata line for find.
type FindHeader struct {
	Path      string
	Returned  int
	Total     int
	Truncated bool
	Visited   int
}

func (h FindHeader) String() string {
	var b strings.Builder
	b.WriteString("[find path=")
	b.WriteString(h.Path)
	b.WriteString(" matches=")
	b.WriteString(strconv.Itoa(h.Returned))
	b.WriteByte('/')
	b.WriteString(strconv.Itoa(h.Total))
	b.WriteString(" truncated=")
	b.WriteString(strconv.FormatBool(h.Truncated))
	b.WriteString(" visited=")
	b.WriteString(strconv.Itoa(h.Visited))
	b.WriteByte(']')
	return b.String()
}

// GrepHeader is the success metadata line for grep.
type GrepHeader struct {
	Path         string
	Returned     int
	Total        int
	Truncated    bool
	FilesScanned int
	// Redacted counts matching rows whose content came from a file classified
	// as private-key and was replaced by a fixed redaction placeholder.
	Redacted int
}

func (h GrepHeader) String() string {
	var b strings.Builder
	b.WriteString("[grep path=")
	b.WriteString(h.Path)
	b.WriteString(" matches=")
	b.WriteString(strconv.Itoa(h.Returned))
	b.WriteByte('/')
	b.WriteString(strconv.Itoa(h.Total))
	b.WriteString(" truncated=")
	b.WriteString(strconv.FormatBool(h.Truncated))
	b.WriteString(" filesScanned=")
	b.WriteString(strconv.Itoa(h.FilesScanned))
	b.WriteString(" redacted=")
	b.WriteString(strconv.Itoa(h.Redacted))
	b.WriteByte(']')
	return b.String()
}

// ListGrepHeader is the success metadata line for list_grep (list|grep).
type ListGrepHeader struct {
	Path      string
	Returned  int
	Total     int // rows that matched the pattern (within the listed window)
	Truncated bool
	Next      int    // entry offset hint when listing base was truncated
	Columns   string // comma-separated columns when list=true
}

func (h ListGrepHeader) String() string {
	var b strings.Builder
	b.WriteString("[list_grep path=")
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

// FindGrepHeader is the success metadata line for find_grep (find|xargs grep).
type FindGrepHeader struct {
	Path         string
	Returned     int
	Total        int
	Truncated    bool
	FilesScanned int
	Redacted     int
	Visited      int
}

func (h FindGrepHeader) String() string {
	var b strings.Builder
	b.WriteString("[find_grep path=")
	b.WriteString(h.Path)
	b.WriteString(" matches=")
	b.WriteString(strconv.Itoa(h.Returned))
	b.WriteByte('/')
	b.WriteString(strconv.Itoa(h.Total))
	b.WriteString(" truncated=")
	b.WriteString(strconv.FormatBool(h.Truncated))
	b.WriteString(" filesScanned=")
	b.WriteString(strconv.Itoa(h.FilesScanned))
	b.WriteString(" redacted=")
	b.WriteString(strconv.Itoa(h.Redacted))
	b.WriteString(" visited=")
	b.WriteString(strconv.Itoa(h.Visited))
	b.WriteByte(']')
	return b.String()
}

// PsHeader is the success metadata line for ps.
type PsHeader struct {
	Returned  int
	Total     int
	Truncated bool
	Columns   string
}

func (h PsHeader) String() string {
	var b strings.Builder
	b.WriteString("[ps entries=")
	b.WriteString(strconv.Itoa(h.Returned))
	b.WriteByte('/')
	b.WriteString(strconv.Itoa(h.Total))
	b.WriteString(" truncated=")
	b.WriteString(strconv.FormatBool(h.Truncated))
	if h.Columns != "" {
		b.WriteString(" columns=")
		b.WriteString(h.Columns)
	}
	b.WriteByte(']')
	return b.String()
}

// PsGrepHeader is the success metadata line for ps_grep.
type PsGrepHeader struct {
	Returned  int
	Total     int
	Truncated bool
	Columns   string
}

func (h PsGrepHeader) String() string {
	var b strings.Builder
	b.WriteString("[ps_grep entries=")
	b.WriteString(strconv.Itoa(h.Returned))
	b.WriteByte('/')
	b.WriteString(strconv.Itoa(h.Total))
	b.WriteString(" truncated=")
	b.WriteString(strconv.FormatBool(h.Truncated))
	if h.Columns != "" {
		b.WriteString(" columns=")
		b.WriteString(h.Columns)
	}
	b.WriteByte(']')
	return b.String()
}

// SsHeader is the success metadata line for ss.
type SsHeader struct {
	Returned  int
	Total     int
	Truncated bool
	Columns   string
}

func (h SsHeader) String() string {
	var b strings.Builder
	b.WriteString("[ss entries=")
	b.WriteString(strconv.Itoa(h.Returned))
	b.WriteByte('/')
	b.WriteString(strconv.Itoa(h.Total))
	b.WriteString(" truncated=")
	b.WriteString(strconv.FormatBool(h.Truncated))
	if h.Columns != "" {
		b.WriteString(" columns=")
		b.WriteString(h.Columns)
	}
	b.WriteByte(']')
	return b.String()
}

// SsGrepHeader is the success metadata line for ss_grep.
type SsGrepHeader struct {
	Returned  int
	Total     int
	Truncated bool
	Columns   string
}

func (h SsGrepHeader) String() string {
	var b strings.Builder
	b.WriteString("[ss_grep entries=")
	b.WriteString(strconv.Itoa(h.Returned))
	b.WriteByte('/')
	b.WriteString(strconv.Itoa(h.Total))
	b.WriteString(" truncated=")
	b.WriteString(strconv.FormatBool(h.Truncated))
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
