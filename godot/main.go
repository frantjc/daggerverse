package main

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/logsquaredn/rubber/.dagger/modules/godot/internal/dagger"
	"gopkg.in/ini.v1"
)

const (
	workDir = "/tmp"
)

type Godot struct {
	Container       *dagger.Container
	ExportTemplates *dagger.Directory
	// +private
	Version string
	// +private
	Flavor string
}

func New(
	ctx context.Context,
	// +optional
	// +default="4.6.1"
	version,
	// +optional
	// +default=stable
	flavor string,
	// +optional
	container *dagger.Container,
) (*Godot, error) {
	arch, err := dag.Arch().Oci(ctx)
	if err != nil {
		return nil, err
	}

	platform := "linux.64"
	slug := "linux.x86_64.zip"
	ext := ".x86_64"

	switch arch {
	case "amd64":
	case "arm64":
		platform = fmt.Sprintf("linux.%s", arch)
		slug = fmt.Sprintf("%s.zip", platform)
		ext = fmt.Sprintf(".%s", arch)
	default:
		return nil, fmt.Errorf("unsupported architecture %s", arch)
	}

	if container == nil {
		godotZipURL := fmt.Sprintf(
			"https://downloads.godotengine.org/?version=%s&flavor=%s&slug=%s&platform=%s",
			url.QueryEscape(version),
			url.QueryEscape(flavor),
			url.QueryEscape(slug),
			url.QueryEscape(platform),
		)
		godotZip := dag.HTTP(godotZipURL)
		godotZipPath := filepath.Join(workDir, slug)
		godotExtractedPath := filepath.Join(workDir, fmt.Sprintf("Godot_v%s-%s_linux%s", version, flavor, ext))
		godot := dag.Wolfi().
			Container().
			WithFile(godotZipPath, godotZip).
			WithExec([]string{"unzip", "-d", workDir, godotZipPath}).
			File(godotExtractedPath)
		container = dag.Wolfi().
			Container(dagger.WolfiContainerOpts{
				Packages: []string{"openssl", "zlib"},
			}).
			WithEnvVariable("HOME", "/root").
			WithFile("/usr/local/bin/godot", godot)
	}

	exportTemplatesZipURL := fmt.Sprintf(
		"https://downloads.godotengine.org/?version=%s&flavor=%s&slug=%s&platform=%s",
		url.QueryEscape(version),
		url.QueryEscape(flavor),
		url.QueryEscape("export_templates.tpz"),
		url.QueryEscape("templates"),
	)
	exportTemplatesZip := dag.HTTP(exportTemplatesZipURL)
	exportTemplatesZipPath := filepath.Join(workDir, slug)
	exportTemplatesExtractedPath := filepath.Join(workDir, "templates")
	exportTemplates := dag.Wolfi().
		Container().
		WithFile(exportTemplatesZipPath, exportTemplatesZip).
		WithExec([]string{"unzip", "-d", workDir, exportTemplatesZipPath}).
		Directory(exportTemplatesExtractedPath)

	return &Godot{
		Container:       container,
		ExportTemplates: exportTemplates,
		Version:         version,
		Flavor:          flavor,
	}, nil
}

func getExportPreset(exportPresets *ini.File, preset string) (*ini.Section, *ini.Section, error) {
	sections := exportPresets.Sections()
	availablePresets := make([]string, 0, len(sections))

	for _, section := range sections {
		sectionName := section.Name()
		if !strings.HasPrefix(sectionName, "preset.") || strings.HasSuffix(sectionName, ".options") {
			continue
		}

		if !section.HasKey("name") {
			continue
		}

		sectionPreset := section.Key("name").String()
		if sectionPreset == "" {
			continue
		}

		availablePresets = append(availablePresets, sectionPreset)
		if sectionPreset == preset {
			optionsSectionName := fmt.Sprintf("%s.options", sectionName)
			if !exportPresets.HasSection(optionsSectionName) {
				return nil, nil, fmt.Errorf(
					"export preset %q is missing options section %q",
					preset,
					optionsSectionName,
				)
			}

			return section, exportPresets.Section(optionsSectionName), nil
		}
	}

	sort.Strings(availablePresets)
	if len(availablePresets) == 0 {
		return nil, nil, fmt.Errorf("no export presets found in export_presets.cfg")
	}

	return nil, nil, fmt.Errorf(
		"export preset %q not found in export_presets.cfg (available: %s)",
		preset,
		strings.Join(availablePresets, ", "),
	)
}

type Export struct {
	Pck    *dagger.File
	Binary *dagger.File
}

func (m *Godot) ExportRelease(
	ctx context.Context,
	workspace *dagger.Workspace,
	preset,
	path,
	// +optional
	osslsigncodeVersion string,
	// +optional
	scriptEncryptionKey *dagger.Secret,
	// +optional
	windowsCodesignIdentityType string,
	// +optional
	windowsCodesignIdentity *dagger.File,
	// +optional
	windowsCodesignPassword *dagger.Secret,
) (*Export, error) {
	return m.export(ctx, "release", workspace.Directory("."), preset, path, osslsigncodeVersion, scriptEncryptionKey, windowsCodesignIdentityType, windowsCodesignIdentity, windowsCodesignPassword)
}

