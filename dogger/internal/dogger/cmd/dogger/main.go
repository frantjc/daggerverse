package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/frantjc/daggerverse/dogger/internal/dogger/command"
	xerrors "github.com/frantjc/x/errors"
	xos "github.com/frantjc/x/os"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var (
		ctx, stop = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		cmd       = command.NewDogger(SemVer())
	)

	err := xerrors.Ignore(cmd.ExecuteContext(ctx), context.Canceled)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	stop()
	xos.ExitFromError(err)
}
