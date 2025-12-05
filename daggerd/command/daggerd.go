package command

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"dagger.io/dagger"
	"dagger.io/dagger/telemetry"
	"github.com/containerd/log"
	"github.com/dagger/dagger/dagql/dagui"
	"github.com/frantjc/daggerverse/daggerd/internal/backends"
	"github.com/frantjc/daggerverse/daggerd/internal/handler"
	sqlstorage "github.com/frantjc/daggerverse/daggerd/internal/storage/sql"
	"github.com/spf13/cobra"
	"github.com/vito/go-sse/sse"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

func NewDaggerd(version string) *cobra.Command {
	var (
		listener string
		cmd      = &cobra.Command{
			Use:           "daggerd",
			SilenceErrors: true,
			SilenceUsage:  true,
			Version:       version,
			RunE: func(cmd *cobra.Command, args []string) error {
				eg, ctx := errgroup.WithContext(cmd.Context())

				if !strings.Contains(listener, "://") {
					// Default to listening on ip:port.
					listener = fmt.Sprintf("tcp://%s", listener)
				}

				l, err := url.Parse(listener)
				if err != nil {
					return err
				}

				lis, err := net.Listen(l.Scheme, strings.TrimPrefix(l.String(), fmt.Sprintf("%s://", l.Scheme)))
				if err != nil {
					return err
				}

				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					return err
				}

				store, err := sqlstorage.New(db)
				if err != nil {
					return err
				}

				var (
					dag *dagger.Client
					tel = dagui.NewDB()
				)

				if rawDaggerSessionPort := os.Getenv("DAGGER_SESSION_PORT"); rawDaggerSessionPort != "" {
					daggerSessionPort, err := strconv.Atoi(rawDaggerSessionPort)
					if err != nil {
						return err
					}

					logEventSource, err := sse.Connect(http.DefaultClient, time.Second, func() *http.Request {
						return (&http.Request{
							URL: &url.URL{
								Scheme: "http",
								Host:   fmt.Sprintf("localhost:%d", daggerSessionPort),
								Path:   "/v1/logs",
							},
						}).WithContext(ctx)
					})
					if err != nil {
						return err
					}
					defer logEventSource.Close()

					traceEventSource, err := sse.Connect(http.DefaultClient, time.Second, func() *http.Request {
						return (&http.Request{
							URL: &url.URL{
								Scheme: "http",
								Host:   fmt.Sprintf("localhost:%d", daggerSessionPort),
								Path:   "/v1/traces",
							},
						}).WithContext(ctx)
					})
					if err != nil {
						return err
					}
					defer traceEventSource.Close()

					eg.Go(func() error {
						for {
							e, err := logEventSource.Next()
							if err != nil {
								// TODO(frantjc): log.
							}

							req := collogspb.ExportLogsServiceRequest{}
							if err := proto.Unmarshal(e.Data, &req); err != nil {
								// TODO(frantjc): log.
							}

							if err := telemetry.ReexportLogsFromPB(ctx, tel.LogExporter(), &req); err != nil {
								// TODO(frantjc): log.
							}

							if err := e.Write(cmd.ErrOrStderr()); err != nil {
								// TODO(frantjc): log.
							}
						}
					})

					eg.Go(func() error {
						for {
							e, err := traceEventSource.Next()
							if err != nil {
								// TODO(frantjc): log.
								continue
							}

							req := coltracepb.ExportTraceServiceRequest{}
							if err := proto.Unmarshal(e.Data, &req); err != nil {
								// TODO(frantjc): log.
							}

							if err := tel.ExportSpans(ctx, telemetry.SpansFromPB(req.ResourceSpans)); err != nil {
								// TODO(frantjc): log.
							}

							if err := e.Write(cmd.ErrOrStderr()); err != nil {
								// TODO(frantjc): log.
							}
						}
					})

					dag, err = dagger.Connect(ctx)
					if err != nil {
						return err
					}
				} else {
					mux := http.NewServeMux()

					mux.HandleFunc("POST /v1/logs", func(w http.ResponseWriter, r *http.Request) {
						body, err := io.ReadAll(r.Body)
						if err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}

						req := collogspb.ExportLogsServiceRequest{}
						if err := proto.Unmarshal(body, &req); err != nil {
							http.Error(w, err.Error(), http.StatusBadRequest)
							return
						}

						if err := telemetry.ReexportLogsFromPB(ctx, tel.LogExporter(), &req); err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
					})

					mux.HandleFunc("POST /v1/traces", func(w http.ResponseWriter, r *http.Request) {
						body, err := io.ReadAll(r.Body)
						if err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}

						req := coltracepb.ExportTraceServiceRequest{}
						if err := proto.Unmarshal(body, &req); err != nil {
							http.Error(w, err.Error(), http.StatusBadRequest)
							return
						}

						if err := tel.ExportSpans(ctx, telemetry.SpansFromPB(req.ResourceSpans)); err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
					})

					otelSrv := &http.Server{
						ReadHeaderTimeout: time.Second * 5,
						Handler:           mux,
						BaseContext: func(_ net.Listener) context.Context {
							return ctx
						},
					}

					otelLis, err := net.Listen("tcp", "127.0.0.1:0")
					if err != nil {
						return err
					}

					dag, err = dagger.Connect(ctx,
						dagger.WithEnvironmentVariable("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", "http://"+otelLis.Addr().String()+"/v1/logs"),
						dagger.WithEnvironmentVariable("OTEL_EXPORTER_OTLP_LOGS_PROTOCOL", "http/protobuf"),
						dagger.WithEnvironmentVariable("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "http://"+otelLis.Addr().String()+"/v1/traces"),
						dagger.WithEnvironmentVariable("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "http/protobuf"),
					)
					if err != nil {
						return err
					}

					eg.Go(func() error {
						return otelSrv.Serve(otelLis)
					})

					eg.Go(func() error {
						<-ctx.Done()
						if err = otelSrv.Shutdown(context.WithoutCancel(ctx)); err != nil {
							return err
						}
						return ctx.Err()
					})
				}

				if err := log.SetLevel(log.DebugLevel.String()); err != nil {
					return err
				}

				srv := &http.Server{
					ReadHeaderTimeout: time.Second * 5,
					Handler: handler.New().
						// TODO(frantjc): Implement all of these.
						WithSystemBackend(&backends.SystemBackend{}, nil, nil).
						WithImageBackend(&backends.ImageBackend{Client: dag, Storage: store}, nil).
						WithContainerBackend(&backends.ContainerBackend{Context: ctx, Client: dag, Storage: store}).
						Handler(ctx, false),
					BaseContext: func(_ net.Listener) context.Context {
						return log.WithLogger(ctx, log.L)
					},
				}

				eg.Go(func() error {
					return srv.Serve(lis)
				})

				eg.Go(func() error {
					<-ctx.Done()
					if err = srv.Shutdown(context.WithoutCancel(ctx)); err != nil {
						return err
					}
					return ctx.Err()
				})

				return eg.Wait()
			},
		}
	)

	cmd.Flags().BoolP("help", "h", false, "Help for "+cmd.Name())
	cmd.Flags().Bool("version", false, "Version for "+cmd.Name())
	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }}")

	cmd.Flags().StringVarP(&listener, "addr", "a", ":1331", "Listen address")

	return cmd
}