func (m *Godot) ExportDebug(
	ctx context.Context,
	workspace *dagger.Workspace,
	preset,
	path,
	// +optional
	osslsigncodeVersion string,
	// +optional
	scriptEncryptionKey *dagger.Secret,
	// +optional
	windowsCodesignIdentityType string,
	// +optional
	windowsCodesignIdentity *dagger.File,
	// +optional
	windowsCodesignPassword *dagger.Secret,
) (*Export, error) {
	return m.export(ctx, "debug", workspace.Directory("."), preset, path, osslsigncodeVersion, scriptEncryptionKey, windowsCodesignIdentityType, windowsCodesignIdentity, windowsCodesignPassword)
}

func osslsigncode(
	// +optional
	// +default="2.13"
	version string,
) *dagger.File {
	src := dag.Git("https://github.com/mtrojnar/osslsigncode").Ref(version).Tree()
	srcPath := "/src"
	buildPath := filepath.Join(srcPath, "build")

	return dag.Wolfi().
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
		File("/usr/local/bin/osslsigncode")
}

func (m *Godot) export(
	ctx context.Context,
	kind string,
	source *dagger.Directory,
	preset,
	path string,
	osslsigncodeVersion string,
	scriptEncryptionKey *dagger.Secret,
	windowsCodesignIdentityType string,
	windowsCodesignIdentity *dagger.File,
	windowsCodesignPassword *dagger.Secret,
) (*Export, error) {
	container := m.Container.WithWorkdir("/src").WithMountedDirectory(".", source)

	rawExportPresets, err := container.File("export_presets.cfg").Contents(ctx)
	if err != nil {
		return nil, err
	}

	exportPresets, err := ini.LoadSources(
		ini.LoadOptions{
			// Godot dedicated server export presets default to using multiline
			// strings which we don't care about and this library chokes on.
			// Beware: this could swallow some actual errors that manifest below.
			SkipUnrecognizableLines: true,
		},
		[]byte(rawExportPresets),
	)
	if err != nil {
		return nil, err
	}

	_, options, err := getExportPreset(exportPresets, preset)
	if err != nil {
		return nil, err
	}

	embedPck, _ := strconv.ParseBool(options.Key("binary_format/embed_pck").Value())


	xdgDataHome, err := container.EnvVariable(ctx, "XDG_DATA_HOME")
	if err != nil {
		return nil, err
	} else if xdgDataHome == "" {
		home, err := container.EnvVariable(ctx, "HOME")
		if err != nil {
			return nil, err
		}

		xdgDataHome = filepath.Join(home, ".local", "share")
	}
	
	export := container.
		WithDirectory(fmt.Sprintf("%s/godot/export_templates/%s.%s", xdgDataHome, m.Version, m.Flavor), m.ExportTemplates, dagger.ContainerWithDirectoryOpts{
			Expand: true,
		}).
		With(func(r *dagger.Container) *dagger.Container {
			if scriptEncryptionKey == nil {
				return r
			} else if embedPck {
				return r.WithError("cannot both embed pck and encrypt")
			}
			return r.WithSecretVariable("GODOT_SCRIPT_ENCRYPTION_KEY", scriptEncryptionKey)
		}).
		With(func(r *dagger.Container) *dagger.Container {
			if windowsCodesignIdentityType == "" {
				return r
			}
			return r.WithEnvVariable("GODOT_WINDOWS_CODESIGN_IDENTITY_TYPE", windowsCodesignIdentityType)
		}).
		With(func(r *dagger.Container) *dagger.Container {
			if windowsCodesignIdentity == nil {
				return r
			}
			windowsCodesignIdentityPath := "/tmp/windows-codesign-identity"
			return r.WithFile(
				"/usr/local/bin/osslsigncode",
				osslsigncode(osslsigncodeVersion),
			).
				WithMountedFile(windowsCodesignIdentityPath, windowsCodesignIdentity).
				WithEnvVariable("GODOT_WINDOWS_CODESIGN_IDENTITY", windowsCodesignIdentityPath)
		}).
		With(func(r *dagger.Container) *dagger.Container {
			if windowsCodesignPassword == nil {
				return r
			}
			return r.WithSecretVariable("GODOT_WINDOWS_CODESIGN_PASSWORD", windowsCodesignPassword)
		}).
		WithExec([]string{"godot", "--headless", fmt.Sprintf("--export-%s", kind), preset, path})

	if embedPck {
		return &Export{
			Binary: export.File(path),
		}, nil
	}

	return &Export{
		Binary: export.File(path),
		Pck:    export.File(fmt.Sprintf("%s.pck", strings.TrimSuffix(path, filepath.Ext(path)))),
	}, nil
}
