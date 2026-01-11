package main

import (
	"bytes"
	"context"
	"encoding/base64"
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/frantjc/daggerverse/release/internal/dagger"
)

type Release struct {
	// +private
	Source *dagger.GitRef
}

func New(src *dagger.GitRef) *Release {
	return &Release{Source: src}
}

var (
	//go:embed cask.rb.tpl
	caskRbTpl string
)

type tplOsArchData struct {
	URL string
	Sha256 string
}
type tplData struct {
	Name string
	Homepage string
	Description string
	Version string
	OsArch map[string]map[string]tplOsArchData
}

func (m *Release) Create(
	ctx context.Context,
	githubToken *dagger.Secret,
	githubRepo,
	pkg string,
	// +optional
	cgo bool,
	// +default=[
	// "linux",
	// "darwin"
	// ]
	goos []string,
	// +default=[
	// "amd64",
	// "arm64"
	// ]
	goarch []string,
	// +optional
	brew bool,
) error {
	tpl, err := template.New("cask").Parse(caskRbTpl)
	if err != nil {
		return err
	}

	data := new(tplData)
	data.Name = filepath.Base(pkg)
	data.Homepage = fmt.Sprintf("https://github.com/%s", githubRepo)

	gh := dag.Gh(githubToken)
	module := m.Source.Tree()

	ref, err := m.Source.Ref(ctx)
	if err != nil {
		return err
	}
	data.Version = strings.TrimPrefix(ref, "refs/tags/")

	data.Description, err = gh.Container().
		WithExec([]string{"gh", "repo", "view", githubRepo, "--json", "description", "--jq", ".description"}).
		Stdout(ctx)
	if err != nil {
		return err
	}

	assets := []*dagger.File{}
	checksumsTxt := new(strings.Builder)
	upx := dag.Upx()
	tar := dag.Tar()

	osPlatformVersion, err := dag.DefaultPlatform(ctx)
	if err != nil {
		return err
	}

	_, platform, ok := strings.Cut(string(osPlatformVersion), "/")
	if !ok {
		return fmt.Errorf("invalid dagger platform %s", osPlatformVersion)
	}

	version, err := dag.Version(ctx)
	if err != nil {
		return err
	}

	daggerTgz := dag.HTTP(
		fmt.Sprintf(
			"https://github.com/dagger/dagger/releases/download/%s/dagger_%s_linux_%s.tar.gz",
			version, version, platform,
		),
	)

	tmpDaggerTgzPath := "/tmp/dagger.tgz"
	tmpDaggerPath := "/tmp/dagger"

	wolfi := dag.Wolfi().
		Container()
	dagg3r := wolfi.
		WithEnvVariable("_EXPERIMENTAL_DAGGER_CLI_BIN", "/usr/bin/dagger").
		WithFile(
			"$_EXPERIMENTAL_DAGGER_CLI_BIN",
			wolfi.
				WithFile(tmpDaggerTgzPath, daggerTgz).
				WithExec([]string{
					"tar", "-xzf", tmpDaggerTgzPath, "-C", filepath.Dir(tmpDaggerPath), filepath.Base(tmpDaggerPath),
				}).
				File(tmpDaggerPath),
			dagger.ContainerWithFileOpts{Expand: true},
		).
		WithWorkdir("/src")

	for _, gs := range goos {
		for _, ga := range goarch {
			bin := dagg3r.
				WithDirectory(".", module).
				WithExec(
					[]string{"dagger", "call", "binary", "--version", data.Version, "--goos", gs, "--goarch", ga, "export", "--path", data.Name},
					dagger.ContainerWithExecOpts{ExperimentalPrivilegedNesting: true},
				).
				File(data.Name)

			if gs == "linux" {
				bin = upx.Pack(bin)
			}

			file := fmt.Sprintf("%s-%s-%s-%s.tar.gz", data.Name, data.Version, gs, ga)
			asset := tar.
				Create(
					module.Filter(dagger.DirectoryFilterOpts{
						Include: []string{
							"README.md",
							"LICENSE",
						},
					}).
						WithFile(
							data.Name,
							bin,
						),
					dagger.TarCreateOpts{
						Compress: true,
						Name: file,
					},
				)

			sha256sum, err := wolfi.WithFile("asset", asset).WithExec([]string{"sha256sum", "asset"}).Stdout(ctx)
			if err != nil {
				return err
			}

			checksum, _, _ := strings.Cut(sha256sum, "  ")

			if _, err = fmt.Fprintln(checksumsTxt, file, checksum); err != nil {
				return err
			}

			if data.OsArch == nil {
				data.OsArch = map[string]map[string]tplOsArchData{}
			}
			osArchData := tplOsArchData{
				URL: fmt.Sprintf("%s/releases/download/%s/%s", data.Homepage, data.Version, file),
				Sha256: strings.TrimPrefix(checksum, "sha256:"),
			}
			if _, ok := data.OsArch[gs]; ok {
				data.OsArch[gs][ga] = osArchData
			} else {
				data.OsArch[gs] = map[string]tplOsArchData{ga: osArchData}
			}

			assets = append(assets, asset)
		}
	}

	assets = append(assets, dag.File("checksums.txt", checksumsTxt.String()))

	if err := gh.
		Release().
		Create(ctx, data.Version, githubRepo, dagger.GhReleaseCreateOpts{Assets: assets}); err != nil {
		return err
	}

	if brew {
		buf := new(bytes.Buffer)
		enc := base64.NewEncoder(base64.StdEncoding, buf)

		if err := tpl.Execute(enc, data); err != nil {
			return err
		}

		if err = enc.Close(); err != nil {
			return err
		}

		owner, _, _ := strings.Cut(githubRepo, "/")

		if _, err := gh.Container().
			WithExec([]string{
				"gh",
				"api",
				fmt.Sprintf("repos/%s/homebrew-tap/contents/Casks/%s.rb", owner, data.Name),
				"-X=PUT",
				"-f", fmt.Sprintf(`message="chore: bump %s to %s"`, data.Name, data.Version),
				"-f", fmt.Sprintf("content=%s", buf.String()),
		}).
			Sync(ctx); err != nil {
			return err
		}
	}

	return nil
}
