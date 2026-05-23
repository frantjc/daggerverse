package main

import (
	"context"
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

func (m *Release) run(ctx context.Context, container *dagger.Container, args ...string) error {
	if container == nil {
		container = m.Gh.Container
	}
	if _, err := container.WithExec(append([]string{"gh", "release", fmt.Sprintf("--repo=%s", m.Repo)}, args...)).Sync(ctx); err != nil {
		return err
	}
	return nil
}

const (
	workDir = "/tmp"
	bodyPath = workDir + "/body"
)

func (m *Gh) SetSecret(
	ctx context.Context,
	name string,
	body *dagger.Secret,
	// +optional
	app,
	// +optional
	env,
	// +optional
	org,
	// +optional
	visibility string,
	// +optional
	user,
	// +optional
	noReposSelected bool,
	// +optional
	repo []string,
) error {
	return m.set(ctx, "secret", name, body, app, env, org, visibility, user, noReposSelected, repo)
}

func (m *Gh) SetVariable(
	ctx context.Context,
	name string,
	body *dagger.Secret,
	// +optional
	env,
	// +optional
	org,
	// +optional
	visibility string,
	// +optional
	repo []string,
) error {
	return m.set(ctx, "variable", name, body, "", env, org, visibility, false, false, repo)
}

func (m *Gh) set(
	ctx context.Context,
	kind,
	name string,
	body *dagger.Secret,
	app,
	env,
	org,
	visibility string,
	user,
	noReposSelected bool,
	repo []string,
) error {
	args := []string{"gh", kind, "set", name}

	if app != "" {
		args = append(args, "--app", app)
	}

	if env != "" {
		args = append(args, "--env", env)
	}

	if org != "" {
		args = append(args, "--org", org)
	}

	if lenRepo := len(repo); lenRepo == 1 {
		args = append(args, "--repo", repo[0])
	} else if lenRepo > 0 {
		for _, r := range repo {
			args = append(args, "--repos", r)
		}
	}

	if noReposSelected {
		args = append(args, "--no-repos-selected")
	}

	if user {
		args = append(args, "--user")
	}

	if visibility != "" {
		args = append(args, "--visibility", visibility)
	}

	if _, err := m.Container.
		WithMountedSecret(bodyPath, body).
		WithExec(args, dagger.ContainerWithExecOpts{
			RedirectStdin: bodyPath,
		}).
		Sync(ctx); err != nil {
		return err
	}

	return nil
}
