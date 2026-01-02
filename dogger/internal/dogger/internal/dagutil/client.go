package dagutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"dagger.io/dagger"
	"github.com/frantjc/daggerverse/dogger/internal/dogger/internal/logutil"
	"github.com/vito/go-sse/sse"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type LogCallback func(*collogspb.ExportLogsServiceRequest)

type TraceCallback func(*coltracepb.ExportTraceServiceRequest)

func Connect(ctx context.Context, l LogCallback, t TraceCallback) (*dagger.Client, error) {
	log := logutil.SloggerFrom(ctx)

	if rawDaggerSessionPort := os.Getenv("DAGGER_SESSION_PORT"); rawDaggerSessionPort != "" {
		daggerSessionPort, err := strconv.Atoi(rawDaggerSessionPort)
		if err != nil {
			return nil, err
		}

		if l != nil {
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
				return nil, err
			}
			defer logEventSource.Close()

			go func() {
				for {
					e, err := logEventSource.Next()
					if err != nil {
						if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
							return
						}
						log.Error(err.Error())
						continue
					} else if e.Name == "attached" {
						continue
					}

					req := collogspb.ExportLogsServiceRequest{}
					if err := protojson.Unmarshal(e.Data, &req); err != nil {
						log.Error(err.Error())
						continue
					}

					l(&req)
				}
			}()
		}

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
			return nil, err
		}
		defer traceEventSource.Close()

		go func() {
			for {
				e, err := traceEventSource.Next()
					if err != nil {
						if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
							return
						}
						log.Error(err.Error())
						continue
					} else if e.Name == "attached" {
						continue
					}

				req := coltracepb.ExportTraceServiceRequest{}
				if err := protojson.Unmarshal(e.Data, &req); err != nil {
					log.Error(err.Error())
					continue
				}

				t(&req)
			}
		}()

		return dagger.Connect(ctx)
	}

	mux := http.NewServeMux()
	opts := []dagger.ClientOpt{}

	otelLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	if l != nil {
		mux.HandleFunc("POST /v1/logs", func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				log.Error(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			req := collogspb.ExportLogsServiceRequest{}
			if err := proto.Unmarshal(body, &req); err != nil {
				log.Error(err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			l(&req)
		})

		opts = append(opts,
			dagger.WithEnvironmentVariable("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", "http://"+otelLis.Addr().String()+"/v1/logs"),
			dagger.WithEnvironmentVariable("OTEL_EXPORTER_OTLP_LOGS_PROTOCOL", "http/protobuf"),
		)
	}

		mux.HandleFunc("POST /v1/traces", func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				log.Error(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			req := coltracepb.ExportTraceServiceRequest{}
			if err := proto.Unmarshal(body, &req); err != nil {
				log.Error(err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			t(&req)
		})

		opts = append(opts,
			dagger.WithEnvironmentVariable("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "http://"+otelLis.Addr().String()+"/v1/traces"),
			dagger.WithEnvironmentVariable("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "http/protobuf"),
		)
	
	eg, ctx := errgroup.WithContext(ctx)

	otelSrv := &http.Server{
		ReadHeaderTimeout: time.Second * 5,
		Handler:           mux,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
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

	return dagger.Connect(ctx, opts...)
}
