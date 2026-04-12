package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/logsquaredn/rubber/.dagger/modules/archive/internal/dagger"
)

type Archive struct{}

func (m *Archive) Tar(
	ctx context.Context,
	directory *dagger.Directory,
	// +optional
	gzip bool,
) (*dagger.File, error) {
	name, err := directory.Name(ctx)
	if err != nil {
		return nil, err
	} else if name == "" {
		name = "input"
	}
	in := fmt.Sprintf("/tmp/%s", name)
	out := fmt.Sprintf("/tmp/%s.tar", name)
	flags := "cf"
	if gzip {
		flags = "czf"
		out = fmt.Sprintf("/tmp/%s.tgz", name)
	}
	return dag.Wolfi().
		Container().
		WithDirectory(in, directory).
		WithExec([]string{"tar", "-C", in, flags, out, "."}).
		File(out), nil
}

func (m *Archive) Untar(
	ctx context.Context,
	file *dagger.File,
) (*dagger.Directory, error) {
	name, err := file.Name(ctx)
	if err != nil {
		return nil, err
	} else if name == "" {
		name = "input.tar"
	}
	flags := "xf"
	switch {
	case strings.HasSuffix(name, "gz"):
		flags = "xzf"
	case strings.HasSuffix(name, "xz"):
		flags = "xJf"
	}
	base := strings.TrimSuffix(name, ".tgz")
	base = strings.TrimSuffix(name, ".tar.gz")
	base = strings.TrimSuffix(name, ".tar.xz")
	if base == "" {
		base = "output"
	}
	in := fmt.Sprintf("/tmp/%s", name)
	out := fmt.Sprintf("/tmp/%s", base)
	return dag.Wolfi().
		Container().
		WithFile(in, file).
		WithWorkdir(out).
		WithExec([]string{"tar", flags, in}).
		Directory(out), nil
}

func (m *Archive) Zip(
	ctx context.Context,
	directory *dagger.Directory,
) (*dagger.File, error) {
	name, err := directory.Name(ctx)
	if err != nil {
		return nil, err
	} else if name == "" {
		name = "input"
	}
	in := fmt.Sprintf("/tmp/%s", name)
	out := fmt.Sprintf("/tmp/%s.zip", name)
	return dag.Wolfi().
		Container(dagger.WolfiContainerOpts{
			Packages: []string{"zip"},
		}).
		WithDirectory(in, directory).
		WithWorkdir(in).
		WithExec([]string{"zip", "-r", "-q", out, "."}).
		File(out), nil
}
