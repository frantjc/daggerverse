// A generated module for Trivy functions

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/frantjc/daggerverse/trivy/internal/dagger"
)

type Trivy struct {
	Container *dagger.Container
}

func New(
	ctx context.Context,
	// +optional
	modules *dagger.Directory,
	// +optional
	// +defaultAddress="docker.io/aquasec/trivy:0.72.0"
	container *dagger.Container,
) *Trivy {
	return &Trivy{
		Container: container.
			WithWorkdir("/src").
			With(func(r *dagger.Container) *dagger.Container {
				if xdgCacheHome, err := r.EnvVariable(ctx, "XDG_CACHE_HOME"); err != nil || xdgCacheHome != "" {
					return r
				}

				return r.WithEnvVariable("XDG_CACHE_HOME", "$HOME/.cache", dagger.ContainerWithEnvVariableOpts{
					Expand: true,
				})
			}).
			With(func(r *dagger.Container) *dagger.Container {
				if trivyCacheDir, err := r.EnvVariable(ctx, "TRIVY_CACHE_DIR"); err != nil || trivyCacheDir != "" {
					return r
				}

				return r.WithEnvVariable("TRIVY_CACHE_DIR", "$XDG_CACHE_HOME/trivy", dagger.ContainerWithEnvVariableOpts{
					Expand: true,
				})
			}).
			WithMountedCache("$TRIVY_CACHE_DIR", dag.CacheVolume("trivy"), dagger.ContainerWithMountedCacheOpts{
				Expand: true,
			}).
			With(func(r *dagger.Container) *dagger.Container {
				if modules != nil {
					return r.WithMountedDirectory("$TRIVY_MODULE_DIR", modules, dagger.ContainerWithMountedDirectoryOpts{
						Expand: true,
					})
				}
				return r
			}),
	}
}

// +check
func (m *Trivy) Repo(
	ctx context.Context,
	ws *dagger.Workspace,
	// +optional
	exclude []string,
	// +optional
	gitignore bool,
	// +optional
	// +default="."
	path string,
	// +optional
	disableTelemtry,
	// +optional
	offlineScan,
	// +optional
	skipVersionCheck,
	// +optional
	ignoreUnfixed bool,
	// +optional
	enableModules,
	// +optional
	severity,
	// +optional
	scanners,
	// +optional
	ignoreStatus []string,
) error {
	exec := []string{"trivy", "repo"}
	if len(enableModules) > 0 {
		exec = append(exec, fmt.Sprintf("--enable-modules=%s", strings.Join(enableModules, ",")))
	}
	if len(severity) > 0 {
		exec = append(exec, fmt.Sprintf("--severity=%s", strings.Join(severity, ",")))
	}
	if len(scanners) > 0 {
		exec = append(exec, fmt.Sprintf("--scanners=%s", strings.Join(scanners, ",")))
	}
	if len(ignoreStatus) > 0 {
		exec = append(exec, fmt.Sprintf("--ignore-status=%s", strings.Join(ignoreStatus, ",")))
	}
	if ignoreUnfixed {
		exec = append(exec, "--ignore-unfixed")
	}
	if disableTelemtry {
		exec = append(exec, "--disable-telemtry")
	}
	if offlineScan {
		exec = append(exec, "--offline-scan")
	}
	exec = append(exec, ".")
	_, err := m.Container.
		WithMountedDirectory(".", ws.Directory(path, dagger.WorkspaceDirectoryOpts{
			Exclude:   exclude,
			Gitignore: gitignore,
		})).
		WithExec(exec).
		Sync(ctx)
	return err
}

func (m *Trivy) Image(
	ctx context.Context,
	container *dagger.Container,
	// +optional
	disableTelemtry,
	// +optional
	offlineScan,
	// +optional
	skipVersionCheck,
	// +optional
	ignoreUnfixed bool,
	// +optional
	enableModules,
	// +optional
	severity,
	// +optional
	scanners,
	// +optional
	ignoreStatus []string,
) error {
	exec := []string{"trivy", "image"}
	if len(enableModules) > 0 {
		exec = append(exec, fmt.Sprintf("--enable-modules=%s", strings.Join(enableModules, ",")))
	}
	if len(severity) > 0 {
		exec = append(exec, fmt.Sprintf("--severity=%s", strings.Join(severity, ",")))
	}
	if len(scanners) > 0 {
		exec = append(exec, fmt.Sprintf("--scanners=%s", strings.Join(scanners, ",")))
	}
	if len(ignoreStatus) > 0 {
		exec = append(exec, fmt.Sprintf("--ignore-status=%s", strings.Join(ignoreStatus, ",")))
	}
	if ignoreUnfixed {
		exec = append(exec, "--ignore-unfixed")
	}
	if disableTelemtry {
		exec = append(exec, "--disable-telemtry")
	}
	if offlineScan {
		exec = append(exec, "--offline-scan")
	}
	name := "image.tar"
	exec = append(exec, fmt.Sprintf("--input=%s", name))

	_, err := m.Container.
		WithMountedFile(name, container.AsTarball().WithName(name)).
		WithExec(exec).
		Sync(ctx)
	return err
}
