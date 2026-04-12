package storage

import (
	"context"
	"time"

	containertypes "github.com/docker/docker/api/types/container"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type Container struct {
	ID         string
	SpanID     string
	ServiceID  string
	Created    time.Time
	Config     containertypes.Config
	HostConfig containertypes.HostConfig
	Platform   ocispec.Platform
}

type Image struct {}

type ImageStore interface {
	CreateImage(context.Context, string, *Image) error
	TagImage(context.Context, string, string) error
	GetImage(context.Context, string) (*Image, error)
	DeleteImage(context.Context, string) error
}

type ContainerStore interface {
	CreateContainer(context.Context, string, *Container) error
	NameContainer(context.Context, string, string) error
	GetContainer(context.Context, string) (*Container, error)
	UpdateContainer(context.Context, string, *Container) error
	DeleteContainer(context.Context, string) error
	ListContainers(context.Context) ([]Container, error)
}
