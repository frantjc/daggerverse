// Build and publish games with the Steamworks tooling.
package main

import (
	"bytes"
	"context"
	_ "embed"
	"path"
	"strconv"
	"text/template"

	"github.com/frantjc/daggerverse/steamworks/internal/dagger"
)

const (
	contentRoot = "/content"
	buildOutput = "/output"
	vdfPath     = "/tmp/app_build.vdf"

	drmFlagCompatibility     = 6
	drmFlagSkipDebuggerCheck = 32
)

var (
	//go:embed app_build.vdf.tpl
	appBuildVdfTplContents string

	appBuildVdfTpl = template.Must(template.New("app_build.vdf").Parse(appBuildVdfTplContents))
)

// Steamworks is a Dagger module for Steamworks tooling.
type Steamworks struct {
	// SDK is the Steamworks SDK source.
	SDK *dagger.Directory
}

// New creates a Steamworks module.
func New(
	// The Steamworks SDK source. Defaults to the head of rlabrecque/SteamworksSDK.
	// +optional
	sdk *dagger.Directory,
) *Steamworks {
	if sdk == nil {
		sdk = dag.Git("https://github.com/rlabrecque/SteamworksSDK").Head().Tree()
	}

	return &Steamworks{
		SDK: sdk,
	}
}

// Steamcmd is steamcmd, logged in as a Steam account if a password is given.
type Steamcmd struct {
	// +private
	Container *dagger.Container
	// +private
	Username string
	// +private
	Password *dagger.Secret
	// +private
	SteamGuardCode *dagger.Secret
}

// Steamcmd returns steamcmd, logged in as a Steam account if a password is given.
func (m *Steamworks) Steamcmd(
	// The steamcmd container.
	// +optional
	// +defaultAddress="docker.io/steamcmd/steamcmd"
	container *dagger.Container,
	// The Steam account to log in as.
	// +optional
	// +default=anonymous
	username string,
	// The password of the Steam account. Without it, steamcmd logs in without one,
	// which is only enough for commands that don't need a real account.
	// +optional
	password *dagger.Secret,
	// The Steam Guard code of the Steam account, if it has one. Ignored without a password.
	// +optional
	steamGuardCode *dagger.Secret,
) *Steamcmd {
	return &Steamcmd{
		Container:      container,
		Username:       username,
		Password:       password,
		SteamGuardCode: steamGuardCode,
	}
}

// login returns the container with the credentials set, if any, and the
// steamcmd arguments, to be Expand-ed, that log in with them.
func (s *Steamcmd) login(container *dagger.Container) (*dagger.Container, []string) {
	exec := []string{"steamcmd", "+login", s.Username}

	if s.Password != nil {
		exec = append(exec, "$STEAMCMD_PASSWORD")
		container = container.WithSecretVariable("STEAMCMD_PASSWORD", s.Password)

		if s.SteamGuardCode != nil {
			exec = append(exec, "$STEAMCMD_GUARD_CODE")
			container = container.WithSecretVariable("STEAMCMD_GUARD_CODE", s.SteamGuardCode)
		}
	}

	return container, exec
}

// Depot maps files from the build content into a Steam depot.
type Depot struct {
	// DepotID is the depot ID.
	DepotID int
	// Path is a glob, relative to the content directory, of the files to include.
	// Defaults to every file.
	Path string
	// Recursive includes matching files in subdirectories.
	Recursive bool
}

// AppBuild uploads content to Steam using steamcmd's run_app_build,
// returning the build output (logs and cache files).
func (s *Steamcmd) AppBuild(
	// The ID of the app to build.
	appID int,
	// The directory of files to upload.
	content *dagger.Directory,
	// The depots to build and the files that belong in each.
	depots []Depot,
	// A description of the build, visible only to the developer.
	// +optional
	desc string,
) (*dagger.Directory, error) {
	for i := range depots {
		if depots[i].Path == "" {
			depots[i].Path = "*"
		}
	}

	buf := new(bytes.Buffer)
	if err := appBuildVdfTpl.Execute(buf, map[string]any{
		"AppID":       appID,
		"Desc":        desc,
		"ContentRoot": contentRoot,
		"BuildOutput": buildOutput,
		"Depots":      depots,
	}); err != nil {
		return nil, err
	}

	container, exec := s.login(
		s.Container.
			WithNewFile(vdfPath, buf.String()).
			WithMountedDirectory(contentRoot, content).
			WithDirectory(buildOutput, dag.Directory()),
	)

	return container.
		WithExec(
			append(exec, "+run_app_build", vdfPath, "+quit"),
			dagger.ContainerWithExecOpts{Expand: true},
		).
		Directory(buildOutput), nil
}

// DRMWrap wraps a Windows executable with Steam DRM using steamcmd's drm_wrap
// and the drmtoolp tool, returning the wrapped executable.
func (s *Steamcmd) DRMWrap(
	ctx context.Context,
	// The ID of the app the executable belongs to.
	appID int,
	// The plaintext executable to wrap. .NET applications are not supported.
	executable *dagger.File,
	// Disable obfuscation. Apply this before layering other DRM solutions.
	// +optional
	compatibility bool,
	// Skip debugger checks.
	// +optional
	skipDebuggerCheck bool,
) (*dagger.File, error) {
	name, err := executable.Name(ctx)
	if err != nil {
		return nil, err
	}

	var (
		in    = path.Join("/in", name)
		out   = path.Join("/out", name)
		flags = 0
	)
	if compatibility {
		flags |= drmFlagCompatibility
	}
	if skipDebuggerCheck {
		flags |= drmFlagSkipDebuggerCheck
	}

	container, exec := s.login(
		s.Container.
			WithMountedFile(in, executable).
			WithDirectory(path.Dir(out), dag.Directory()),
	)

	return container.
		WithExec(
			append(exec, "+drm_wrap", strconv.Itoa(appID), in, out, "drmtoolp", strconv.Itoa(flags), "+quit"),
			dagger.ContainerWithExecOpts{Expand: true},
		).
		File(out), nil
}
