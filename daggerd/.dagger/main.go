// A generated module for Daggerd functions

package main

import (
	"context"
	"dagger/daggerd/internal/dagger"
	_ "embed"
	"fmt"
	"path"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

type Daggerd struct {
	Source *dagger.Directory
}

func New(
	// +optional
	// +defaultPath="."
	src *dagger.Directory,
) *Daggerd {
	return &Daggerd{
		Source: src,
	}
}

func (m *Daggerd) Version(ctx context.Context) string {
	version := "0.0.0-unknown"

	ref, err := m.Source.AsGit().LatestVersion().Ref(ctx)
	if err == nil {
		version = strings.TrimPrefix(ref, "refs/tags/v")
	}

	return version
}

func (m *Daggerd) Binary(ctx context.Context) *dagger.File {
	return dag.Go(dagger.GoOpts{
		Module: m.Source.Filter(dagger.DirectoryFilterOpts{
			Exclude: []string{".github/**"},
		}),
		AdditionalWolfiPackages: []string{"gcc"},
	}).
		Build(dagger.GoBuildOpts{
			Pkg:     "./cmd/daggerd",
			Ldflags: "-s -w -X main.version=" + m.Version(ctx),
			Cgo:     true,
		})
}

func (m *Daggerd) Container(ctx context.Context) (*dagger.Container, error) {
	return dag.Wolfi().
		Container().
		WithFile("/usr/bin/daggerd", m.Binary(ctx)).
		WithEntrypoint([]string{"daggerd"}), nil
}

func (m *Daggerd) Service(ctx context.Context) (*dagger.Service, error) {
	container, err := m.Container(ctx)
	if err != nil {
		return nil, err
	}

	return container.
		WithExposedPort(1331).
		AsService(dagger.ContainerAsServiceOpts{ExperimentalPrivilegedNesting: true}), nil
}

func (m *Daggerd) Dev(ctx context.Context) (*dagger.Container, error) {
	service, err := m.Service(ctx)
	if err != nil {
		return nil, err
	}

	goModContents, err := m.Source.File("go.mod").Contents(ctx)
	if err != nil {
		return nil, err
	}

	parsedGoMod, err := modfile.Parse("go.mod", []byte(goModContents), nil)
	if err != nil {
		return nil, err
	}

	tag := "cli-alpine3.22"

	for _, require := range parsedGoMod.Require {
		if require.Mod.Path == "github.com/docker/docker" {
			tag = strings.TrimPrefix(semver.Canonical(require.Mod.Version), "v") + "-" + tag
			break
		}
	}

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

	return dag.Container().
		From("docker.io/library/docker:" + tag).
		WithServiceBinding("daggerd", service).
		WithEnvVariable("DOCKER_HOST", "tcp://daggerd:1331").
		WithEnvVariable("_EXPERIMENTAL_DAGGER_CLI_BIN", "/usr/bin/dagger").
		WithFile(
			"$_EXPERIMENTAL_DAGGER_CLI_BIN",
			dag.Wolfi().
				Container().
				WithFile(tmpDaggerTgzPath, daggerTgz).
				WithExec([]string{
					"tar", "-xzf", tmpDaggerTgzPath, "-C", path.Dir(tmpDaggerPath), path.Base(tmpDaggerPath),
				}).
				File(tmpDaggerPath),
			dagger.ContainerWithFileOpts{Expand: true},
		).
		Terminal(dagger.ContainerTerminalOpts{ExperimentalPrivilegedNesting: true}), nil
}
