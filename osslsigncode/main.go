package main

import (
	"context"
	"path/filepath"

	"github.com/logsquaredn/rubber/.dagger/modules/osslsigncode/internal/dagger"
)

type Osslsigncode struct {
	Binary *dagger.File
}

func New(
	ctx context.Context,
	// +optional
	// +default="2.13"
	version string,
) (*Osslsigncode, error) {
	src := dag.Git("https://github.com/mtrojnar/osslsigncode").Ref(version).Tree()
	srcPath := "/src"
	buildPath := filepath.Join(srcPath, "build")

	return &Osslsigncode{
		Binary: dag.Wolfi().
			Container(dagger.WolfiContainerOpts{
				Packages: []string{
					"build-base",
					"cmake",
					"curl-dev",
					"openssl-dev",
					"pkgconf",
					"python3",
					"zlib-dev",
				},
			}).
			WithMountedDirectory(srcPath, src).
			WithWorkdir(buildPath).
			WithExec([]string{"cmake", "-S", "..", "-DCMAKE_INSTALL_PREFIX=/usr/local", "-DCMAKE_BUILD_TYPE=Release"}).
			WithExec([]string{"cmake", "--build", "."}).
			WithExec([]string{"cmake", "--install", "."}).
			File("/usr/local/bin/osslsigncode"),
	}, nil
}
