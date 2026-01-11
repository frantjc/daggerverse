package main

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/frantjc/daggerverse/release/internal/dagger"
)

type Release struct {
	Source *dagger.GitRef
}

var (
	//go:embed cask.rb.tpl
	caskRbTpl string
)

type Cask struct {
	Homepage string
	Description string
	LinuxURL string
	LinuxSha256 string
	DarwinURL string
	DarwinSha256 string
}

func (m *Release) Create(
	ctx context.Context,
	githubToken *dagger.Secret,
	githubRepo,
	// +optional
	pkg string,
	// +optional
	cgo,
	// +optional
	brew bool,
) error {
	gh := dag.Gh(githubToken)
	module := m.Source.Tree()

	// description, err := gh.Container().
	// 	WithDirectory(".", module).
	// 	WithExec([]string{"gh", "repo", "view", "--json", "description", "--jq", ".description"}).
	// 	Stdout(ctx)
	// if err != nil {
	// 	return err
	// }

	ref, err := m.Source.Ref(ctx)
	if err != nil {
		return err
	}
	tag := strings.TrimPrefix(ref, "refs/tags/")

	g0 := dag.Go(dagger.GoOpts{
		Module: module,
	})

	assets := []*dagger.File{}
	checksumsTxt := new(strings.Builder)

	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "arm64"} {
			bin := g0.Build(dagger.GoBuildOpts{
				Pkg: pkg,
				Ldflags: "-s -w -X main.version=" + tag,
				Cgo: cgo,
				Goarch: goarch,
				Goos: goos,
			})

			name, err := bin.Name(ctx)
			if err != nil {
				return err
			}

			asset := bin.WithName(fmt.Sprintf("%s_%s_%s_%s", name, tag, goos, goarch))
			assets = append(assets, asset)
		}
	}

	for _, asset := range assets {
		name, err := asset.Name(ctx)
		if err != nil {
			return err
		}

		checksum, err := asset.Digest(ctx)
		if err != nil {
			return err
		}

		if _, err = fmt.Fprintln(checksumsTxt, name, checksum); err != nil {
			return err
		}
	}


	assets = append(assets, dag.File("checksums.txt", checksumsTxt.String()))

	if err := gh.
		Release().
		Create(ctx, tag, githubRepo, dagger.GhReleaseCreateOpts{Assets: assets}); err != nil {
		return err
	}

	if brew {
		owner, _, _ := strings.Cut(githubRepo, "/")
		var _ = owner
	}

	return nil
}
