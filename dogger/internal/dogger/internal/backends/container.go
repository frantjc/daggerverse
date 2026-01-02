package backends

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"dagger.io/dagger"
	containerrouter "github.com/docker/docker/api/server/router/container"
	"github.com/docker/docker/api/types/backend"
	containertypes "github.com/docker/docker/api/types/container"
	filterstypes "github.com/docker/docker/api/types/filters"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/frantjc/daggerverse/dogger/internal/dogger/internal/dagutil"
	"github.com/frantjc/daggerverse/dogger/internal/dogger/internal/storage"
	"github.com/google/uuid"
	archive "github.com/moby/go-archive"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"go.opentelemetry.io/otel"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
)

var (
	ErrUnimplemented = errors.New("unimplemented")
)

type ContainerBackend struct {
	// Context is the fallback ctx for functions that do not accept a ctx argument.
	Context context.Context
	// Storage is where data about the containers gets kept.
	Storage storage.ContainerStore
}

var _ containerrouter.Backend = new(ContainerBackend)

// ContainerArchivePath implements containerrouter.Backend.
func (c *ContainerBackend) ContainerArchivePath(name string, path string) (content io.ReadCloser, stat *containertypes.PathStat, err error) {
	return nil, nil, ErrUnimplemented
}

// ContainerAttach implements containerrouter.Backend.
func (c *ContainerBackend) ContainerAttach(name string, cfg *backend.ContainerAttachConfig) error {
	ctx, cancel := context.WithCancel(c.Context)
	defer cancel()

	data, err := c.Storage.GetContainer(ctx, name)
	if err != nil {
		return err
	}

	stdin, stdout, stderr, err := cfg.GetStreams(false, cancel)
	if err != nil {
		return err
	}

	var (
		_ = data
		_ = stdin
		_ = stdout
		_ = stderr
	)

	return nil
}

// ContainerChanges implements containerrouter.Backend.
func (c *ContainerBackend) ContainerChanges(ctx context.Context, name string) ([]archive.Change, error) {
	return nil, ErrUnimplemented
}

// ContainerCreate implements containerrouter.Backend.
func (c *ContainerBackend) ContainerCreate(ctx context.Context, config backend.ContainerCreateConfig) (containertypes.CreateResponse, error) {
	if config.Config == nil {
		return containertypes.CreateResponse{}, fmt.Errorf("image is required")
	}

	warnings := []string{}

	hostConfig := containertypes.HostConfig{}
	if config.HostConfig != nil {
		hostConfig = *config.HostConfig
	}

	id := newID()

	platform := ocispec.Platform{}
	if config.Platform != nil {
		platform = *config.Platform
	}

	if err := c.Storage.CreateContainer(ctx, id, &storage.Container{
		ID:         id,
		Created:    time.Now(),
		Config:     *config.Config,
		HostConfig: hostConfig,
		Platform:   platform,
	}); err != nil {
		return containertypes.CreateResponse{}, err
	}

	if config.Name == "" {
		config.Name = newName()
	}

	if err := c.Storage.NameContainer(ctx, id, id[:12]); err != nil {
		return containertypes.CreateResponse{}, err
	}

	if err := c.Storage.NameContainer(ctx, id, config.Name); err != nil {
		return containertypes.CreateResponse{}, err
	}

	return containertypes.CreateResponse{ID: id, Warnings: warnings}, nil
}

// ContainerExecCreate implements containerrouter.Backend.
func (c *ContainerBackend) ContainerExecCreate(name string, options *containertypes.ExecOptions) (string, error) {
	return "", ErrUnimplemented
}

// ContainerExecInspect implements containerrouter.Backend.
func (c *ContainerBackend) ContainerExecInspect(id string) (*backend.ExecInspect, error) {
	return nil, ErrUnimplemented
}

// ContainerExecResize implements containerrouter.Backend.
func (c *ContainerBackend) ContainerExecResize(ctx context.Context, name string, height uint32, width uint32) error {
	return ErrUnimplemented
}

// ContainerExecStart implements containerrouter.Backend.
func (c *ContainerBackend) ContainerExecStart(ctx context.Context, name string, options backend.ExecStartConfig) error {
	return ErrUnimplemented
}

// ContainerExport implements containerrouter.Backend.
func (c *ContainerBackend) ContainerExport(ctx context.Context, name string, out io.Writer) error {
	return ErrUnimplemented
}

// ContainerExtractToDir implements containerrouter.Backend.
func (c *ContainerBackend) ContainerExtractToDir(name string, path string, copyUIDGID bool, noOverwriteDirNonDir bool, content io.Reader) error {
	return ErrUnimplemented
}

