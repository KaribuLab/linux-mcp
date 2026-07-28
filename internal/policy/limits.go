package policy

// Default output caps (v1). Tuned for agent token budget, not host RAM.
const (
	MaxLines       = 100
	MaxBytes       = 64 * 1024
	MaxListEntries = 1000
	// PrefixBytes is the bounded window used for binary NUL check and private-key sniff.
	PrefixBytes = 8 * 1024
)

// Limits holds per-call read caps. Zero fields mean package defaults.
type Limits struct {
	MaxLines int
	MaxBytes int
}

func (l Limits) withDefaults() Limits {
	if l.MaxLines <= 0 {
		l.MaxLines = MaxLines
	}
	if l.MaxBytes <= 0 {
		l.MaxBytes = MaxBytes
	}
	return l
}
