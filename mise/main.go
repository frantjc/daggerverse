package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver/v3"
	"github.com/frantjc/daggerverse/mise/internal/dagger"
	xslices "github.com/frantjc/x/slices"
)

type ConfigMinVersion struct {
	Hard string `toml:"hard,omitempty"`
	Soft string `toml:"soft,omitempty"`
}

// UnmarshalTOML implements [toml.Unmarshaler].
func (m *ConfigMinVersion) UnmarshalTOML(v any) error {
	switch val := v.(type) {
	case string:
		m.Hard = val
	case map[string]any:
		if hard, ok := val["hard"].(string); ok {
			m.Hard = hard
		}
		if soft, ok := val["soft"].(string); ok {
			m.Soft = soft
		}
	default:
		return fmt.Errorf("unexpected type %T for min_version", v)
	}
	return nil
}

var (
	_ toml.Unmarshaler = new(ConfigMinVersion)
)

type ConfigTool struct {
	Version string
	Depends []string
}

// UnmarshalTOML implements [toml.Unmarshaler].
func (t *ConfigTool) UnmarshalTOML(v any) error {
	switch val := v.(type) {
	case string:
		t.Version = val
	case map[string]any:
		if version, ok := val["version"].(string); ok {
			t.Version = version
		}
		switch depends := val["depends"].(type) {
		case string:
			t.Depends = []string{depends}
		case []any:
			for _, d := range depends {
				if str, ok := d.(string); ok {
					t.Depends = append(t.Depends, str)
				}
			}
		}
	default:
		return fmt.Errorf("unexpected type %T for tool", v)
	}
	return nil
}

var (
	_ toml.Unmarshaler = new(ConfigTool)
)

type ConfigEnvUnderscore struct {
	Source []string
	Path   []string
}

// UnmarshalTOML implements [toml.Unmarshaler].
func (e *ConfigEnvUnderscore) UnmarshalTOML(v any) error {
	val, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected type %T for env._", v)
	}
	toStrings := func(v any) []string {
		switch s := v.(type) {
		case string:
			return []string{s}
		case []any:
			var out []string
			for _, item := range s {
				if str, ok := item.(string); ok {
					out = append(out, str)
				}
			}
			return out
		}
		return nil
	}
	if f, ok := val["source"]; ok {
		e.Source = toStrings(f)
	}
	if p, ok := val["path"]; ok {
		e.Path = toStrings(p)
	}
	return nil
}

var (
	_ toml.Unmarshaler = new(ConfigEnvUnderscore)
)

type ConfigEnv struct {
	Underscore ConfigEnvUnderscore
	Vars       map[string]string
}

// UnmarshalTOML implements [toml.Unmarshaler].
func (e *ConfigEnv) UnmarshalTOML(v any) error {
	val, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected type %T for env", v)
	}
	for k, v := range val {
		if k == "_" {
			if err := e.Underscore.UnmarshalTOML(v); err != nil {
				return err
			}
			continue
		}
		if str, ok := v.(string); ok {
			if e.Vars == nil {
				e.Vars = map[string]string{}
			}
			e.Vars[k] = str
		}
	}
	return nil
}

var (
	_ toml.Unmarshaler = new(ConfigEnv)
)

type Config struct {
	MinVersion *ConfigMinVersion     `toml:"min_version,omitempty"`
	Tools      map[string]ConfigTool `toml:"tools,omitempty"`
	ToolAlias  map[string]string     `toml:"tool_alias,omitempty"`
	Env        ConfigEnv             `toml:"env,omitempty"`
}

var (
	// knownRegistry maps tool short names to their backends so that
	// `mise tool --json` only has to be run for tools not listed here.
	knownRegistry = map[string]string{
		"ansible-core": "pipx",
		"azure-cli":    "pipx",
		"go":           "core",
		"node":         "core",
		"opentofu":     "aqua",
		"pnpm":         "aqua",
	}
)

type Mise struct {
	// +private
	Config *Config
	// +private
	Src *dagger.Directory
}

