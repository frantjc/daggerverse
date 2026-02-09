package main

import (
	"context"
	"fmt"

	"github.com/frantjc/daggerverse/tar/internal/dagger"
)

type Tar struct {
	// +private
	Container *dagger.Container
}

func New() (*Tar, error) {
	return &Tar{
		Container: dag.Wolfi().Container(),
	}, nil
}

func (m *Tar) Create(
	ctx context.Context,
	directory *dagger.Directory,
	// +optional
	compress bool,
) (*dagger.File, error) {
	name, err := directory.Name(ctx)
	if err != nil {
		return nil, err
	}
	in := fmt.Sprintf("/%s", name)
	exec := []string{"tar", "-C", in}
	out := fmt.Sprintf("/%s.tar", name)
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
		File(out), nil
}
