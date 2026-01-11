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
	CaskRbTpl string
)

func (m *Release) Create(
	ctx context.Context,
	githubToken *dagger.Secret,
	githubRepo,
	// +optional
	pkg string,
	// +optional
	cgo bool,
) error {
	gh := dag.Gh(githubToken)
	module := m.Source.Tree()

	ref, err := m.Source.Ref(ctx)
	if err != nil {
		return err
	}
	tag := strings.TrimPrefix(ref, "refs/tags/")

	// description, err := gh.Container().
	// 	WithDirectory(".", module).
	// 	WithExec([]string{"gh", "repo", "view", "--json", "description", "--jq", ".description"}).
	// 	Stdout(ctx)
	// if err != nil {
	// 	return err
	// }

	g0 := dag.Go(dagger.GoOpts{
		Module: module,
	})
	name := ""

	assets := []*dagger.File{}
	checksumsTxt := new(strings.Builder)
	upx := dag.Upx()
	tar := dag.Tar()

	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "arm64"} {
			bin := upx.Pack(
				g0.Build(dagger.GoBuildOpts{
					Pkg:     pkg,
					Ldflags: "-s -w -X main.version=" + tag,
					Cgo:     cgo,
					Goarch:  goarch,
					Goos:    goos,
				}),
			)

			if name == "" {
				name, err = bin.Name(ctx)
				if err != nil {
					return err
				}
			}

			asset := tar.
				Create(
					module.Filter(dagger.DirectoryFilterOpts{
						Include: []string{
							"README.md",
							"LICENSE",
						},
					}).
						WithFile(
							fmt.Sprintf("%s-%s-%s-%s.tgz", name, tag, goos, goarch),
							bin,
						),
				)
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

	return nil
}