// ContainerInspect implements containerrouter.Backend.
func (c *ContainerBackend) ContainerInspect(ctx context.Context, name string, options backend.ContainerInspectOptions) (*containertypes.InspectResponse, error) {
	data, err := c.Storage.GetContainer(ctx, name)
	if err != nil {
		return nil, err
	}

	return &containertypes.InspectResponse{
		ContainerJSONBase: &containertypes.ContainerJSONBase{
			ID:   data.ID,
			Args: append(data.Config.Entrypoint, data.Config.Cmd...),
			State: &containertypes.State{
				Status:  containertypes.StateRunning,
				Running: true,
			},
			Name:       name,
			HostConfig: &data.HostConfig,
		},
		Config: &data.Config,
		NetworkSettings: &containertypes.NetworkSettings{
			Networks: map[string]*networktypes.EndpointSettings{},
		},
	}, nil
}

// ContainerKill implements containerrouter.Backend.
func (c *ContainerBackend) ContainerKill(name string, signal string) error {
	return ErrUnimplemented
}

// ContainerLogs implements containerrouter.Backend.
func (c *ContainerBackend) ContainerLogs(ctx context.Context, name string, config *containertypes.LogsOptions) (<-chan *backend.LogMessage, bool, error) {
	return nil, false, ErrUnimplemented
}

// ContainerPause implements containerrouter.Backend.
func (c *ContainerBackend) ContainerPause(name string) error {
	return ErrUnimplemented
}

// ContainerRename implements containerrouter.Backend.
func (c *ContainerBackend) ContainerRename(oldName string, newName string) error {
	return c.Storage.NameContainer(c.Context, oldName, newName)
}

// ContainerResize implements containerrouter.Backend.
func (c *ContainerBackend) ContainerResize(ctx context.Context, name string, height uint32, width uint32) error {
	return ErrUnimplemented
}

// ContainerRestart implements containerrouter.Backend.
func (c *ContainerBackend) ContainerRestart(ctx context.Context, name string, options containertypes.StopOptions) error {
	return ErrUnimplemented
}

// ContainerRm implements containerrouter.Backend.
func (c *ContainerBackend) ContainerRm(name string, config *backend.ContainerRmConfig) error {
	return c.Storage.DeleteContainer(c.Context, name)
}

// ContainerStart implements containerrouter.Backend.
func (c *ContainerBackend) ContainerStart(ctx context.Context, name string, checkpoint string, checkpointDir string) error {
	data, err := c.Storage.GetContainer(ctx, name)
	if err != nil {
		return err
	}

	ctx, span := otel.Tracer("").Start(ctx, "repro")
	defer span.End()

	var _ = span.SpanContext().SpanID().String() // spanID. All logs from this span and its children are from the service.

	dag, err := dagutil.Connect(ctx,
		func(req *collogspb.ExportLogsServiceRequest) {
			// TODO(frantjc) Store logs retrievable as a log chan and an io.Reader by children of a given spanID.
		},
		func(req *coltracepb.ExportTraceServiceRequest) {
			// TODO(frantjc): Track span lineage.
		},
	)
	if err != nil {
		return err
	}

	config := data.Config
	hostConfig := data.HostConfig
	platform := &data.Platform

	container := dag.
		Container(dagger.ContainerOpts{
			Platform: getDaggerPlatform(platform),
		}).
		From(config.Image)

	for exposedPort := range config.ExposedPorts {
		container = container.WithExposedPort(exposedPort.Int(), dagger.ContainerWithExposedPortOpts{
			Protocol: dagger.NetworkProtocol(exposedPort.Proto()),
		})
	}

	for k, v := range config.Labels {
		container = container.WithLabel(k, v)
	}

	if config.User != "" {
		container = container.WithUser(config.User)
	}

	if config.WorkingDir != "" {
		container = container.WithWorkdir(config.WorkingDir)
	}

	for k, v := range hostConfig.Annotations {
		container = container.WithAnnotation(k, v)
	}

	svc := container.
		AsService(dagger.ContainerAsServiceOpts{
			UseEntrypoint:                 len(config.Entrypoint) == 0,
			Args:                          append(config.Entrypoint, config.Cmd...),
			ExperimentalPrivilegedNesting: true,
			InsecureRootCapabilities:      hostConfig.Privileged,
		})

	if config.Hostname != "" {
		svc, err = svc.
			WithHostname(config.Hostname).
			Start(ctx)
		if err != nil {
			return err
		}
	} else {
		svc, err = svc.Start(ctx)
		if err != nil {
			return err
		}

		config.Hostname, err = svc.Endpoint(ctx)
		if err != nil {
			return err
		}

		if err = c.Storage.UpdateContainer(ctx, name, data); err != nil {
			return err
		}
	}

	return nil
}

