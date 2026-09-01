package internal

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Muxcore-Media/core/pkg/contracts"
	meshv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/mesh/v1"
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
	if info.HTTPAddr != defaultGRPCAddr {
		t.Errorf("HTTPAddr = %q, want %q", info.HTTPAddr, defaultGRPCAddr)
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
	t.Setenv(envAllowWildcard, "1")
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

func TestExecuteEmptyPayload(t *testing.T) {
	t.Setenv(envAllowWildcard, "1")
	m := NewModule(Config{AllowedCommands: "*"})
	resp, err := m.Execute(context.Background(), &executorv1.ExecuteRequest{Payload: nil})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if resp.GetSuccess() {
		t.Fatal("expected failure for empty payload")
	}
	if !strings.Contains(resp.GetError(), "empty command payload") {
		t.Fatalf("unexpected error: %q", resp.GetError())
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
	t.Setenv(envAllowWildcard, "1")
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

func TestExecuteStarDeniedWithoutFlag(t *testing.T) {
	t.Setenv(envAllowWildcard, "")
	m := NewModule(Config{AllowedCommands: "*"})
	resp, err := m.Execute(context.Background(), &executorv1.ExecuteRequest{
		Payload: []byte("echo ok"),
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if resp.GetSuccess() {
		t.Fatal("expected * denied without EXECUTOR_ALLOW_WILDCARD")
	}
}

func TestExecuteFirstTokenAllowlist(t *testing.T) {
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

func TestExecutePrefixInjectionDenied(t *testing.T) {
	m := NewModule(Config{AllowedCommands: "echo"})
	cases := []string{
		"echo hi; rm -rf /",
		"echo hi | cat /etc/passwd",
		"echo hi && rm -rf /",
		"echo $(whoami)",
		"echo `id`",
	}
	for _, payload := range cases {
		resp, err := m.Execute(context.Background(), &executorv1.ExecuteRequest{Payload: []byte(payload)})
		if err != nil {
			t.Fatalf("payload=%q: %v", payload, err)
		}
		if resp.GetSuccess() {
			t.Fatalf("payload=%q should be denied", payload)
		}
	}
}

func TestExecutePrefixBypassDenied(t *testing.T) {
	m := NewModule(Config{AllowedCommands: "echo"})
	resp, err := m.Execute(context.Background(), &executorv1.ExecuteRequest{
		Payload: []byte("echohi"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetSuccess() {
		t.Fatal("expected echohi denied (not a prefix match anymore)")
	}
}

func TestCommandAllowed(t *testing.T) {
	cases := []struct {
		allow   string
		payload string
		want    bool
	}{
		{"", "echo", false},
		{"echo", "echo hi", true},
		{"echo", "ls", false},
		{"git,make", "make test", true},
		{" , ", "echo", false},
		{"echo", "echo hi; rm -rf /", false},
	}
	for _, tc := range cases {
		m := NewModule(Config{AllowedCommands: tc.allow})
		if got := m.commandAllowed(tc.payload); got != tc.want {
			t.Fatalf("allow=%q payload=%q got %v want %v", tc.allow, tc.payload, got, tc.want)
		}
	}
	t.Setenv(envAllowWildcard, "1")
	m := NewModule(Config{AllowedCommands: "*"})
	if got := m.commandAllowed("anything"); !got {
		t.Fatalf("wildcard allowlist with EXECUTOR_ALLOW_WILDCARD=1: got %v want true", got)
	}
}

func TestExecuteNonZeroExitUsesStderr(t *testing.T) {
	m := NewModule(Config{AllowedCommands: "cat"})
	resp, err := m.Execute(context.Background(), &executorv1.ExecuteRequest{
		Payload: []byte("cat /nonexistent/path/for/muxcore/test"),
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if resp.GetSuccess() {
		t.Fatal("expected failure on non-zero exit")
	}
	if !strings.Contains(resp.GetError(), "No such file") {
		t.Fatalf("expected stderr in error, got %q", resp.GetError())
	}
}

func TestExecuteOutputLimit(t *testing.T) {
	t.Setenv(envAllowWildcard, "1")
	m := NewModule(Config{AllowedCommands: "*", MaxOutput: 32})
	resp, err := m.Execute(context.Background(), &executorv1.ExecuteRequest{
		Payload: []byte("dd if=/dev/zero bs=1 count=64"),
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if resp.GetSuccess() {
		t.Fatal("expected failure on output overflow")
	}
	if !strings.Contains(resp.GetError(), "output limit exceeded") {
		t.Fatalf("unexpected error: %q", resp.GetError())
	}
}

func TestExecuteDefaultTimeout(t *testing.T) {
	t.Setenv(envAllowWildcard, "1")
	m := NewModule(Config{AllowedCommands: "*", Timeout: 50 * time.Millisecond})
	start := time.Now()
	resp, err := m.Execute(context.Background(), &executorv1.ExecuteRequest{
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

func TestExecuteContextTimeout(t *testing.T) {
	t.Setenv(envAllowWildcard, "1")
	m := NewModule(Config{AllowedCommands: "*", Timeout: time.Hour})
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
	t.Setenv(envAllowWildcard, "1")
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

func TestUpdateSettingUnknownKey(t *testing.T) {
	m := NewModule(Config{})
	if err := m.UpdateSetting("nope", "x"); err == nil {
		t.Fatal("expected error for unknown setting")
	}
}

func TestUpdateSettingWildcardDenied(t *testing.T) {
	t.Setenv(envAllowWildcard, "")
	m := NewModule(Config{})
	if err := m.UpdateSetting("allowed_commands", "*"); err == nil {
		t.Fatal("expected wildcard update denied without EXECUTOR_ALLOW_WILDCARD")
	}
}

func TestSettingsAllowedCommandsLive(t *testing.T) {
	m := NewModule(Config{})
	defs := m.Settings()
	if len(defs) != 2 || defs[0].Key != "allowed_commands" {
		t.Fatalf("Settings=%+v", defs)
	}
	resp, err := m.Execute(context.Background(), &executorv1.ExecuteRequest{Payload: []byte("echo hi")})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetSuccess() {
		t.Fatal("empty allowlist should deny")
	}
	if err := m.UpdateSetting("allowed_commands", "echo"); err != nil {
		t.Fatal(err)
	}
	resp, err = m.Execute(context.Background(), &executorv1.ExecuteRequest{Payload: []byte("echo hi")})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("expected success after allowlist update: %s", resp.GetError())
	}
}

func TestMeshExecuteWorkerTask(t *testing.T) {
	t.Setenv(envAllowWildcard, "1")
	m := NewModule(Config{AllowedCommands: "*"})
	srv := &executorMeshServer{moduleID: "executor-shell", mod: m}

	raw, err := json.Marshal(contracts.WorkerTask{
		Type:    "shell",
		Payload: []byte("echo mesh"),
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := srv.Call(context.Background(), &meshv1.CallRequest{
		TargetModule: "executor-shell",
		Method:       meshMethodExecute,
		Payload:      raw,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetError() != "" {
		t.Fatalf("mesh Execute error: %s", resp.GetError())
	}
	var got executorv1.ExecuteResponse
	if err := json.Unmarshal(resp.GetPayload(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.GetSuccess() {
		t.Fatalf("expected success: %s", got.GetError())
	}
	if strings.TrimSpace(string(got.GetResult())) != "mesh" {
		t.Fatalf("result: %q", got.GetResult())
	}
}

func TestMeshExecuteUnknownMethod(t *testing.T) {
	m := NewModule(Config{})
	srv := &executorMeshServer{moduleID: "executor-shell", mod: m}
	resp, err := srv.Call(context.Background(), &meshv1.CallRequest{
		TargetModule: "executor-shell",
		Method:       "Nope",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.GetError(), "unknown method") {
		t.Fatalf("got %q", resp.GetError())
	}
}

func TestSplitShellArgsQuoted(t *testing.T) {
	t.Setenv(envAllowWildcard, "1")
	m := NewModule(Config{AllowedCommands: "echo"})
	resp, err := m.Execute(context.Background(), &executorv1.ExecuteRequest{
		Payload: []byte(`echo "hello world"`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("quoted args failed: %s", resp.GetError())
	}
	if strings.TrimSpace(string(resp.GetResult())) != "hello world" {
		t.Fatalf("result: %q", resp.GetResult())
	}
}
