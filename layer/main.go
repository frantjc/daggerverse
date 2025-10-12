// A generated module for Layer functions

package main

import (
	"context"

	"github.com/frantjc/daggerverse/layer/internal/dagger"
	xslices "github.com/frantjc/x/slices"
)

type Layer struct{}


func (m *Layer) DirectoryOntoContainer(
	ctx context.Context,
	directory *dagger.Directory,
	container *dagger.Container,
	path string,
	// +optional
	includes [][]string,
	// +optional
	exclude []string,
	// +optional
	owner string,
	// +optional
	expand bool,
) (*dagger.Container, error) {
	for _, include := range includes {
		container = container.WithDirectory(path, directory, dagger.ContainerWithDirectoryOpts{
			Include: include,
			Owner: owner,
			Expand: expand,
			Exclude: exclude,
		})
	}
	
	return container.WithDirectory(path, directory, dagger.ContainerWithDirectoryOpts{
		Owner: owner,
		Expand: expand,
		Exclude: append(exclude, xslices.Flatten(includes...)...),
	}), nil
}

