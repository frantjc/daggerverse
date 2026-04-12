package command_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/frantjc/daggerverse/dogger/internal/dogger/command"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestDogger(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	addr := lis.Addr().String()
	assert.NoError(t, lis.Close())

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	go func() {
		dogger := command.NewDogger("v0.0.0-test")
		dogger.SetOut(t.Output())
		dogger.SetErr(t.Output())
		dogger.SetArgs([]string{"--addr", addr})
		assert.NoError(t,
			dogger.ExecuteContext(ctx),
		)
	}()

	for {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			assert.NoError(t, conn.Close())
			break
		}
	}

	expected := uuid.NewString()
	docker := exec.CommandContext(ctx, "docker", "run", "busybox", "echo", expected)
	docker.Env = append(docker.Env, fmt.Sprintf("DOCKER_HOST=%s", addr))
	buf := new(bytes.Buffer)
	docker.Stdout = io.MultiWriter(t.Output(), buf)
	docker.Stderr = t.Output()
	assert.NoError(t,docker.Run())
	actual := strings.TrimSpace(buf.String())
	assert.Equal(t, expected, actual)
}
