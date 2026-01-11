package main

import (
	"github.com/frantjc/daggerverse/tar/internal/dagger"
)

type Tar struct {
	Container *dagger.Container
}

const (
	group = "tar"
	user  = group
	home  = "/home/" + user
)

func New() (*Tar, error) {
	return &Tar{
		Container: dag.Wolfi().
			Container().
			WithEnvVariable("HOME", home).
			WithWorkdir("$HOME", dagger.ContainerWithWorkdirOpts{Expand: true}),
	}, nil
}

func (m *Tar) Create(
	directory *dagger.Directory,
) *dagger.File {
	in := "$HOME/in"
	out := "$HOME/out.tar.gz"
	return m.Container.
		WithDirectory(in, directory, dagger.ContainerWithDirectoryOpts{Expand: true}).
		WithExec([]string{"tar", "-C", in, "-xzf", out}, dagger.ContainerWithExecOpts{Expand: true}).
		File(out, dagger.ContainerFileOpts{Expand: true})
}
