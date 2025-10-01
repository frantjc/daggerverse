package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"github.com/frantjc/daggerverse/compose/internal/dagger"
)

type Compose struct {
	Source *dagger.Directory
	Files  []string
	Env    []EnvVar
}

type EnvVar struct {
	Name  string
	Value string
}

func New(
	// +optional
	// +defaultPath="."
	source *dagger.Directory,
	// +optional
	// +default=["docker-compose.yml"]
	files []string,
) *Compose {
	return &Compose{
		Source: source,
		Files:  files,
	}
}

func (m *Compose) WithEnv(name, val string) *Compose {
	m.Env = append(m.Env, EnvVar{
		Name:  name,
		Value: val,
	})
	return m
}

func (m *Compose) Up(
	ctx context.Context,
	// +optional
	services ...string,
) (*dagger.Service, error) {
	env := make(types.Mapping)
	for _, e := range m.Env {
		env[e.Name] = e.Value
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if _, err = m.Source.Export(ctx, wd); err != nil {
		return nil, err
	}

	loaderConfig := types.ConfigDetails{
		Version:     "3",
		WorkingDir:  wd,
		Environment: env,
	}

	for _, f := range m.Files {
		content, err := m.Source.File(f).Contents(ctx)
		if err != nil {
			return nil, err
		}

		loaderConfig.ConfigFiles = append(loaderConfig.ConfigFiles, types.ConfigFile{
			Filename: filepath.Base(f),
			Content:  []byte(content),
		})
	}

	project, err := loader.LoadWithContext(
		ctx,
		loaderConfig,
		func(options *loader.Options) {
			options.SetProjectName("dagger-compose", true)
		},
	)
	if err != nil {
		return nil, err
	}

	proxy := dag.Proxy()

	for _, composeSvc := range project.Services {
		if len(services) > 0 && !slices.Contains(services, composeSvc.Name) {
			continue
		}

		svc, err := m.convert(project, composeSvc)
		if err != nil {
			return nil, err
		}

		for _, port := range composeSvc.Ports {
			frontend, err := strconv.Atoi(port.Published)
			if err != nil {
				return nil, err
			}

			switch port.Mode {
			case "ingress":
				proxy = proxy.WithService(
					svc,
					composeSvc.Name,
					frontend,
					int(port.Target),
				)
			default:
				return nil, fmt.Errorf("port mode %s not supported", port.Mode)
			}
		}
	}

	return proxy.Service(), nil
}

func (m *Compose) convert(project *types.Project, svc types.ServiceConfig) (*dagger.Service, error) {
	container := dag.Container()

	if svc.Image != "" {
		container = container.From(svc.Image)
	} else if svc.Build != nil {
		args := []dagger.BuildArg{}
		for name, val := range svc.Build.Args {
			if val != nil {
				args = append(args, dagger.BuildArg{
					Name:  name,
					Value: *val,
				})
			}
		}

		container = m.Source.Directory(svc.Build.Context).DockerBuild(dagger.DirectoryDockerBuildOpts{
			Dockerfile: svc.Build.Dockerfile,
			BuildArgs:  args,
			Target:     svc.Build.Target,
		})
	}

	// sort env to ensure same container
	envs := []EnvVar{}
	for name, val := range svc.Environment {
		if val != nil {
			envs = append(envs, EnvVar{name, *val})
		}
	}
	sort.Slice(envs, func(i, j int) bool {
		return envs[i].Name < envs[j].Name
	})
	for _, env := range envs {
		container = container.WithEnvVariable(env.Name, env.Value)
	}

	for _, port := range svc.Ports {
		switch port.Mode {
		case "ingress":
			protocol := dagger.NetworkProtocolTcp
			switch port.Protocol {
			case "udp":
				protocol = dagger.NetworkProtocolUdp
			case "", "tcp":
				protocol = dagger.NetworkProtocolTcp
			default:
				return nil, fmt.Errorf("protocol %s not supported", port.Protocol)
			}

			container = container.WithExposedPort(int(port.Target), dagger.ContainerWithExposedPortOpts{
				Protocol: protocol,
			})
		default:
			return nil, fmt.Errorf("port mode %s not supported", port.Mode)
		}
	}

	for _, expose := range svc.Expose {
		port, err := strconv.Atoi(expose)
		if err != nil {
			return nil, err
		}

		container = container.WithExposedPort(port)
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	for _, vol := range svc.Volumes {
		switch vol.Type {
		case types.VolumeTypeBind:
			source, err := filepath.Rel(wd, vol.Source)
			if err != nil {
				return nil, err
			}

			container = container.WithMountedDirectory(vol.Target, m.Source.Directory(source))
		case types.VolumeTypeVolume:
			container = container.WithMountedCache(vol.Target, dag.CacheVolume(vol.Source))
		default:
			return nil, fmt.Errorf("volume type %s not supported", vol.Type)
		}
	}

	for depName := range svc.DependsOn {
		cfg, err := project.GetService(depName)
		if err != nil {
			return nil, err
		}

		svc, err := m.convert(project, cfg)
		if err != nil {
			return nil, err
		}

		container = container.WithServiceBinding(depName, svc)
	}

	if !svc.Entrypoint.IsZero() {
		container = container.WithEntrypoint(svc.Entrypoint)
	}

	return container.AsService(dagger.ContainerAsServiceOpts{
		Args:                     svc.Command,
		UseEntrypoint:            true,
		InsecureRootCapabilities: svc.Privileged,
	}), nil
}
