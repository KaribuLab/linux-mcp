package tool

// showFlag reports whether an optional Show* flag is visible.
// nil or true => visible; false => hidden.
func showFlag(v *bool) bool {
	return v == nil || *v
}

// visibleColumnsOrdered returns identity columns (always on) followed by
// optional columns from order that are enabled in flags. Order is fixed;
// flags only include/omit.
func visibleColumnsOrdered(identity []string, order []string, flags map[string]bool) []string {
	out := make([]string, 0, len(identity)+len(order))
	out = append(out, identity...)
	for _, c := range order {
		if flags[c] {
			out = append(out, c)
		}
	}
	return out
}

func joinColumns(cols []string) string {
	if len(cols) == 0 {
		return ""
	}
	n := 0
	for _, c := range cols {
		n += len(c)
	}
	b := make([]byte, 0, n+len(cols)-1)
	for i, c := range cols {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, c...)
	}
	return string(b)
}
