package main

import (
	"context"
	"github.com/frantjc/daggerverse/.dagger/internal/dagger"
)

type DaggerverseDev struct {
	Source *dagger.Directory
}

func New(
	// +defaultPath="."
	source *dagger.Directory,
) *DaggerverseDev {
	return &DaggerverseDev{
		Source: source,
	}
}

// +generate
func (m *DaggerverseDev) Bump(ctx context.Context) *dagger.Changeset {
	src := m.Source

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
		".",
	} {
		src = src.WithChanges(
			dag.DaggerModule(m.Source, dagger.DaggerModuleOpts{
				Path: daggerModulePath,
			}).
				Bump(),
		)
	}
	
	return src.Changes(m.Source)
}