func New(
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
	// +default="2026.5.15"
	version string,
) (*Mise, error) {
	c := &Config{
		MinVersion: &ConfigMinVersion{
			Hard: version,
		},
	}

	if ws != nil {
		config := ws.Directory(path).File("mise.toml")

		contents, err := config.Contents(ctx)
		if err != nil {
			return nil, err
		}

		if err := toml.Unmarshal([]byte(contents), c); err != nil {
			return nil, err
		}
	}

	return &Mise{
		Config: c,
		Src: ws.Directory(path, dagger.WorkspaceDirectoryOpts{
			Exclude:   exclude,
			Gitignore: gitignore,
		}),
	}, nil
}

func (m *Mise) Container(
	ctx context.Context,
	// +optional
	noEnv,
	// +optional
	noHooks bool,
	// +optional
	tools []string,
) (*dagger.Container, error) {
	lenTools := len(tools)

	var usesTool = func(tool string) bool {
		_, ok := m.Config.Tools[tool]
		return ok && (lenTools == 0 || slices.Contains(tools, tool))
	}
	var selected = func(tool string) bool {
		return lenTools == 0 || slices.Contains(tools, tool)
	}

	arch, err := dag.Arch().Microsoft(ctx)
	if err != nil {
		return nil, err
	}

	version, err := semver.NewVersion(m.Config.MinVersion.Hard)
	if err != nil {
		return nil, err
	}

	mise := dag.Archive().
		Untar(
			dag.HTTP(
				fmt.Sprintf(
					"http://github.com/jdx/mise/releases/download/v%s/mise-v%s-linux-%s.tar.gz",
					version, version, arch,
				),
			),
		).
		File("mise/bin/mise")

	probe := dag.Wolfi().
		Container().
		WithFile("/usr/local/bin/mise", mise)
	toolBackends := map[string]string{}
	var backendOf = func(name string) (string, error) {
		if backend, ok := toolBackends[name]; ok {
			return backend, nil
		}

		resolved := name
		if alias, ok := m.Config.ToolAlias[name]; ok {
			resolved = alias
		}

		backend, _, found := strings.Cut(resolved, ":")
		if !found {
			if backend, found = knownRegistry[resolved]; !found {
				stdout, err := probe.
					WithExec([]string{"mise", "tool", "--json", resolved}).
					Stdout(ctx)
				if err != nil {
					return "", err
				}

				info := struct {
					Backend string `json:"backend"`
				}{}
				if err := json.Unmarshal([]byte(stdout), &info); err != nil {
					return "", err
				}

				backend, _, _ = strings.Cut(info.Backend, ":")
			}
		}

		toolBackends[name] = backend
		return backend, nil
	}

	var usesBackendsWithoutDepends = func(backends []string, depends string) (bool, error) {
		for name, tool := range m.Config.Tools {
			if !selected(name) || slices.Contains(tool.Depends, depends) {
				continue
			}

			backend, err := backendOf(name)
			if err != nil {
				return false, err
			}

			if slices.Contains(backends, backend) {
				return true, nil
			}
		}
		return false, nil
	}
	packages := []string{"git", "bash"}
	var appendPackagesIf = func(cond bool, pkgs ...string) []string {
		if cond {
			return append(packages, pkgs...)
		}
		return packages
	}
	for _, b := range []struct {
		backends []string
		depends  string
		pkg      string
	}{
		{[]string{"pipx", "pypi"}, "uv", "uv"},
		{[]string{"go"}, "go", "go-1.27"},
		{[]string{"npm"}, "node", "nodejs-26"},
	} {
		uses, err := usesBackendsWithoutDepends(b.backends, b.depends)
		if err != nil {
			return nil, err
		}
		packages = appendPackagesIf(uses, b.pkg)
	}
	packages = appendPackagesIf(usesTool("opentofu"), "curl")
	packages = appendPackagesIf(usesTool("go"), "gcc")
	packages = appendPackagesIf(usesTool("node") || usesTool("pnpm"), "libatomic", "libstdc++")
	packages = xslices.Unique(packages)

	container := dag.Wolfi().
		Container(dagger.WolfiContainerOpts{
			Packages: packages,
		}).
		WithEnvVariable("HOME", "/root").
		WithEnvVariable("PATH", "$HOME/.local/bin:$PATH", dagger.ContainerWithEnvVariableOpts{
			Expand: true,
		}).
		WithFile("$HOME/.local/bin/mise", mise, dagger.ContainerWithFileOpts{
			Expand: true,
		})

	miseCacheDir, err := container.WithExec([]string{"mise", "cache", "path"}).Stdout(ctx)
	if err != nil {
		return nil, err
	}
	miseCacheDir = strings.TrimSpace(miseCacheDir)
	container = container.
		WithMountedCache(miseCacheDir, dag.CacheVolume("mise-cache"), dagger.ContainerWithMountedCacheOpts{
			Expand: true,
		})

	miseDataDir, err := container.EnvVariable(ctx, "MISE_DATA_DIR")
	if err != nil {
		return nil, err
	}
	if miseDataDir == "" {
		xdgDataHome, err := container.EnvVariable(ctx, "XDG_DATA_HOME")
		if err != nil {
			return nil, err
		} else if xdgDataHome == "" {
			home, err := container.EnvVariable(ctx, "HOME")
			if err != nil {
				return nil, err
			}

			xdgDataHome = filepath.Join(home, ".local", "share")
		}

		miseDataDir = filepath.Join(xdgDataHome, "mise")
	}
	container = container.
		WithMountedCache(miseDataDir, dag.CacheVolume("mise-data")).
		WithEnvVariable("PATH", fmt.Sprintf("%s:$PATH", filepath.Join(miseDataDir, "shims")), dagger.ContainerWithEnvVariableOpts{
			Expand: true,
		})

	container = container.
		With(func(r *dagger.Container) *dagger.Container {
			if noHooks {
				return r.WithEnvVariable("MISE_NO_HOOKS", "1")
			}
			return r
		}).
		With(func(r *dagger.Container) *dagger.Container {
			if noEnv {
				return r.WithEnvVariable("MISE_NO_ENV", "1")
			}
			return r
		})

	container = container.
		WithEnvVariable("MISE_YES", "1").
		WithWorkdir("/src").
		With(func(r *dagger.Container) *dagger.Container {
			include := []string{"mise.toml", "mise.*.toml"}
			if !noEnv {
				include = append(include, m.Config.Env.Underscore.Source...)
				include = append(include, m.Config.Env.Underscore.Path...)
			}
			include = xslices.Unique(include)
			return r.WithMountedDirectory(
				".",
				m.Src.Filter(dagger.DirectoryFilterOpts{
					Include: include,
				}),
			).
				WithExec(append([]string{"mise", "install"}, tools...)).
				WithMountedDirectory(".", m.Src)
		})

	miseEnv, err := container.WithExec([]string{"mise", "env"}).Stdout(ctx)
	if err != nil {
		return nil, err
	}

	envFile := new(strings.Builder)
	scanner := bufio.NewScanner(strings.NewReader(miseEnv))
	for scanner.Scan() {
		if _, err = fmt.Fprintln(envFile, strings.TrimPrefix(scanner.Text(), "export ")); err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	container = container.WithEnvFileVariables(dag.File(".env", envFile.String()).AsEnvFile())

	return container, nil
}

// +check
func (m *Mise) Doctor(
	ctx context.Context,
	// +optional
	noEnv,
	// +optional
	noHooks bool,
	// +optional
	tools []string,
	// +optional
	container *dagger.Container,
) error {
	container, err := m.Container(ctx,
		noEnv,
		noHooks,
		tools,
	)
	if err != nil {
		return err
	}
	_, err = container.
		WithExec([]string{"mise", "doctor"}).
		Sync(ctx)
	return err
}
