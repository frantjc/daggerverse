package main

import (
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
	file *dagger.File,
	// +optional
	ultraBrute bool,
) *dagger.File {
	in := "$HOME/in"
	arg := "--brute"
	if ultraBrute {
		arg = "--ultra-brute"
	}
	return m.Container.
		WithFile(in, file, dagger.ContainerWithFileOpts{Expand: true}).
		WithExec([]string{"upx", arg, in}, dagger.ContainerWithExecOpts{Expand: true}).
		File(in, dagger.ContainerFileOpts{Expand: true})
}
