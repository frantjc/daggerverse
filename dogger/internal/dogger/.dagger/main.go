// A generated module for DoggerDev functions

package main

import (
	"context"
	"github.com/frantjc/daggerverse/dogger/internal/dogger/.dagger/internal/dagger"
)

type DoggerDev struct {
	Source *dagger.Directory
}

func New(
	// +optional
	// +defaultPath="."
	src *dagger.Directory,
) (*DoggerDev, error) {
	return &DoggerDev{
		Source: src,
	}, nil
}

func (m *DoggerDev) Binary(ctx context.Context) *dagger.File {
	return dag.Go(dagger.GoOpts{
		Module: m.Source,
		AdditionalWolfiPackages: []string{"gcc"},
	}).
		Build(dagger.GoBuildOpts{
			Pkg:     "./cmd/dogger",
			Cgo: true,
		})
}
