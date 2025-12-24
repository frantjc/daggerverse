package command_test

import (
	"fmt"
	"net"
	"os/exec"
	"testing"
	"time"

	"github.com/frantjc/daggerverse/dogger/internal/dogger/command"
	"github.com/stretchr/testify/assert"
	_ "github.com/mattn/go-sqlite3"
)

func TestDogger(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	addr := lis.Addr().String()
	assert.NoError(t, lis.Close())

	go func() {
		dogger := command.NewDogger("v0.0.0-test")
		dogger.SetArgs([]string{"--addr", addr})
		assert.NoError(t,
			dogger.ExecuteContext(t.Context()),
		)
	}()

	for {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			assert.NoError(t, conn.Close())
			break
		}
	}

	docker := exec.CommandContext(t.Context(), "docker", "run", "busybox", "echo", "hello")
	docker.Env = append(docker.Env, fmt.Sprintf("DOCKER_HOST=%s", addr))
	docker.Stdout = t.Output()
	docker.Stderr = t.Output()
	assert.NoError(t,
		docker.Run(),
	)
}
