package main

import (
	"context"
	"fmt"

	"github.com/frantjc/daggerverse/upx/internal/dagger"
)

type Upx struct {
	Container *dagger.Container
}

func New() (*Upx, error) {
	return &Upx{
		Container: dag.Wolfi().
			Container(dagger.WolfiContainerOpts{
				Packages: []string{"upx"},
			}),
	}, nil
}

func (m *Upx) Pack(
	ctx context.Context,
	executable *dagger.File,
	// +optional
	brute,
	// +optional
	lzma bool,
) (*dagger.File, error) {
	name, err := executable.Name(ctx)
	if err != nil {
		return nil, err
	}
	in := fmt.Sprintf("%s", name)
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
		WithFile(in, executable).
		WithExec(exec).
		File(in), nil
}
