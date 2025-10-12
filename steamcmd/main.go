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

// TODO(frantjc): Split this up into multiple layers using depots (only when auth is passed: depots required auth).
func (m *Steamcmd) AppUpdate(
	ctx context.Context,
	appID int,
	// +optional
	// +default="linux"
	branch string,
	// +optional
	betaPassword string,
	// +optional
	platformType PlatformType,
) (*dagger.Directory, error) {
	rawAppInfo, err := m.AppInfoPrint(ctx, appID)
	if err != nil {
		return nil, err
	}

	appInfo := &steamcmd.AppInfo{}

	if err := vdf.NewDecoder(strings.NewReader(rawAppInfo)).Decode(appInfo); err != nil {
		return nil, err
	}

	steamappDirectoryPath := path.Join("/tmp", fmt.Sprint(appID))

	appUpdateArgs, err := steamcmd.Args(nil,
		steamcmd.ForceInstallDir(steamappDirectoryPath),
		steamcmd.Login{},
		steamcmd.ForcePlatformType(platformType),
		steamcmd.AppUpdate{
			AppID:        appID,
			Beta:         branch,
			BetaPassword: betaPassword,
		},
		steamcmd.Quit,
	)
	if err != nil {
		return nil, err
	}

	cache := branch
	if depot, ok := appInfo.Depots.Branches[branch]; ok {
		cache = fmt.Sprint(depot.TimeUpdated)
	}

	steamcmdAppUpdateExec := append([]string{"steamcmd"}, appUpdateArgs...)

	return m.Container().
		WithEnvVariable("_SINDRI_CACHE", cache).
		WithExec(steamcmdAppUpdateExec).
		Directory(steamappDirectoryPath), nil
}

func (m *Steamcmd) AppUpdateOntoContainer(
	ctx context.Context,
	container *dagger.Container,
	path string,
	appID int,
	// +optional
	// +default="linux"
	branch string,
	// +optional
	betaPassword string,
	// +optional
	platformType PlatformType,
	// +optional
	includes [][]string,
	// +optional
	exclude []string,
	// +optional
	owner string,
	// +optional
	expand bool,
) (*dagger.Container, error) {
	steamappDirectory, err := m.AppUpdate(ctx, appID, branch, betaPassword, platformType)
	if err != nil {
		return nil, err
	}

	for _, include := range includes {
		container = container.WithDirectory(path, steamappDirectory, dagger.ContainerWithDirectoryOpts{
			Include: include,
			Owner: owner,
			Expand: expand,
			Exclude: exclude,
		})
	}
	
	return container.WithDirectory(path, steamappDirectory, dagger.ContainerWithDirectoryOpts{
		Owner: owner,
		Expand: expand,
		Exclude: append(exclude, xslices.Flatten(includes...)...),
	}), nil
}
