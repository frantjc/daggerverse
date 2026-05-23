package main

import (
	"context"
	"fmt"
	"slices"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver/v3"
	"github.com/frantjc/daggerverse/mise/internal/dagger"
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
	Env        ConfigEnv             `toml:"env,omitempty"`
}

type Mise struct {
	Version string
	// +private
	Source *dagger.Directory
	// +private
	Config *Config
}

func New(
	// +optional
	// +default="2026.5.15"
	version string,
) (*Mise, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return nil, err
	}
	return &Mise{
		Version: v.String(),
		Config: &Config{
			MinVersion: &ConfigMinVersion{
				Hard: v.String(),
			},
		},
		Source: dag.Directory(),
	}, nil
}

func (m *Mise) WithConfigFile(
	ctx context.Context,
	config *dagger.File,
) (*Mise, error) {
	contents, err := config.Contents(ctx)
	if err != nil {
		return nil, err
	}

	if err := toml.Unmarshal([]byte(contents), m.Config); err != nil {
		return nil, err
	}

	v, err := semver.NewVersion(m.Version)
	if err != nil {
		return nil, err
	}

	var cv *semver.Version
	if m.Config.MinVersion.Hard != "" {
		if cv, err = semver.NewVersion(m.Config.MinVersion.Hard); err != nil {
			return nil, err
		}
	} else if m.Config.MinVersion.Soft != "" {
		if cv, err = semver.NewVersion(m.Config.MinVersion.Soft); err != nil {
			return nil, err
		}
	}
	if cv != nil && v.LessThan(cv) {
		m.Version = cv.String()
	}

	m.Source = m.Source.WithFile("mise.toml", config)
	return m, nil
}

func (m *Mise) WithSource(ctx context.Context, source *dagger.Directory) (*Mise, error) {
	m.Source = source
	return m.WithConfigFile(ctx, m.Source.File("mise.toml"))
}

func (m *Mise) Binary(ctx context.Context) (*dagger.File, error) {
	arch, err := dag.Arch().Microsoft(ctx)
	if err != nil {
		return nil, err
	}

	return dag.Archive().
		Untar(
			dag.HTTP(
				fmt.Sprintf(
					"http://github.com/jdx/mise/releases/download/v%s/mise-v%s-linux-%s.tar.gz",
					m.Version, m.Version, arch,
				),
			),
		).
		File("mise/bin/mise"), nil
}

func (m *Mise) Container(
	ctx context.Context,
	// +optional
	noEnv,
	// +optional
	noHooks,
	// +optional
	install bool,
	// +optional
	tools []string,
) (*dagger.Container, error) {
	mise, err := m.Binary(ctx)
	if err != nil {
		return nil, err
	}

	lenTools := len(tools)
	if lenTools > 0 {
		install = true
	}

	// mise's node install has some system dependencies.
	packages := []string{}
	if _, hasNode := m.Config.Tools["node"]; hasNode && (lenTools == 0 || slices.Contains(tools, "node")) {
		packages = append(packages, "bash", "libatomic", "libstdc++")
	}

	return dag.Wolfi().
		Container(dagger.WolfiContainerOpts{
			Packages: packages,
		}).
		WithEnvVariable("HOME", "/root").
		WithMountedCache("$HOME/.local/share/mise", dag.CacheVolume("mise-data"), dagger.ContainerWithMountedCacheOpts{
			Expand: true,
		}).
		WithMountedCache("$HOME/.cache/mise", dag.CacheVolume("mise-cache"), dagger.ContainerWithMountedCacheOpts{
			Expand: true,
		}).
		WithEnvVariable("PATH", "$HOME/.local/share/mise/shims:$PATH", dagger.ContainerWithEnvVariableOpts{
			Expand: true,
		}).
		WithFile("/usr/local/bin/mise", mise).
		WithEnvVariable("MISE_TRUSTED_CONFIG_PATHS", "/src/mise.toml").
		WithWorkdir("/src").
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
		}).
		With(func(r *dagger.Container) *dagger.Container {
			if install {
				include := []string{"mise.toml"}
				if !noEnv {
					include = append(include, m.Config.Env.Underscore.Source...)
					include = append(include, m.Config.Env.Underscore.Path...)
				}
				return r.WithMountedDirectory(".", m.Source.Filter(dagger.DirectoryFilterOpts{
					Include: include,
				})).
					WithExec(append([]string{"mise", "install"}, tools...))
			}
			return r
		}).
		WithMountedDirectory(".", m.Source), nil
}
