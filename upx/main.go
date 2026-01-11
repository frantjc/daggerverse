package main

import (
	"context"
	"fmt"

	"github.com/frantjc/daggerverse/upx/internal/dagger"
)

type Upx struct {
	Container *dagger.Container
}

const (
	group = "upx"
	user  = group
	home  = "/home/" + user
)

func New() (*Upx, error) {
	return &Upx{
		Container: dag.Wolfi().
			Container(dagger.WolfiContainerOpts{
				Packages: []string{"upx"},
			}).
			WithEnvVariable("HOME", home).
			WithWorkdir("$HOME", dagger.ContainerWithWorkdirOpts{Expand: true}),
	}, nil
}

func (m *Upx) Pack(
	ctx context.Context,
	file *dagger.File,
	// +optional
	brute,
	// +optional
	lzma bool,
) (*dagger.File, error) {
	name, err := file.Name(ctx)
	if err != nil {
		return nil, err
	}
	in := fmt.Sprintf("$HOME/%s", name)
	exec := []string{"upx"}
	if brute {
		exec = append(exec, "--brute")
	}
	if lzma {
		exec = append(exec, "--lzma")
	} else {
		exec = append(exec, "--no-lzma")
	}
	exec = append(exec, in)
	return m.Container.
		WithFile(in, file, dagger.ContainerWithFileOpts{Expand: true}).
		WithExec(exec, dagger.ContainerWithExecOpts{Expand: true}).
		File(in, dagger.ContainerFileOpts{Expand: true}), nil
}
