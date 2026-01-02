package shim

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/frantjc/forge/githubactions"
	xerrors "github.com/frantjc/x/errors"
	xos "github.com/frantjc/x/os"
	"github.com/spf13/cobra"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	err := xerrors.Ignore(newShim().ExecuteContext(ctx), context.Canceled)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	stop()
	xos.ExitFromError(err)
}

func newShim() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "shim",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}

			globalContext, err := githubactions.NewGlobalContextFromPath(wd)
			if err != nil {
				globalContext = githubactions.NewGlobalContextFromEnv()
			}

			subcmd := exec.CommandContext(cmd.Context(), args[0], args[1:]...) //nolint:gosec
			subcmd.Dir = globalContext.GitHubContext.Workspace
			subcmd.Env = append(os.Environ(), globalContext.Env()...)
			subcmd.Stdin = cmd.InOrStdin()
			subcmd.Stdout = githubactions.NewWorkflowCommandWriter(cmd.OutOrStdout(), globalContext)
			subcmd.Stderr = cmd.ErrOrStderr()

			return subcmd.Run()
		},
	}

	cmd.Flags().BoolP("help", "h", false, "Help for "+cmd.Name())

	return cmd
}
