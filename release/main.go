package main

import (
	"bytes"
	"context"
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

const (
	gid            = "1001"
	uid            = gid
	group          = "release"
	user           = group
	owner          = user + ":" + group
	home           = "/home/" + user
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
		Container().
		WithExec([]string{"addgroup", "-S", "-g", gid, group}).
		WithExec([]string{"adduser", "-S", "-G", group, "-u", uid, user}).
		WithEnvVariable("HOME", home)
	dagg3r := wolfi.
		WithWorkdir("$HOME", dagger.ContainerWithWorkdirOpts{Expand: true}).
		WithEnvVariable("PATH", home+"/.local/bin:$PATH", dagger.ContainerWithEnvVariableOpts{Expand: true}).
		WithEnvVariable("_EXPERIMENTAL_DAGGER_CLI_BIN", home+"/.local/bin/dagger").
		WithFile(
			"$_EXPERIMENTAL_DAGGER_CLI_BIN",
			wolfi.
				WithFile(tmpDaggerTgzPath, daggerTgz).
				WithExec([]string{
					"tar", "-xzf", tmpDaggerTgzPath, "-C", filepath.Dir(tmpDaggerPath), filepath.Base(tmpDaggerPath),
				}).
				File(tmpDaggerPath),
			dagger.ContainerWithFileOpts{Expand: true, Owner: owner, Permissions: 0700},
		)

	for _, gs := range goos {
		for _, ga := range goarch {
			bin := upx.Pack(
				dagg3r.
					WithDirectory(".", module).
					WithExec(
						[]string{"dagger", "call", "binary", "--version", data.Version, "--goos", gs, "--goarch", ga, "export", "--path", data.Name},
						dagger.ContainerWithExecOpts{ExperimentalPrivilegedNesting: true},
					).
					File(data.Name),
			)

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
							dagger.DirectoryWithFileOpts{
								Permissions: 0755,
							},
						),
				).
				WithName(file)

			checksum, err := asset.Digest(ctx)
			if err != nil {
				return err
			}

			if data.OsArch == nil {
				data.OsArch = map[string]map[string]tplOsArchData{}
			}
		
			osArchData := tplOsArchData{
				URL: fmt.Sprintf("%s/releases/download/%s/%s", data.Homepage, data.Version, file),
				Sha256: checksum,
			}
			if _, ok := data.OsArch[gs]; ok {
				data.OsArch[gs][ga] = osArchData
			} else {
				data.OsArch[gs] = map[string]tplOsArchData{ga: osArchData}
			}

			if _, err = fmt.Fprintln(checksumsTxt, file, checksum); err != nil {
				return err
			}

			assets = append(assets, asset)
		}
	}

	assets = append(assets, dag.File("checksums.txt", checksumsTxt.String()))

	buf := new(bytes.Buffer)

	if err := tpl.Execute(buf, data); err != nil {
		return err
	}

	if err := gh.
		Release().
		Create(ctx, data.Version, githubRepo, dagger.GhReleaseCreateOpts{Assets: assets}); err != nil {
		return err
	}

	owner, _, _ := strings.Cut(githubRepo, "/")

	cask := fmt.Sprintf("%s.rb", data.Name)
	if _, err := gh.Container().
		WithFile(
			cask,
			dag.File(cask, buf.String()),
		).
		WithExec([]string{
			"gh",
			"api",
			fmt.Sprintf("repos/%s/homebrew-tap/contents/Casks/%s", owner, cask),
			"-X=PUT",
  			"-f", fmt.Sprintf(`message="chore: bump %s to %s"`, data.Name, data.Version),
			"-f", fmt.Sprintf("content=@%s", cask),
  }).
		Sync(ctx); err != nil {
		return err
	}

	return nil
}
