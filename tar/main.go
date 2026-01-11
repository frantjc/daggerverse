package main

import (
	"github.com/frantjc/daggerverse/tar/internal/dagger"
)

type Tar struct {
	Container *dagger.Container
}

func New() (*Tar, error) {
	return &Tar{
		Container: dag.Wolfi().Container(),
	}, nil
}

func (m *Tar) Create(
	directory *dagger.Directory,
	// +optional
	compress bool,
) *dagger.File {
	in := "$HOME/in"
	exec := []string{"tar", "-C", in}
	out := "$HOME/out.tar"
	if compress {
		out += ".gz"
		exec = append(exec, "-czf")
	} else {
		exec = append(exec, "-cf")
	}
	exec = append(exec, out, ".")
	return m.Container.
		WithDirectory(in, directory, dagger.ContainerWithDirectoryOpts{Expand: true}).
		WithExec(exec, dagger.ContainerWithExecOpts{Expand: true}).
		File(out, dagger.ContainerFileOpts{Expand: true})
}
