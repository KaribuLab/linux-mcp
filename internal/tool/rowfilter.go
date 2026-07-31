package tool

import (
	"path/filepath"
	"regexp"
	"strings"
)

// matchTableRow applies list_grep-style pattern rules to a markdown data row.
// When extended=false and pattern looks like a glob, match against the
// identityCell-th markdown cell (1-based; e.g. Name=1 for list, Comm=2 for ps,
// Local=2 for ss). Otherwise use compiled RE2 (literal or regex) against the full row.
func matchTableRow(row, pattern string, extended, ignoreCase bool, re *regexp.Regexp, identityCell int) (bool, error) {
	if !extended && patternLooksLikeGlob(pattern) {
		name := markdownCell(row, identityCell)
		pat := pattern
		candidate := name
		if ignoreCase {
			pat = strings.ToLower(pat)
			candidate = strings.ToLower(candidate)
		}
		return filepath.Match(pat, candidate)
	}
	if re == nil {
		return false, nil
	}
	return re.MatchString(strings.TrimRight(row, "\n")), nil
}

// markdownCell returns the n-th cell (1-based) from a |a|b|c| row.
func markdownCell(row string, n int) string {
	line := strings.TrimRight(row, "\n")
	parts := strings.Split(line, "|")
	// parts[0] is empty before first |; parts[1] is first cell.
	if n < 1 || n >= len(parts) {
		return ""
	}
	return parts[n]
}

// compileRowFilter prepares a regexp when needed for matchTableRow.
func compileRowFilter(pattern string, extended, ignoreCase bool) (*regexp.Regexp, error) {
	if !extended && patternLooksLikeGlob(pattern) {
		return nil, nil
	}
	return compileGrepPattern(pattern, extended, ignoreCase)
}
