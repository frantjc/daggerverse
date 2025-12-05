package backends

import (
	"context"
	"fmt"
	"io"

	"dagger.io/dagger"
	"github.com/distribution/reference"
	imagerouter "github.com/docker/docker/api/server/router/image"
	"github.com/docker/docker/api/types/backend"
	"github.com/docker/docker/api/types/filters"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/image"
	"github.com/frantjc/daggerverse/daggerd/internal/storage"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type ImageBackend struct {
	Client  *dagger.Client
	Storage storage.ImageStore
}

var _ imagerouter.Backend = new(ImageBackend)

// ExportImage implements image.Backend.
func (i *ImageBackend) ExportImage(ctx context.Context, names []string, platform *v1.Platform, outStream io.Writer) error {
	return ErrUnimplemented
}

// GetImage implements image.Backend.
func (i *ImageBackend) GetImage(ctx context.Context, refOrID string, options backend.GetImageOpts) (*image.Image, error) {
	return nil, ErrUnimplemented
}

// ImageDelete implements image.Backend.
func (i *ImageBackend) ImageDelete(ctx context.Context, imageRef string, options imagetypes.RemoveOptions) ([]imagetypes.DeleteResponse, error) {
	return nil, ErrUnimplemented
}

// ImageHistory implements image.Backend.
func (i *ImageBackend) ImageHistory(ctx context.Context, imageName string, platform *v1.Platform) ([]*imagetypes.HistoryResponseItem, error) {
	return nil, ErrUnimplemented
}

// ImageInspect implements image.Backend.
func (i *ImageBackend) ImageInspect(ctx context.Context, refOrID string, options backend.ImageInspectOpts) (*imagetypes.InspectResponse, error) {
	return nil, ErrUnimplemented
}

// Images implements image.Backend.
func (i *ImageBackend) Images(ctx context.Context, opts imagetypes.ListOptions) ([]*imagetypes.Summary, error) {
	return nil, ErrUnimplemented
}

// ImagesPrune implements image.Backend.
func (i *ImageBackend) ImagesPrune(ctx context.Context, pruneFilters filters.Args) (*imagetypes.PruneReport, error) {
	return nil, ErrUnimplemented
}

// ImportImage implements image.Backend.
func (i *ImageBackend) ImportImage(ctx context.Context, ref reference.Named, platform *v1.Platform, msg string, layerReader io.Reader, changes []string) (image.ID, error) {
	return "", ErrUnimplemented
}

// LoadImage implements image.Backend.
func (i *ImageBackend) LoadImage(ctx context.Context, inTar io.ReadCloser, platform *v1.Platform, outStream io.Writer, quiet bool) error {
	return ErrUnimplemented
}

// PullImage implements image.Backend.
func (i *ImageBackend) PullImage(ctx context.Context, ref reference.Named, platform *v1.Platform, metaHeaders map[string][]string, authConfig *registry.AuthConfig, outStream io.Writer) error {
	dag := i.Client

	container := dag.Container(dagger.ContainerOpts{Platform: getDaggerPlatform(platform)})

	if authConfig != nil {
		container = container.WithRegistryAuth(
			authConfig.ServerAddress,
			authConfig.Username,
			dag.SetSecret("password", authConfig.Password),
		)
	}

	if _, err := container.From(ref.String()).Sync(ctx); err != nil {
		return err
	}

	return nil
}

// PushImage implements image.Backend.
func (i *ImageBackend) PushImage(ctx context.Context, ref reference.Named, platform *v1.Platform, metaHeaders map[string][]string, authConfig *registry.AuthConfig, outStream io.Writer) error {
	return ErrUnimplemented
}

// TagImage implements image.Backend.
func (i *ImageBackend) TagImage(ctx context.Context, id image.ID, newRef reference.Named) error {
	return ErrUnimplemented
}

func getDaggerPlatform(p *v1.Platform) dagger.Platform {
	platform := ""
	if p != nil {
		platform = fmt.Sprintf("%s/%s", p.OS, p.Architecture)
		if p.OSVersion != "" {
			platform = fmt.Sprintf("%s/%s", platform, p.OSVersion)
		}
	}
	return dagger.Platform(platform)
}
