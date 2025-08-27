package configports

import (
	"context"

	configdomain "github.com/kilometers-ai/kilometers-cli/internal/core/domain/config"
)

type Loader interface {
	Load(ctx context.Context) (configdomain.Snapshot, error)
	Name() string
}

type Storage interface {
	LoadSaved(ctx context.Context) (configdomain.Snapshot, error)
	Save(ctx context.Context, snap configdomain.Snapshot) error
}

type Validator interface {
	Validate(snap configdomain.Snapshot) error
}
