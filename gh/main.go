package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/logsquaredn/rubber/.dagger/modules/gh/internal/dagger"
)

type Gh struct {
	Container *dagger.Container
}

func New(
	githubToken *dagger.Secret,
	// +optional
	container *dagger.Container,
) *Gh {
	if container == nil {
		container = dag.Wolfi().
			Container(dagger.WolfiContainerOpts{
				Packages: []string{"gh"},
			})
	}
	return &Gh{
		Container: container.WithSecretVariable("GITHUB_TOKEN", githubToken),
	}
}

type Release struct {
	// +private
	Gh *Gh
	// +private
	Repo string
	// +private
	Tag string
}

func (m *Gh) Release(repo, tag string) *Release {
	return &Release{Gh: m, Repo: repo, Tag: tag}
}

func (m *Release) Create(
	ctx context.Context,
	// +optional
	title string,
	// +optional
	generateNotes,
	// +optional
	latest,
	// +optional
	prerelease,
	// +optional
	draft,
	// +optional
	verifyTag bool,
) error {
	args := []string{"create", m.Tag}

	if title != "" {
		args = append(args, fmt.Sprintf("--title=%s", title))
	}

	if generateNotes {
		args = append(args, "--generate-notes")
	}

	if latest {
		args = append(args, "--latest")
	} else {
		if prerelease {
			args = append(args, "--prerelease")
		}

		if draft {
			args = append(args, "--draft")
		}
	}

	if verifyTag {
		args = append(args, "--verify-tag")
	}

	return m.run(ctx, nil, args...)
}

func (m *Release) Edit(
	ctx context.Context,
	// +optional
	title string,
	// +optional
	generateNotes,
	// +optional
	latest,
	// +optional
	prerelease,
	// +optional
	draft,
	// +optional
	verifyTag bool,
) error {
	args := []string{"edit", m.Tag}

	if title != "" {
		args = append(args, fmt.Sprintf("--title=%s", title))
	}

	if generateNotes {
		args = append(args, "--generate-notes")
	}

	if latest {
		args = append(args, "--latest", "--draft=false", "--prerelease=false")
	} else {
		if prerelease {
			args = append(args, "--prerelease")
		}

		if draft {
			args = append(args, "--draft")
		}
	}

	if verifyTag {
		args = append(args, "--verify-tag")
	}

	return m.run(ctx, nil, args...)
}

func (m *Release) Upload(
	ctx context.Context,
	assets []*dagger.File,
	// +optional
	clobber bool,
) error {
	container := m.Gh.Container
	args := []string{"upload", m.Tag}

	if clobber {
		args = append(args, "--clobber")
	}

	for _, asset := range assets {
		name, err := asset.Name(ctx)
		if err != nil {
			return err
		}

		container = container.
			WithFile(
				name,
				asset,
			)

		args = append(args, name)
	}

	return m.run(ctx, container, args...)
}

func (m *Release) Assets(
	ctx context.Context,
) ([]ReleaseAsset, error) {
	out, err := m.runOutput(ctx, m.Gh.Container, "view", "--json", "assets")
	if err != nil {
		return nil, err
	}

	var view struct {
		Assets []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"assets"`
	}
	if err := json.Unmarshal([]byte(out), &view); err != nil {
		return nil, err
	}

	assets := make([]ReleaseAsset, len(view.Assets))
	for i, asset := range view.Assets {
		assets[i] = ReleaseAsset{Release: *m, Name: asset.Name, Dig: asset.Digest}
	}
	return assets, nil
}

func (m *Release) Download(
	ctx context.Context,
	pattern []string,
	// +optional
	archive string,
) *dagger.Directory {
	container := m.Gh.Container
	args := []string{"download", m.Tag, "--dir=."}

	for _, p := range pattern {
		args = append(args, fmt.Sprintf("--pattern=%s", p))
	}

	if archive != "" {
		args = append(args, fmt.Sprintf("--archive=%s", archive))
	}

	return container.
		WithWorkdir("/dl").
		WithExec(append([]string{"gh", "release", fmt.Sprintf("--repo=%s", m.Repo)}, args...)).
		Directory("/dl")
}

func (m *Release) run(ctx context.Context, container *dagger.Container, args ...string) error {
	if container == nil {
		container = m.Gh.Container
	}
	if _, err := container.WithExec(append([]string{"gh", "release", fmt.Sprintf("--repo=%s", m.Repo)}, args...)).Sync(ctx); err != nil {
		return err
	}
	return nil
}

func (m *Release) runOutput(ctx context.Context, container *dagger.Container, args ...string) (string, error) {
	if container == nil {
		container = m.Gh.Container
	}
	return container.WithExec(append([]string{"gh", "release", fmt.Sprintf("--repo=%s", m.Repo)}, args...)).Stdout(ctx)
}

type ReleaseAsset struct {
	// +private
	Release
	// +private
	Name string
	// +private
	Dig string
}

func (m *ReleaseAsset) File(ctx context.Context) *dagger.File {
	return m.Release.Download(ctx, []string{m.Name}, "").File(m.Name)
}

func (m *ReleaseAsset) Digest() string {
	return m.Dig
}
