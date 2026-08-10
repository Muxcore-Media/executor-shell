package internal

import (
	"context"
	"strings"
	"testing"
	"time"

	executorv1 "github.com/Muxcore-Media/executor-shell/muxcore/executor/v1"
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
	m := NewModule(Config{GRPCAddr: ":0", AllowedCommands: "*"})
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

func TestExecuteEmptyAllowlistDenies(t *testing.T) {
	m := NewModule(Config{AllowedCommands: ""})
	resp, err := m.Execute(context.Background(), &executorv1.ExecuteRequest{
		Payload: []byte("echo hi"),
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if resp.GetSuccess() {
		t.Fatal("expected deny with empty allowlist")
	}
	if !strings.Contains(resp.GetError(), "empty allowlist") {
		t.Fatalf("unexpected error: %q", resp.GetError())
	}
}

func TestExecuteStarAllows(t *testing.T) {
	m := NewModule(Config{AllowedCommands: "*"})
	resp, err := m.Execute(context.Background(), &executorv1.ExecuteRequest{
		Payload: []byte("echo ok"),
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("expected success with *: %s", resp.GetError())
	}
	if strings.TrimSpace(string(resp.GetResult())) != "ok" {
		t.Fatalf("result: %q", resp.GetResult())
	}
}

func TestExecutePrefixAllowlist(t *testing.T) {
	m := NewModule(Config{AllowedCommands: "echo,true"})
	ctx := context.Background()

	ok, err := m.Execute(ctx, &executorv1.ExecuteRequest{Payload: []byte("echo allowed")})
	if err != nil {
		t.Fatal(err)
	}
	if !ok.GetSuccess() {
		t.Fatalf("expected echo allowed: %s", ok.GetError())
	}

	denied, err := m.Execute(ctx, &executorv1.ExecuteRequest{Payload: []byte("ls")})
	if err != nil {
		t.Fatal(err)
	}
	if denied.GetSuccess() {
		t.Fatal("expected ls denied")
	}
}

func TestCommandAllowed(t *testing.T) {
	cases := []struct {
		allow   string
		payload string
		want    bool
	}{
		{"", "echo", false},
		{"*", "anything", true},
		{"echo", "echo hi", true},
		{"echo", "ls", false},
		{"git,make", "make test", true},
		{" , ", "echo", false},
	}
	for _, tc := range cases {
		m := NewModule(Config{AllowedCommands: tc.allow})
		if got := m.commandAllowed(tc.payload); got != tc.want {
			t.Fatalf("allow=%q payload=%q got %v want %v", tc.allow, tc.payload, got, tc.want)
		}
	}
}

func TestExecuteContextTimeout(t *testing.T) {
	m := NewModule(Config{AllowedCommands: "*"})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	resp, err := m.Execute(ctx, &executorv1.ExecuteRequest{
		Payload: []byte("sleep 5"),
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if resp.GetSuccess() {
		t.Fatal("expected failure on timeout")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("command not cancelled promptly: %v", elapsed)
	}
}

func TestExecuteContextCancel(t *testing.T) {
	m := NewModule(Config{AllowedCommands: "*"})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		resp, err := m.Execute(ctx, &executorv1.ExecuteRequest{
			Payload: []byte("sleep 5"),
		})
		if err != nil {
			t.Errorf("Execute error: %v", err)
			return
		}
		if resp.GetSuccess() {
			t.Error("expected failure after cancel")
		}
	}()
	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Execute did not return after cancel")
	}
}
