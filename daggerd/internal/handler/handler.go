package handler

import (
	"context"
	"net/http"

	"github.com/docker/docker/api/server"
	"github.com/docker/docker/api/server/middleware"
	"github.com/docker/docker/api/server/router"
	"github.com/docker/docker/api/server/router/checkpoint"
	"github.com/docker/docker/api/server/router/container"
	"github.com/docker/docker/api/server/router/image"
	"github.com/docker/docker/api/server/router/system"
	"github.com/docker/docker/pkg/sysinfo"
	"github.com/docker/docker/runconfig"
)

type Handler struct {
	server  *server.Server
	sysInfo *sysinfo.SysInfo
	routers []router.Router
	decoder runconfig.ContainerDecoder
}

func (h *Handler) getSysInfo() *sysinfo.SysInfo {
	return h.sysInfo
}

func (h *Handler) features() map[string]bool {
	return map[string]bool{"buildkit": true}
}

type middlewareFunc func(func(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error) func(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error

func (m middlewareFunc) WrapHandler(f func(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error) func(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	return m(f)
}

func New() *Handler {
	h := &Handler{
		server:  new(server.Server),
		sysInfo: sysinfo.New(),
		routers: []router.Router{},
	}
	h.server.UseMiddleware(middlewareFunc(middleware.DebugRequestMiddleware))
	h.decoder = runconfig.ContainerDecoder{GetSysInfo: h.getSysInfo}
	return h
}

func (h *Handler) WithRouters(r ...router.Router) *Handler {
	h.routers = append(h.routers, r...)
	return h
}

func (h *Handler) WithContainerBackend(b container.Backend) *Handler {
	return h.WithRouters(container.NewRouter(b, h.decoder, h.sysInfo.CgroupUnified))
}

func (h *Handler) WithImageBackend(b image.Backend, s image.Searcher) *Handler {
	return h.WithRouters(image.NewRouter(b, s))
}

func (h *Handler) WithSystemBackend(b system.Backend, cb system.ClusterBackend, bb system.BuildBackend) *Handler {
	return h.WithRouters(system.NewRouter(b, cb, bb, h.features))
}

func (h *Handler) WithCheckpointBackend(b checkpoint.Backend) *Handler {
	return h.WithRouters(checkpoint.NewRouter(b, h.decoder))
}

func (h *Handler) Handler(ctx context.Context, experimental bool) http.Handler {
	routers := make([]router.Router, len(h.routers))
	copy(routers, h.routers)

	for _, r := range routers {
		for _, route := range r.Routes() {
			if experimentalRoute, ok := route.(router.ExperimentalRoute); ok {
				if experimental {
					experimentalRoute.Enable()
				} else {
					experimentalRoute.Disable()
				}
			}
		}
	}

	h.server.UseMiddleware(middleware.NewExperimentalMiddleware(experimental))
	mux := h.server.CreateMux(ctx, routers...)

	return mux
}
