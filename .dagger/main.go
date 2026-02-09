package main

import (
	"context"
	"github.com/frantjc/daggerverse/.dagger/internal/dagger"
)

type DaggerverseDev struct {
	Source *dagger.Directory
}

func New(
	// +optional
	// +defaultPath="."
	source *dagger.Directory,
) *DaggerverseDev {
	return &DaggerverseDev{
		Source: source,
	}
}

// +generate
func (m *DaggerverseDev) Bump(ctx context.Context) *dagger.Changeset {
	cs := dag.Directory().Changes(dag.Directory())

	for _, daggerModulePath := range []string{
		"compose",
		"dagger", "dagger-module",
		"dogger", "dogger/internal/dogger",
		"forge",
		"gh",
		"go",
		"release",
		"tar",
		"tls",
		"upx",
	} {
		cs = cs.WithChangeset(
			dag.DaggerModule(dagger.DaggerModuleOpts{
				Source: m.Source,
				Path: daggerModulePath,
			}).
				Bump(),
		)
	}
	
	return cs
}
