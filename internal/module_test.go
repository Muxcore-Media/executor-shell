package internal

import (
	"context"
	"testing"

	executorv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/executor/v1"
)

func TestModuleInfo(t *testing.T) {
	m := NewModule(Config{})
	info := m.Info()
	if info.ID == "" {
		t.Error("module ID must not be empty")
	}
	if info.Version == "" {
		t.Error("module version must not be empty")
	}
	if info.MinCoreVersion == "" {
		t.Error("MinCoreVersion must not be empty")
	}
	if len(info.Contracts) == 0 {
		t.Error("Contracts must not be empty")
	}
	if info.Contracts[0].Interface != "Executor" {
		t.Errorf("expected Executor contract, got %s", info.Contracts[0].Interface)
	}
	if len(info.Capabilities) == 0 || info.Capabilities[0] != "executor.shell" {
		t.Errorf("expected capability executor.shell, got %v", info.Capabilities)
	}
}

func TestCanHandle(t *testing.T) {
	m := NewModule(Config{})

	resp, err := m.CanHandle(context.Background(), &executorv1.CanHandleRequest{TaskType: "shell"})
	if err != nil {
		t.Fatalf("CanHandle error: %v", err)
	}
	if !resp.GetCanHandle() {
		t.Error("expected CanHandle to return true for shell")
	}

	resp, err = m.CanHandle(context.Background(), &executorv1.CanHandleRequest{TaskType: "download"})
	if err != nil {
		t.Fatalf("CanHandle error: %v", err)
	}
	if resp.GetCanHandle() {
		t.Error("expected CanHandle to return false for non-shell task type")
	}
}

func TestModuleLifecycle(t *testing.T) {
	m := NewModule(Config{GRPCAddr: ":0"})
	ctx := context.Background()

	if err := m.Init(ctx); err != nil {
		t.Fatalf("Init error: %v", err)
	}
	if err := m.Health(ctx); err != nil {
		t.Errorf("Health check failed after Init: %v", err)
	}
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if err := m.Stop(ctx); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
}
