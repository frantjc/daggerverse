package main

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/frantjc/daggerverse/kwok/internal/dagger"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/yaml"
)

type Kwok struct{}

type Default string

func (d Default) String() (string, error) {
	s := ""
	return s, yaml.Unmarshal([]byte(d), &s)
}

func (d Default) URL() (*url.URL, error) {
	s, err := d.String()
	if err != nil {
		return nil, err
	}

	return url.Parse(s)
}

func (d Default) List() ([]string, error) {
	s := []string{}
	return s, yaml.Unmarshal([]byte(d), &s)
}

var (
	parseHelpRegexp = regexp.MustCompile(`(?s)^(--\S+).*\(default ([^)]+)\)`)
)

// parseHelp parses all flag help lines from rawHelp output into a map of
// flag name → raw Default, e.g. "etcd-binary" → Default(`"/usr/local/bin/etcd"`).
func parseHelp(rawHelp string) map[string]Default {
	flags := make(map[string]Default)
	for chunk := range strings.SplitSeq(rawHelp, "\n      --") {
		if m := parseHelpRegexp.FindStringSubmatch(fmt.Sprintf("--%s", chunk)); m != nil {
			flags[m[1]] = Default(m[2])
		}
	}
	return flags
}

func (m *Kwok) Cluster(
	ctx context.Context,
	// +optional
	// +defaultAddress="registry.k8s.io/kwok/cluster:v0.7.0-k8s.v1.33.0"
	container *dagger.Container,
	// +optional
	components []string,
	// +optional
	controllerPort int,
	// +optional
	disable []string,
	// +optional
	disableQPSLimits bool,
	// +optional
	enable []string,
	// +optional
	enableCRDs []string,
	// +optional
	// +default=8080
	kubeApiserverInsecurePort int,
	// +optional
	kubeAuditPolicy *dagger.File,
	// +optional
	etcd *dagger.File,
	// +optional
	jaeger *dagger.File,
	// +optional
	kubeApiserver *dagger.File,
	// +optional
	kubeScheduler *dagger.File,
	// +optional
	kubeControllerManager *dagger.File,
	// +optional
	kwokController *dagger.File,
	// +optional
	metricsServer *dagger.File,
	// +optional
	prometheus *dagger.File,
) (*Cluster, error) {
	rawHelp, err := container.WithExec([]string{"kwokctl", "create", "cluster", "--help"}).Stdout(ctx)
	if err != nil {
		return nil, err
	}

	flags := parseHelp(rawHelp)

	componentMap := make(map[string]struct{})
	for _, c := range components {
		componentMap[c] = struct{}{}
	}

	if len(components) == 0 {
		if def, ok := flags["--components"]; ok {
			cs, err := def.List()
			if err != nil {
				return nil, err
			}
			for _, c := range cs {
				componentMap[c] = struct{}{}
			}
		} else {
			return nil, fmt.Errorf("parse default components")
		}
	}

	createClusterExec := []string{"kwokctl", "create", "cluster", "--wait", "9s", "--runtime", "binary"}

	for _, component := range components {
		createClusterExec = append(createClusterExec, "--components", component)
	}

	if controllerPort != 0 {
		container = container.WithExposedPort(controllerPort, dagger.ContainerWithExposedPortOpts{
			Description: "kwok-controller",
		})
		createClusterExec = append(createClusterExec, "--controller-port", fmt.Sprint(controllerPort))
	}

	for _, d := range disable {
		createClusterExec = append(createClusterExec, "--disable", d)
	}

	if disableQPSLimits {
		createClusterExec = append(createClusterExec, "--disable-qps-limits")
	}

	for _, e := range enable {
		createClusterExec = append(createClusterExec, "--enable", e)
	}

	for _, e := range enableCRDs {
		createClusterExec = append(createClusterExec, "--enable-crds", e)
	}

	binaryArgs := map[string]*dagger.File{
		"etcd":                    etcd,
		"jaeger":                  jaeger,
		"kube-apiserver":          kubeApiserver,
		"kube-scheduler":          kubeScheduler,
		"kube-controller-manager": kubeControllerManager,
		"kwok-controller":         kwokController,
		"metrics-server":          metricsServer,
		"prometheus":              prometheus,
	}

	archive := dag.Archive()

	for binary := range binaryArgs {
		if _, ok := componentMap[binary]; !ok {
			continue
		}

		binaryFlag := fmt.Sprintf("--%s-binary", binary)
		path := fmt.Sprintf("/usr/local/bin/%s", binary)

		if f := binaryArgs[binary]; f != nil {
			container = container.WithFile(path, f, dagger.ContainerWithFileOpts{Permissions: 0700})
			createClusterExec = append(createClusterExec, binaryFlag, path)
		} else if binaryDefault, ok := flags[binaryFlag]; ok {
			binaryDefaultURL, err := binaryDefault.URL()
			if err != nil {
				return nil, err
			}

			switch binaryDefaultURL.Scheme {
			case "http", "https":
				file := dag.HTTP(binaryDefaultURL.String())
				if fragment := binaryDefaultURL.Fragment; fragment != "" {
					tar := archive.Untar(file)
					name, err := tar.Name(ctx)
					if err != nil {
						return nil, err
					}
					file = tar.Directory(name).File(fragment)
				}
				path := fmt.Sprintf("/usr/local/bin/%s", binary)
				container = container.WithFile(path, file, dagger.ContainerWithFileOpts{Permissions: 0700})
				createClusterExec = append(createClusterExec, binaryFlag, path)
			}
		}
	}

	if kubeAuditPolicy != nil {
		path := "$HOME/.kwok/kube-audit-policy.yaml"
		container = container.WithFile(path, kubeAuditPolicy, dagger.ContainerWithFileOpts{
			Expand: true,
		})
		createClusterExec = append(createClusterExec, "--kube-audit-policy", path)
	}

	container = container.WithExposedPort(kubeApiserverInsecurePort, dagger.ContainerWithExposedPortOpts{
		Description: "kube-apiserver",
	}).
		WithExec(createClusterExec, dagger.ContainerWithExecOpts{
			Expand: true,
		})

	return &Cluster{
		Container:                 container,
		KubeApiserverInsecurePort: kubeApiserverInsecurePort,
	}, nil
}

type Cluster struct {
	Container *dagger.Container
	// +private
	KubeApiserverInsecurePort int
}

func (c *Cluster) KubeConfig(
	ctx context.Context,
	// +optional
	// +default="kwok"
	alias string,
) (*dagger.File, error) {
	// TODO(frantjc): It would be nice to use the ~/.kube/config from the Service Container.
	cfg := api.NewConfig()
	cfg.Clusters[alias] = &api.Cluster{
		Server: fmt.Sprintf("http://%s:%d", alias, c.KubeApiserverInsecurePort),
	}
	cfg.AuthInfos[alias] = api.NewAuthInfo()
	cfg.Contexts[alias] = &api.Context{
		Cluster:  alias,
		AuthInfo: alias,
	}
	cfg.CurrentContext = alias

	rawKubeconfig, err := clientcmd.Write(*cfg)
	if err != nil {
		return nil, err
	}

	return dag.File("config", string(rawKubeconfig)), nil
}
