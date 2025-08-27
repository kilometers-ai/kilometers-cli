package appconfig

import (
	"context"

	configdomain "github.com/kilometers-ai/kilometers-cli/internal/core/domain/config"
	configports "github.com/kilometers-ai/kilometers-cli/internal/core/ports/config"
)

// Aggregator merges multiple loader snapshots, honoring priorities.
type Aggregator struct {
	loaders []configports.Loader
}

func NewAggregator(loaders ...configports.Loader) *Aggregator {
	return &Aggregator{loaders: loaders}
}

// LoadSnapshot returns the merged snapshot, including CLI overrides as priority 1
func (a *Aggregator) LoadSnapshot(ctx context.Context, overrides map[string]interface{}) (configdomain.Snapshot, error) {
	snap := make(configdomain.Snapshot)
	// CLI overrides priority 1
	for field, v := range overrides {
		snap[field] = configdomain.Entry{Key: field, Value: v, Source: "cli", SourcePath: "command_line_flag", Priority: 1}
	}

	for _, l := range a.loaders {
		s, _ := l.Load(ctx)
		snap.Merge(s)
	}
	return snap, nil
}
