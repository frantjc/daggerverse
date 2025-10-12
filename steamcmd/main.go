// A generated module for Steamcmd functions

package main

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/frantjc/daggerverse/steamcmd/internal/dagger"
	vdf "github.com/frantjc/go-encoding-vdf"
	"github.com/frantjc/go-steamcmd"
	xslices "github.com/frantjc/x/slices"
)

type Steamcmd struct{}

func (m *Steamcmd) Container() *dagger.Container {
	return dag.Container().
		From("steamcmd/steamcmd")
}

func (m *Steamcmd) AppInfoPrint(
	ctx context.Context,
	appID int,
) (string, error) {
	appInfoPrintArgs, err := steamcmd.Args(nil,
		steamcmd.Login{},
		steamcmd.AppInfoPrint(appID),
		steamcmd.Quit,
	)
	if err != nil {
		return "", err
	}

	steamcmdAppInfoPrintExec := append([]string{"steamcmd"}, appInfoPrintArgs...)
	cache := fmt.Sprint(time.Now().Unix())

	rawAppInfo, err := m.Container().
		WithEnvVariable("_SINDRI_CACHE", cache).
		WithExec(steamcmdAppInfoPrintExec).
		CombinedOutput(ctx)
	if err != nil {
		panic(err)
	}

	appInfoStartTokenIndex := strings.Index(rawAppInfo, "{")
	if appInfoStartTokenIndex == -1 {
		return "", fmt.Errorf("app_info_print did not output VDF")
	}

	appInfoEndTokenIndex := strings.LastIndex(rawAppInfo[appInfoStartTokenIndex:], "}")
	if appInfoEndTokenIndex == -1 {
		return "", fmt.Errorf("app_info_print did not output VDF")
	}

	return regexp.MustCompile(`\s+`).
		ReplaceAllString(
			rawAppInfo[appInfoStartTokenIndex:appInfoStartTokenIndex+appInfoEndTokenIndex],
			" ",
		), nil
}

type PlatformType = steamcmd.PlatformType

type AppUpdateOpts struct {
	// +default="linux"
	Branch string
	BetaPassword string
	PlatformType PlatformType
}

// TODO(frantjc): Split this up into multiple layers using depots (only when auth is passed: depots required auth).
func (m *Steamcmd) AppUpdate(
	ctx context.Context,
	appID int,
	// +optional
	opts *AppUpdateOpts,
) (*dagger.Directory, error) {
	rawAppInfo, err := m.AppInfoPrint(ctx, appID)
	if err != nil {
		return nil, err
	}

	appInfo := &steamcmd.AppInfo{}

	if err := vdf.NewDecoder(strings.NewReader(rawAppInfo)).Decode(appInfo); err != nil {
		return nil, err
	}

	steamappDirectoryPath := path.Join("/opt/sindri/steamapps", fmt.Sprint(appID))

	appUpdateArgs, err := steamcmd.Args(nil,
		steamcmd.ForceInstallDir(steamappDirectoryPath),
		steamcmd.Login{},
		steamcmd.ForcePlatformType(opts.PlatformType),
		steamcmd.AppUpdate{
			AppID:        appID,
			Beta:         opts.Branch,
			BetaPassword: opts.BetaPassword,
		},
		steamcmd.Quit,
	)
	if err != nil {
		return nil, err
	}

	cache := opts.Branch
	if depot, ok := appInfo.Depots.Branches[cache]; ok {
		cache = fmt.Sprint(depot.TimeUpdated)
	}

	steamcmdAppUpdateExec := append([]string{"steamcmd"}, appUpdateArgs...)

	return m.Container().
		WithEnvVariable("_SINDRI_CACHE", cache).
		WithExec(steamcmdAppUpdateExec).
		Directory(steamappDirectoryPath), nil
}

type AppUpdateOntoContainerOpts struct {
	AppUpdateOpts
	Includes [][]string
	Exclude []string
	Owner string
	Expand bool
}

func (m *Steamcmd) AppUpdateOntoContainer(
	ctx context.Context,
	container *dagger.Container,
	path string,
	appID int,
	// +optional
	opts *AppUpdateOntoContainerOpts,
) (*dagger.Container, error) {
	steamappDirectory, err := m.AppUpdate(ctx, appID, &opts.AppUpdateOpts)
	if err != nil {
		return nil, err
	}

	for _, include := range opts.Includes {
		container = container.WithDirectory(path, steamappDirectory, dagger.ContainerWithDirectoryOpts{
			Include: include,
			Owner: opts.Owner,
			Expand: opts.Expand,
			Exclude: opts.Exclude,
		})
	}
	
	return container.WithDirectory(path, steamappDirectory, dagger.ContainerWithDirectoryOpts{
		Owner: opts.Owner,
		Expand: opts.Expand,
		Exclude: append(opts.Exclude, xslices.Flatten(opts.Includes...)...),
	}), nil
}
