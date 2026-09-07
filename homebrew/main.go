package main

import (
	"bytes"
	"cmp"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
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
	githubToken *dagger.Secret,
	githubRepo,
	tag string,
	// +optional
	container *dagger.Container,
) error {
	gh := dag.Gh(githubToken, dagger.GhOpts{
		Container: container,
	})

	tpl, err := template.New("cask").Parse(caskRbTpl)
	if err != nil {
		return err
	}

	org, repo, ok := strings.Cut(githubRepo, "/")
	if !ok {
		return fmt.Errorf("expected org/repo format, got %q", githubRepo)
	}

	viewContents, err := gh.Container().
		WithExec([]string{"gh", "repo", "view", githubRepo, "--json", "description,homepageUrl"}).
		Stdout(ctx)
	if err != nil {
		return err
	}

	view := struct {
		Description string `json:"description"`
		Homepage    string `json:"homepageUrl"`
	}{}

	if err := json.Unmarshal([]byte(viewContents), &view); err != nil {
		return err
	}

	version := strings.TrimPrefix(tag, "v")

	data := &tplData{
		Name:        repo,
		Homepage:    cmp.Or(view.Homepage, fmt.Sprintf("https://github.com/%s", githubRepo)),
		Description: view.Description,
		Version:     version,
		OsArch:      map[string]map[string]tplOsArchData{},
	}

	assets, err := gh.Release(githubRepo, tag).Assets(ctx)
	if err != nil {
		return err
	}

	for i := range assets {
		asset := assets[i]

		name, err := asset.Name(ctx)
		if err != nil {
			return err
		}

		os, arch, ok := parseOsArch(name)
		if !ok {
			continue
		}

		if _, ok := data.OsArch[os]; !ok {
			data.OsArch[os] = map[string]tplOsArchData{}
		}

		url, err := asset.URL(ctx)
		if err != nil {
			return err
		}

		digest, err := asset.Digest(ctx)
		if err != nil {
			return err
		}

		data.OsArch[os][arch] = tplOsArchData{
			URL:    url,
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

	endpoint := fmt.Sprintf("repos/%s/homebrew-tap/contents/Casks/%s.rb", org, repo)
	upload := []string{
		"gh",
		"api",
		"-X=PUT",
		endpoint,
		"-f", fmt.Sprintf("message=chore: bump %s to %s", repo, version),
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

var (
	// assetNameRegexp matches a goos/goarch pair anywhere in a release asset
	// file name with a .tar.gz or .tgz extension, e.g.
	// "barge-v0.3.3-darwin-amd64.tar.gz". It doesn't anchor on the
	// name/version prefix so that it's agnostic to tag/version formatting.
	assetNameRegexp = regexp.MustCompile(
		`(?:^|[-_])(?P<goos>darwin|linux|windows)[-_](?P<goarch>386|amd64|x86_64|arm64|arm)(?:[-_].*)?\.(?:tar\.gz|tgz)$`,
	)
)

// parseOsArch parses a release asset file name for its goos/goarch,
// returning them as Homebrew-compatible os/arch strings, e.g. "macos"/"arm".
func parseOsArch(assetName string) (os, arch string, ok bool) {
	m := assetNameRegexp.FindStringSubmatch(assetName)
	if m == nil {
		return "", "", false
	}

	os = m[assetNameRegexp.SubexpIndex("goos")]
	switch os {
	case "darwin":
		os = "macos"
	}

	arch = m[assetNameRegexp.SubexpIndex("goarch")]
	switch arch {
	case "amd64", "x86_64":
		arch = "intel"
	case "arm64":
		arch = "arm"
	}

	return os, arch, true
}
