package main

import (
	"context"
	"fmt"
	"strings"
)

type Arch struct {
	// +private
	Arch string
	// +private
	Version string
}

func New(ctx context.Context) (*Arch, error) {
	defaultPlatform, err := dag.DefaultPlatform(ctx)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(defaultPlatform), "/")
	lenParts := len(parts)
	switch lenParts {
	case 0, 1:
		return nil, fmt.Errorf("invalid platform %s", defaultPlatform)
	case 2:
		return &Arch{Arch: parts[1]}, nil
	}
	return &Arch{Arch: parts[1], Version: parts[2]}, nil
}

// OCI returns the architecture in OCI/Go naming (e.g. "amd64", "arm64").
func (a *Arch) OCI() string {
	return a.Arch
}

// GNU returns the architecture in GNU/uname naming (e.g. "x86_64", "aarch64").
func (a *Arch) GNU() (string, error) {
	switch a.Arch {
	case "amd64":
		return "x86_64", nil
	case "arm64":
		return "aarch64", nil
	case "arm":
		return "armv7l", nil
	case "386":
		return "i686", nil
	default:
		return "", fmt.Errorf("unknown arch %q", a.Arch)
	}
}

// Microsoft returns the architecture in Microsoft naming (e.g. "x64", "arm64").
func (a *Arch) Microsoft() (string, error) {
	switch a.Arch {
	case "amd64":
		return "x64", nil
	case "arm64":
		return "arm64", nil
	case "arm":
		return "arm", nil
	case "386":
		return "x86", nil
	default:
		return "", fmt.Errorf("unknown arch %q", a.Arch)
	}
}
