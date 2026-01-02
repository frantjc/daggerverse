package main

import (
	"github.com/frantjc/daggerverse/gh/internal/dagger"
)

type Gh struct {
	githubToken *dagger.Secret
}

func New(githubToken *dagger.Secret) *Gh {
	return &Gh{githubToken: githubToken}
}

func (m *Gh) Release() *dagger.Release {
	return dag.Release(m.githubToken)
}
