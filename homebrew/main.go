package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"

	"github.com/frantjc/daggerverse/mise/internal/dagger"
)

type Homebrew struct{}

var (
	//go:embed cask.rb.tpl
	caskRbTpl string
)

type tplOsArchData struct {
	URL    string
	Sha256 string
}

type tplData struct {
	Name        string
	Homepage    string
	Description string
	Version     string
	OsArch      map[string]map[string]tplOsArchData
}

// Cask renders a Homebrew cask formula for the given GitHub release and
// uploads it to the org's homebrew-tap repo, at Casks/<name>.rb, creating or
// updating the file as needed. Name, homepage and description are looked up
// from the repo.
func (m *Homebrew) Cask(
	ctx context.Context,
	gh *dagger.Gh,
	repo,
	tag string,
) error {
	tpl, err := template.New("cask").Parse(caskRbTpl)
	if err != nil {
		return err
	}

	org, name, ok := strings.Cut(repo, "/")
	if !ok {
		return fmt.Errorf("expected org/repo format, got %q", repo)
	}

	description, err := gh.Container().
		WithExec([]string{"gh", "repo", "view", repo, "--json", "description", "--jq", ".description"}).
		Stdout(ctx)
	if err != nil {
		return err
	}

	homepage := fmt.Sprintf("https://github.com/%s", repo)
	version := strings.TrimPrefix(tag, "v")

	data := &tplData{
		Name:        name,
		Homepage:    homepage,
		Description: strings.TrimSpace(description),
		Version:     version,
		OsArch:      map[string]map[string]tplOsArchData{},
	}

	assets, err := gh.Release(repo, tag).Assets(ctx)
	if err != nil {
		return err
	}

	for i := range assets {
		asset := assets[i]

		file := asset.File()

		fileName, err := file.Name(ctx)
		if err != nil {
			return err
		}

		goos, goarch, ok := parseAssetName(fileName, name, version)
		if !ok {
			continue
		}

		digest, err := asset.Digest(ctx)
		if err != nil {
			return err
		}

		os := "linux"
		if goos == "darwin" {
			os = "macos"
		}

		arch := "intel"
		if goarch == "arm64" {
			arch = "arm"
		}

		if _, ok := data.OsArch[os]; !ok {
			data.OsArch[os] = map[string]tplOsArchData{}
		}

		data.OsArch[os][arch] = tplOsArchData{
			URL:    fmt.Sprintf("%s/releases/download/%s/%s", homepage, version, fileName),
			Sha256: strings.TrimPrefix(digest, "sha256:"),
		}
	}

	buf := new(bytes.Buffer)
	enc := base64.NewEncoder(base64.StdEncoding, buf)

	if err := tpl.Execute(enc, data); err != nil {
		return err
	}

	if err := enc.Close(); err != nil {
		return err
	}

	endpoint := fmt.Sprintf("repos/%s/homebrew-tap/contents/Casks/%s.rb", org, name)
	upload := []string{
		"gh",
		"api",
		"-X=PUT",
		endpoint,
		"-f", fmt.Sprintf("message=chore: bump %s to %s", name, version),
		"-f", fmt.Sprintf("content=%s", buf.String()),
	}

	if sha, err := gh.Container().
		WithExec([]string{
			"gh",
			"api",
			endpoint,
			"--jq",
			".sha",
		}).
		Stdout(ctx); err == nil {
		upload = append(upload, "-f", fmt.Sprintf("sha=%s", strings.TrimSpace(sha)))
	}

	_, err = gh.Container().
		WithExec(upload).
		Sync(ctx)

	return err
}

// parseAssetName parses a release asset file name of the form
// <name>-<version>-<goos>-<goarch>.tar.gz into its goos and goarch parts.
func parseAssetName(fileName, name, version string) (goos, goarch string, ok bool) {
	prefix := fmt.Sprintf("%s-%s-", name, version)
	if !strings.HasPrefix(fileName, prefix) {
		return "", "", false
	}

	rest := strings.TrimSuffix(strings.TrimPrefix(fileName, prefix), ".tar.gz")

	goos, goarch, ok = strings.Cut(rest, "-")
	if !ok {
		return "", "", false
	}

	return goos, goarch, true
}
