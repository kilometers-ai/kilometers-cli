package configdomain

// Entry represents a single configuration value with provenance and priority.
type Entry struct {
	Key        string
	Value      interface{}
	Source     string
	SourcePath string
	Priority   int
}

// Snapshot is a collection of config entries keyed by field name.
type Snapshot map[string]Entry

// Merge merges another snapshot into this one respecting priority
// (lower number indicates higher priority).
func (s Snapshot) Merge(other Snapshot) {
	for k, e := range other {
		if existing, ok := s[k]; !ok || e.Priority <= existing.Priority {
			s[k] = e
		}
	}
}
