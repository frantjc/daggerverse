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
	in := "/in"
	exec := []string{"tar", "-C", in}
	out := "/out.tar"
	if compress {
		out += ".gz"
		exec = append(exec, "-czf")
	} else {
		exec = append(exec, "-cf")
	}
	exec = append(exec, out, ".")
	return m.Container.
		WithDirectory(in, directory).
		WithExec(exec).
		File(out)
}
