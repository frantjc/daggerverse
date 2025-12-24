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
	"github.com/frantjc/daggerverse/dogger/internal/dogger/internal/storage"
	"github.com/google/uuid"
	archive "github.com/moby/go-archive"
)

var (
	ErrUnimplemented = errors.New("unimplemented")
)

type ContainerBackend struct {
	Context context.Context
	Client  *dagger.Client
	Storage storage.ContainerStore
}

var _ containerrouter.Backend = new(ContainerBackend)

// ContainerArchivePath implements container.Backend.
func (c *ContainerBackend) ContainerArchivePath(name string, path string) (content io.ReadCloser, stat *containertypes.PathStat, err error) {
	return nil, nil, ErrUnimplemented
}

// ContainerAttach implements container.Backend.
func (c *ContainerBackend) ContainerAttach(name string, cfg *backend.ContainerAttachConfig) error {
	return ErrUnimplemented
}

// ContainerChanges implements container.Backend.
func (c *ContainerBackend) ContainerChanges(ctx context.Context, name string) ([]archive.Change, error) {
	return nil, ErrUnimplemented
}

// ContainerCreate implements container.Backend.
func (c *ContainerBackend) ContainerCreate(ctx context.Context, config backend.ContainerCreateConfig) (containertypes.CreateResponse, error) {
	dag := c.Client

	container := dag.Container(dagger.ContainerOpts{Platform: getDaggerPlatform(config.Platform)})

	if config.Config == nil {
		return containertypes.CreateResponse{}, fmt.Errorf("image is required")
	}

	container = container.From(config.Config.Image)
	warnings := []string{}

	for exposedPort := range config.Config.ExposedPorts {
		container = container.WithExposedPort(exposedPort.Int(), dagger.ContainerWithExposedPortOpts{
			Protocol: dagger.NetworkProtocol(exposedPort.Proto()),
		})
	}

	for k, v := range config.Config.Labels {
		container = container.WithLabel(k, v)
	}

	if config.Config.User != "" {
		container = container.WithUser(config.Config.User)
	}

	if config.Config.WorkingDir != "" {
		container = container.WithWorkdir(config.Config.WorkingDir)
	}

	hostConfig := containertypes.HostConfig{}
	if config.HostConfig != nil {
		hostConfig = *config.HostConfig
	}

	for k, v := range hostConfig.Annotations {
		container = container.WithAnnotation(k, v)
	}

	id := newID()

	container = container.
		WithExec(append(config.Config.Entrypoint, config.Config.Cmd...), dagger.ContainerWithExecOpts{
			UseEntrypoint:                 len(config.Config.Entrypoint) == 0,
			ExperimentalPrivilegedNesting: true,
			InsecureRootCapabilities:      config.HostConfig.Privileged,
		})

	if err := c.Storage.CreateContainer(ctx, id, &storage.Container{
		ID:         id,
		Created:    time.Now(),
		Config:     *config.Config,
		HostConfig: hostConfig,
	}); err != nil {
		return containertypes.CreateResponse{ID: id, Warnings: warnings}, err
	}

	if config.Name == "" {
		config.Name = newName()
	}

	if err := c.Storage.NameContainer(ctx, id, id[:12]); err != nil {
		return containertypes.CreateResponse{ID: id, Warnings: warnings}, err
	}

	if err := c.Storage.NameContainer(ctx, id, config.Name); err != nil {
		return containertypes.CreateResponse{ID: id, Warnings: warnings}, err
	}

	return containertypes.CreateResponse{ID: id, Warnings: warnings}, nil
}

// ContainerExecCreate implements container.Backend.
func (c *ContainerBackend) ContainerExecCreate(name string, options *containertypes.ExecOptions) (string, error) {
	return "", ErrUnimplemented
}

// ContainerExecInspect implements container.Backend.
func (c *ContainerBackend) ContainerExecInspect(id string) (*backend.ExecInspect, error) {
	return nil, ErrUnimplemented
}

// ContainerExecResize implements container.Backend.
func (c *ContainerBackend) ContainerExecResize(ctx context.Context, name string, height uint32, width uint32) error {
	return ErrUnimplemented
}

// ContainerExecStart implements container.Backend.
func (c *ContainerBackend) ContainerExecStart(ctx context.Context, name string, options backend.ExecStartConfig) error {
	return ErrUnimplemented
}

// ContainerExport implements container.Backend.
func (c *ContainerBackend) ContainerExport(ctx context.Context, name string, out io.Writer) error {
	return ErrUnimplemented
}

// ContainerExtractToDir implements container.Backend.
func (c *ContainerBackend) ContainerExtractToDir(name string, path string, copyUIDGID bool, noOverwriteDirNonDir bool, content io.Reader) error {
	return ErrUnimplemented
}

// ContainerInspect implements container.Backend.
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

// ContainerKill implements container.Backend.
func (c *ContainerBackend) ContainerKill(name string, signal string) error {
	return c.serviceStop(c.Context, name, true)
}

