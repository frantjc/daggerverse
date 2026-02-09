package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/frantjc/daggerverse/dagger/internal/dagger"
)

type Dagger struct{}

func (m *Dagger) Binary(ctx context.Context) (*dagger.File, error) {
	version, err := dag.Version(ctx)
	if err != nil {
		return nil, err
	}

	osPlatformVersion, err := dag.DefaultPlatform(ctx)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(string(osPlatformVersion), "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid dagger platform %s", osPlatformVersion)
	}

	platform := parts[1]

	daggerTgz := dag.HTTP(
		fmt.Sprintf(
			"https://github.com/dagger/dagger/releases/download/%s/dagger_%s_linux_%s.tar.gz",
			version, version, platform,
		),
	)

	tmpDaggerTgzPath := "/tmp/dagger.tgz"
	tmpDaggerPath := "/tmp/dagger"

	return dag.Wolfi().
		Container().
		WithFile(tmpDaggerTgzPath, daggerTgz).
		WithExec([]string{
			"tar", "-xzf", tmpDaggerTgzPath, "-C", filepath.Dir(tmpDaggerPath), filepath.Base(tmpDaggerPath),
		}).
		File(tmpDaggerPath), nil
}

// func (m *Dagger) Container(
// 	ctx context.Context,
// 	// +default="/usr/local/bin/dagger"
// 	path string,
// 	// +optional
// 	container *dagger.Container,
// 	// +optional
// 	opts dagger.ContainerWithFileOpts,
// ) (*dagger.Container, error) {
// 	if container == nil {
// 		container = dag.Wolfi().Container()
// 	}

// 	binary, err := m.Binary(ctx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return container.
// 		WithEnvVariable("_EXPERIMENTAL_DAGGER_CLI_BIN", path).
// 		WithFile("$_EXPERIMENTAL_DAGGER_CLI_BIN", binary, dagger.ContainerWithFileOpts{Expand: true}, opts), nil
// }
