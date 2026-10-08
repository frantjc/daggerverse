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

// withSteamcmd returns the container with the credentials set, if any, and
// the exec that runs steamcmd, logged in with them, followed by args.
// Secret variables can't be Expand-ed, so a shell does the substitution instead.
func (s *Steamcmd) withSteamcmd(container *dagger.Container, args ...string) (*dagger.Container, []string) {
	script := `exec steamcmd +login "$STEAMCMD_USERNAME"`
	container = container.WithEnvVariable("STEAMCMD_USERNAME", s.Username)

	if s.Password != nil {
		script += ` "$STEAMCMD_PASSWORD"`
		container = container.WithSecretVariable("STEAMCMD_PASSWORD", s.Password)

		if s.SteamGuardCode != nil {
			script += ` "$STEAMCMD_GUARD_CODE"`
			container = container.WithSecretVariable("STEAMCMD_GUARD_CODE", s.SteamGuardCode)
		}
	}

	return container, append([]string{"sh", "-c", script + ` "$@"`, "sh"}, args...)
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

// Depot creates a Steam depot that maps files from the build content.
func (m *Steamworks) Depot(
	// The depot ID.
	depotID int,
	// A glob, relative to the content directory, of the files to include.
	// Defaults to every file.
	// +optional
	path string,
	// Include matching files in subdirectories.
	// +optional
	recursive bool,
) *Depot {
	return &Depot{
		DepotID:   depotID,
		Path:      path,
		Recursive: recursive,
	}
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

	container, exec := s.withSteamcmd(
		s.Container.
			WithNewFile(vdfPath, buf.String()).
			WithMountedDirectory(contentRoot, content).
			WithDirectory(buildOutput, dag.Directory()),
		"+run_app_build", vdfPath, "+quit",
	)

	return container.WithExec(exec).Directory(buildOutput), nil
}

// DrmWrap wraps a Windows executable with Steam DRM using steamcmd's drm_wrap
// and the drmtoolp tool, returning the wrapped executable.
func (s *Steamcmd) DrmWrap(
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

	container, exec := s.withSteamcmd(
		s.Container.
			WithMountedFile(in, executable).
			WithDirectory(path.Dir(out), dag.Directory()),
		"+drm_wrap", strconv.Itoa(appID), in, out, "drmtoolp", strconv.Itoa(flags), "+quit",
	)

	return container.WithExec(exec).File(out), nil
}