func (c *ContainerBackend) serviceStop(ctx context.Context, name string, kill bool) error {
	data, err := c.Storage.GetContainer(ctx, name)
	if err != nil {
		return nil
	} // else if data.ServiceID == "" {
	// 	return nil
	// }

	// service := c.Client.LoadServiceFromID(dagger.ServiceID(data.ServiceID))

	// service, err = service.Stop(ctx, dagger.ServiceStopOpts{Kill: kill})
	// if err != nil {
	// 	return nil
	// }

	var _ = data

	return nil
}

// ContainerLogs implements container.Backend.
func (c *ContainerBackend) ContainerLogs(ctx context.Context, name string, config *containertypes.LogsOptions) (<-chan *backend.LogMessage, bool, error) {
	return nil, false, ErrUnimplemented
}

// ContainerPause implements container.Backend.
func (c *ContainerBackend) ContainerPause(name string) error {
	return ErrUnimplemented
}

// ContainerRename implements container.Backend.
func (c *ContainerBackend) ContainerRename(oldName string, newName string) error {
	return c.Storage.NameContainer(c.Context, oldName, newName)
}

// ContainerResize implements container.Backend.
func (c *ContainerBackend) ContainerResize(ctx context.Context, name string, height uint32, width uint32) error {
	return ErrUnimplemented
}

// ContainerRestart implements container.Backend.
func (c *ContainerBackend) ContainerRestart(ctx context.Context, name string, options containertypes.StopOptions) error {
	return ErrUnimplemented
}

// ContainerRm implements container.Backend.
func (c *ContainerBackend) ContainerRm(name string, config *backend.ContainerRmConfig) error {
	return c.Storage.DeleteContainer(c.Context, name)
}

// ContainerStart implements container.Backend.
func (c *ContainerBackend) ContainerStart(ctx context.Context, name string, checkpoint string, checkpointDir string) error {
	if _, err := c.Storage.GetContainer(ctx, name); err != nil {
		return err
	}

	return nil
}

// ContainerStatPath implements container.Backend.
func (c *ContainerBackend) ContainerStatPath(name string, path string) (stat *containertypes.PathStat, err error) {
	return nil, ErrUnimplemented
}

// ContainerStats implements container.Backend.
func (c *ContainerBackend) ContainerStats(ctx context.Context, name string, config *backend.ContainerStatsConfig) error {
	return ErrUnimplemented
}

// ContainerStop implements container.Backend.
func (c *ContainerBackend) ContainerStop(ctx context.Context, name string, options containertypes.StopOptions) error {
	cCtx := ctx
	cancel := func() {}

	switch {
	case options.Timeout == nil:
		cCtx, cancel = context.WithTimeout(cCtx, time.Second*10)
	case *options.Timeout == 0:
		return c.serviceStop(ctx, name, true)
	case *options.Timeout > 0:
		cCtx, cancel = context.WithTimeout(cCtx, time.Second*time.Duration(*options.Timeout))
	}
	defer cancel()

	if err := c.serviceStop(cCtx, name, false); errors.Is(err, context.Canceled) {
		return c.serviceStop(ctx, name, true)
	} else if err != nil {
		return err
	}

	return nil
}

// ContainerTop implements container.Backend.
func (c *ContainerBackend) ContainerTop(name string, psArgs string) (*containertypes.TopResponse, error) {
	return nil, ErrUnimplemented
}

// ContainerUnpause implements container.Backend.
func (c *ContainerBackend) ContainerUnpause(name string) error {
	return ErrUnimplemented
}

// ContainerUpdate implements container.Backend.
func (c *ContainerBackend) ContainerUpdate(name string, hostConfig *containertypes.HostConfig) (containertypes.UpdateResponse, error) {
	return containertypes.UpdateResponse{}, ErrUnimplemented
}

// ContainerWait implements container.Backend.
func (c *ContainerBackend) ContainerWait(ctx context.Context, name string, condition containertypes.WaitCondition) (<-chan containertypes.StateStatus, error) {
	// TODO(frantjc)
	exitCode := 0

	statuses := make(chan containertypes.StateStatus)
	statuses <- containertypes.NewStateStatus(exitCode, nil)

	return statuses, nil
}

// Containers implements container.Backend.
func (c *ContainerBackend) Containers(ctx context.Context, config *containertypes.ListOptions) ([]*containertypes.Summary, error) {
	containers, err := c.Storage.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]*containertypes.Summary, len(containers))

	for i, container := range containers {
		fmt.Println(container)

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

// ContainersPrune implements container.Backend.
func (c *ContainerBackend) ContainersPrune(ctx context.Context, pruneFilters filterstypes.Args) (*containertypes.PruneReport, error) {
	return nil, ErrUnimplemented
}

// CreateImageFromContainer implements container.Backend.
func (c *ContainerBackend) CreateImageFromContainer(ctx context.Context, name string, config *backend.CreateImageConfig) (imageID string, err error) {
	return "", ErrUnimplemented
}

// ExecExists implements container.Backend.
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
