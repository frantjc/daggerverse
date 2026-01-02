package command

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"dagger.io/dagger/telemetry"
	"github.com/containerd/log"
	"github.com/frantjc/daggerverse/dogger/internal/dogger/internal/backends"
	"github.com/frantjc/daggerverse/dogger/internal/dogger/internal/handler"
	"github.com/frantjc/daggerverse/dogger/internal/dogger/internal/logutil"
	sqlstorage "github.com/frantjc/daggerverse/dogger/internal/dogger/internal/storage/sql"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func NewDogger(version string) *cobra.Command {
	var (
		address string
		slogCfg = new(logutil.SlogConfig)
		cmd     = &cobra.Command{
			Use:           "dogger",
			SilenceErrors: true,
			SilenceUsage:  true,
			Version:       version,
			RunE: func(cmd *cobra.Command, args []string) error {
				eg, ctx := errgroup.WithContext(
					logutil.SloggerInto(
						telemetry.InitEmbedded(cmd.Context(), nil),
						slog.New(slog.NewJSONHandler(cmd.OutOrStdout(), &slog.HandlerOptions{
							Level: slogCfg,
						})), 
					),
				)

				if !strings.Contains(address, "://") {
					// Default to listening on ip:port.
					address = fmt.Sprintf("tcp://%s", address)
				}

				a, err := url.Parse(address)
				if err != nil {
					return err
				}

				lis, err := net.Listen(a.Scheme, strings.TrimPrefix(a.String(), fmt.Sprintf("%s://", a.Scheme)))
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

				level, _, _ := strings.Cut(strings.ToLower(slogCfg.Level().String()), "+")
				if err := log.SetLevel(level); err != nil {
					return err
				}
				log.SetFormat(log.JSONFormat)

				srv := &http.Server{
					ReadHeaderTimeout: time.Second * 5,
					Handler: handler.New().
						// TODO(frantjc): Implement all of these.
						WithSystemBackend(&backends.SystemBackend{}, nil, nil).
						WithImageBackend(&backends.ImageBackend{Storage: store}, nil).
						WithContainerBackend(&backends.ContainerBackend{Context: ctx, Storage: store}).
						Handler(ctx, false),
					BaseContext: func(_ net.Listener) context.Context {
						return ctx // log.WithLogger(ctx, log.L)
					},
				}
				log := logutil.SloggerFrom(ctx)

				eg.Go(func() error {
					log.Info("listening...", "addr", lis.Addr().String())
					return srv.Serve(lis)
				})

				eg.Go(func() error {
					<-ctx.Done()
					log.Info("shutting down")
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
	slogCfg.AddFlags(cmd.Flags())
	cmd.Flags().StringVarP(&address, "addr", "a", ":1331", "Listen address")

	return cmd
}