// ContainerStatPath implements containerrouter.Backend.
func (c *ContainerBackend) ContainerStatPath(name string, path string) (stat *containertypes.PathStat, err error) {
	return nil, ErrUnimplemented
}

// ContainerStats implements containerrouter.Backend.
func (c *ContainerBackend) ContainerStats(ctx context.Context, name string, config *backend.ContainerStatsConfig) error {
	return ErrUnimplemented
}

// ContainerStop implements containerrouter.Backend.
func (c *ContainerBackend) ContainerStop(ctx context.Context, name string, options containertypes.StopOptions) error {
	return ErrUnimplemented
}

// ContainerTop implements containerrouter.Backend.
func (c *ContainerBackend) ContainerTop(name string, psArgs string) (*containertypes.TopResponse, error) {
	return nil, ErrUnimplemented
}

// ContainerUnpause implements containerrouter.Backend.
func (c *ContainerBackend) ContainerUnpause(name string) error {
	return ErrUnimplemented
}

// ContainerUpdate implements containerrouter.Backend.
func (c *ContainerBackend) ContainerUpdate(name string, hostConfig *containertypes.HostConfig) (containertypes.UpdateResponse, error) {
	return containertypes.UpdateResponse{}, ErrUnimplemented
}

// ContainerWait implements containerrouter.Backend.
func (c *ContainerBackend) ContainerWait(ctx context.Context, name string, condition containertypes.WaitCondition) (<-chan containertypes.StateStatus, error) {
	// TODO(frantjc)
	exitCode := 0

	statuses := make(chan containertypes.StateStatus)
	statuses <- containertypes.NewStateStatus(exitCode, nil)

	return statuses, nil
}

// Containers implements containerrouter.Backend.
func (c *ContainerBackend) Containers(ctx context.Context, config *containertypes.ListOptions) ([]*containertypes.Summary, error) {
	data, err := c.Storage.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]*containertypes.Summary, len(data))

	for i, container := range data {
		ports := []containertypes.Port{}

		for port, portBindings := range container.HostConfig.PortBindings {
			for _, portBinding := range portBindings {
				publicPort, err := strconv.Atoi(portBinding.HostPort)
				if err != nil {
					return nil, err
				}

				ports = append(ports, containertypes.Port{
					IP:          portBinding.HostIP,
					PrivatePort: uint16(port.Int()),
					PublicPort:  uint16(publicPort),
					Type:        port.Proto(),
				})
			}
		}

		summaries[i] = &containertypes.Summary{
			ID:      container.ID,
			Created: container.Created.Unix(),
			Labels:  container.Config.Labels,
			HostConfig: struct {
				NetworkMode string            "json:\",omitempty\""
				Annotations map[string]string "json:\",omitempty\""
			}{
				Annotations: container.HostConfig.Annotations,
				NetworkMode: string(container.HostConfig.NetworkMode),
			},
			State:   containertypes.NoHealthcheck,
			Status:  "TODO",
			Command: strings.Join(container.Config.Cmd, " "),
			Ports:   ports,
		}
	}

	return summaries, nil
}

// ContainersPrune implements containerrouter.Backend.
func (c *ContainerBackend) ContainersPrune(ctx context.Context, pruneFilters filterstypes.Args) (*containertypes.PruneReport, error) {
	return nil, ErrUnimplemented
}

// CreateImageFromContainer implements containerrouter.Backend.
func (c *ContainerBackend) CreateImageFromContainer(ctx context.Context, name string, config *backend.CreateImageConfig) (imageID string, err error) {
	return "", ErrUnimplemented
}

// ExecExists implements containerrouter.Backend.
func (c *ContainerBackend) ExecExists(name string) (bool, error) {
	return false, ErrUnimplemented
}

