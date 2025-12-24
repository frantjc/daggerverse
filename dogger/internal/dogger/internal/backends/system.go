package backends

import (
	"context"
	"time"

	systemrouter "github.com/docker/docker/api/server/router/system"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/backend"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/api/types/system"
)

type SystemBackend struct{}

var _ systemrouter.Backend = new(SystemBackend)

// AuthenticateToRegistry implements system.Backend.
func (s *SystemBackend) AuthenticateToRegistry(ctx context.Context, authConfig *registry.AuthConfig) (string, string, error) {
	return "", "", ErrUnimplemented
}

// SubscribeToEvents implements system.Backend.
func (s *SystemBackend) SubscribeToEvents(since, until time.Time, ef filters.Args) ([]events.Message, chan interface{}) {
	return nil, nil
}

// SystemDiskUsage implements system.Backend.
func (s *SystemBackend) SystemDiskUsage(ctx context.Context, opts backend.DiskUsageOptions) (*backend.DiskUsage, error) {
	return nil, ErrUnimplemented
}

// SystemInfo implements system.Backend.
func (s *SystemBackend) SystemInfo(context.Context) (*system.Info, error) {
	return nil, ErrUnimplemented
}

// SystemVersion implements system.Backend.
func (s *SystemBackend) SystemVersion(context.Context) (types.Version, error) {
	return types.Version{}, ErrUnimplemented
}

// UnsubscribeFromEvents implements system.Backend.
func (s *SystemBackend) UnsubscribeFromEvents(chan interface{}) {
}
