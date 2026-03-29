package ipc

import (
	"awesomeProject/internal/daemon/info"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type stubController struct{}

func (stubController) ToggleProcessJail(pid int) (bool, error) {
	if pid == 0 {
		return false, errors.New("invalid pid")
	}
	return true, nil
}

func TestClientServerRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "awesomer.sock")
	state := info.NewAPI()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := Start(ctx, path, state, stubController{})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "operation not permitted") {
			t.Skipf("unix sockets are not available in this sandbox: %v", err)
		}
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = server.Close() }()

	client := NewClient(path)
	deadline := time.Now().Add(time.Second)
	for {
		if err := client.Ping(); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("client ping timed out")
		}
		time.Sleep(10 * time.Millisecond)
	}

	client.SetJail(101)
	client.SetJail(202)

	if !client.InJail(101) {
		t.Fatal("InJail(101) = false, want true")
	}
	if len(client.PIDs()) != 2 {
		t.Fatalf("len(PIDs()) = %d, want 2", len(client.PIDs()))
	}

	client.DeleteFromJail(101)
	if client.InJail(101) {
		t.Fatal("InJail(101) = true, want false")
	}

	inJail, err := client.ToggleProcessJail(202)
	if err != nil {
		t.Fatalf("ToggleProcessJail() error = %v", err)
	}
	if !inJail {
		t.Fatal("ToggleProcessJail() = false, want true")
	}
}