var (
	adjectives = []string{
		"admiring", "adoring", "affectionate", "agitated", "amazing", "angry", "awesome", "beautiful", "blissful", "bold",
		"boring", "brave", "busy", "charming", "clever", "cool", "compassionate", "competent", "condescending", "confident",
		"cranky", "crazy", "dazzling", "determined", "distracted", "dreamy", "eager", "ecstatic", "elastic", "elated",
		"elegant", "eloquent", "epic", "exciting", "fervent", "festive", "flamboyant", "focused", "friendly", "frantic",
		"frosty", "funny", "gallant", "gifted", "goofy", "gracious", "great", "happy", "hardcore", "heuristic",
		"hopeful", "hungry", "infallible", "inspiring", "intelligent", "interesting", "jolly", "jovial", "keen", "kind",
		"laughing", "loving", "lucid", "magical", "mystifying", "modest", "musing", "naughty", "nervous", "nice",
		"nifty", "nostalgic", "objective", "optimistic", "peaceful", "pedantic", "pensive", "practical", "priceless", "quirky",
		"quizzical", "recursing", "relaxed", "reverent", "romantic", "sad", "serene", "sharp", "silly", "sleepy",
		"stoic", "strange", "stupefied", "suspicious", "sweet", "tender", "thirsty", "trusting", "unruffled", "upbeat",
		"vibrant", "vigilant", "vigorous", "wizardly", "wonderful", "xenodochial", "youthful", "zealous", "zen",
	}
	nouns = []string{
		"albattani", "allen", "almeida", "antonelli", "agnesi", "archimedes", "ardinghelli", "aryabhata", "austin", "babbage",
		"banach", "banzai", "bardeen", "bartik", "bassi", "beaver", "bell", "benz", "bhabha", "bhaskara",
		"black", "blackburn", "blackwell", "bohr", "booth", "borg", "bose", "bouman", "boyd", "brahmagupta",
		"brattain", "brown", "buck", "burnell", "cannon", "carson", "cartwright", "carver", "cerf", "chandrasekhar",
		"chaplygin", "chatelet", "chatterjee", "chebyshev", "chicken", "cohen", "chaum", "clarke", "colden", "cori",
		"cray", "curran", "curie", "darwin", "davinci", "dewdney", "dhawan", "diffie", "dijkstra", "dirac",
		"driscoll", "dubinsky", "easley", "edison", "einstein", "elbakyan", "elgamal", "elion", "ellis", "engelbart",
		"euclid", "euler", "faraday", "feistel", "fermat", "fermi", "feynman", "franklin", "gagarin", "galileo",
		"galois", "ganguly", "gates", "gauss", "germain", "goldberg", "goldstine", "goldwasser", "golick", "goodall",
		"gould", "greider", "grothendieck", "haibt", "hamilton", "haslett", "hawking", "hellman", "hermann", "herschel",
		"hertz", "heyrovsky", "hodgkin", "hofstadter", "hoover", "hopper", "hugle", "hypatia", "ishizaka", "jackson",
		"jang", "jemison", "jennings", "jepsen", "johnson", "joliot", "jones", "kalam", "kapitsa", "kare",
		"keldysh", "keller", "kepler", "khayyam", "khorana", "kilby", "kirch", "knuth", "kowalevski", "lalande",
		"lamarr", "lamport", "leakey", "leavitt", "lederberg", "lehmann", "lewin", "lichterman", "liskov", "lovelace",
		"lumiere", "mahavira", "margulis", "matsumoto", "maxwell", "mayer", "mccarthy", "mcclintock", "mclaren", "mclean",
		"mcnulty", "mendel", "mendeleev", "meitner", "meninsky", "merkle", "mestorf", "mirzakhani", "moore", "morse",
		"murdoch", "moser", "napier", "nash", "neumann", "newton", "nightingale", "nobel", "noether", "northcutt",
		"noyce", "panini", "pare", "pascal", "pasteur", "payne", "perlman", "pike", "poincare", "poitras",
		"proskuriakova", "ptolemy", "raman", "ramanujan", "ride", "montalcini", "ritchie", "rhodes", "robinson", "roentgen",
		"rosalind", "roy", "rubin", "saha", "sammet", "sanderson", "shaw", "sherrington", "shockley", "shtern",
		"sinoussi", "snyder", "solomon", "spence", "stallman", "stonebraker", "sutherland", "swanson", "swartz", "swirles",
		"taussig", "tereshkova", "tesla", "tharp", "thompson", "torvalds", "tu", "turing", "ulam", "unruh",
		"van", "varahamihira", "vaughan", "visvesvaraya", "volhard", "villani", "wescoff", "wilbur", "wiles", "williams",
		"williamson", "wilson", "wing", "wozniak", "wright", "wu", "yalow", "yonath", "zhukovsky",
	}
)

func newName() string {
	adjective := adjectives[rand.IntN(len(adjectives))]
	noun := nouns[rand.IntN(len(nouns))]
	return fmt.Sprintf("%s_%s", adjective, noun)
}

func newID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}
