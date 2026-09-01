package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/Muxcore-Media/core/pkg/contracts"
	executorv1 "github.com/Muxcore-Media/executor-shell/muxcore/executor/v1"
)

const (
	defaultGRPCAddr    = ":9605"
	defaultTimeout     = 5 * time.Minute
	defaultMaxOutput   = 1 << 20 // 1 MiB
	envTimeout         = "EXECUTOR_TIMEOUT"
	envMaxOutput       = "EXECUTOR_MAX_OUTPUT"
	envAllowWildcard   = "EXECUTOR_ALLOW_WILDCARD"
	envAllowedCommands = "EXECUTOR_ALLOWED_COMMANDS"
	envGRPCAddr        = "EXECUTOR_GRPC_ADDR"
)

type Module struct {
	executorv1.UnimplementedExecutorServiceServer

	grpcSrv *grpc.Server
	lis     net.Listener

	mu              sync.RWMutex
	id              string
	grpcAddr        string
	allowedCommands string
	timeout         time.Duration
	maxOutput       int64
}

type Config struct {
	ID              string
	GRPCAddr        string
	AllowedCommands string
	Timeout         time.Duration
	MaxOutput       int64
}

func NewModule(cfg Config) *Module {
	if cfg.ID == "" {
		cfg.ID = "executor-shell"
	}
	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = defaultGRPCAddr
	}
	if v := os.Getenv(envGRPCAddr); v != "" {
		cfg.GRPCAddr = v
	}
	if v := os.Getenv(envAllowedCommands); v != "" {
		cfg.AllowedCommands = v
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = parseDurationEnv(envTimeout, defaultTimeout)
	}
	if cfg.MaxOutput <= 0 {
		cfg.MaxOutput = parseInt64Env(envMaxOutput, defaultMaxOutput)
	}
	return &Module{
		id:              cfg.ID,
		grpcAddr:        cfg.GRPCAddr,
		allowedCommands: cfg.AllowedCommands,
		timeout:         cfg.Timeout,
		maxOutput:       cfg.MaxOutput,
	}
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func parseInt64Env(key string, fallback int64) int64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	var n int64
	if _, err := fmt.Sscan(v, &n); err != nil || n <= 0 {
		return fallback
	}
	return n
}

func (m *Module) Info() contracts.ModuleInfo {
	return contracts.ModuleInfo{
		ID:           m.id,
		Name:         "Executor Shell",
		Version:      "0.1.4",
		Roles:        []string{"executor"},
		Description:  "Generic shell command executor for the worker pool",
		Author:       "MuxCore",
		Capabilities: []string{"executor.shell", "settings"},
		Contracts: []contracts.ContractDeclaration{
			{
				Repo:      "github.com/Muxcore-Media/core/pkg/contracts",
				Interface: "Executor",
				Version:   "v0.4.0",
			},
		},
		MinCoreVersion: "0.4.0",
		HTTPAddr:       m.grpcAddr,
	}
}

func (m *Module) Init(ctx context.Context) error {
	_ = ctx
	lis, err := net.Listen("tcp", m.grpcAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", m.grpcAddr, err)
	}
	m.lis = lis
	slog.Info("executor-shell initialized",
		"addr", m.grpcAddr,
		"allowed_commands", summarizeAllowlist(m.allowedCommands),
		"timeout", m.timeoutDuration().String(),
		"max_output", m.maxOutputBytes(),
	)
	return nil
}

func (m *Module) Start(ctx context.Context) error {
	_ = ctx
	m.grpcSrv = grpc.NewServer()
	executorv1.RegisterExecutorServiceServer(m.grpcSrv, m)
	RegisterExecutorMesh(m.grpcSrv, m.id, m)
	go func() {
		slog.Info("executor-shell gRPC service started", "addr", m.grpcAddr)
		if err := m.grpcSrv.Serve(m.lis); err != nil {
			slog.Error("executor-shell gRPC serve error", "error", err)
		}
	}()
	return nil
}

func (m *Module) Stop(ctx context.Context) error {
	_ = ctx
	if m.grpcSrv != nil {
		m.grpcSrv.GracefulStop()
	}
	slog.Info("executor-shell stopped")
	return nil
}

func (m *Module) Health(ctx context.Context) error {
	_ = ctx
	if m.lis == nil {
		return fmt.Errorf("not initialized")
	}
	return nil
}

func (m *Module) CanHandle(ctx context.Context, req *executorv1.CanHandleRequest) (*executorv1.CanHandleResponse, error) {
	_ = ctx
	return &executorv1.CanHandleResponse{
		CanHandle: req.GetTaskType() == "shell",
	}, nil
}

func envFlagTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func (m *Module) timeoutDuration() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.timeout
}

func (m *Module) maxOutputBytes() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.maxOutput
}

func (m *Module) commandAllowed(payload string) bool {
	args, err := splitShellArgs(payload)
	if err != nil {
		return false
	}
	first := args[0]

	m.mu.RLock()
	raw := strings.TrimSpace(m.allowedCommands)
	m.mu.RUnlock()
	if raw == "" {
		return false
	}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if entry == "*" {
			if !envFlagTruthy(os.Getenv(envAllowWildcard)) {
				slog.Warn("executor-shell: wildcard allowlist rejected; set EXECUTOR_ALLOW_WILDCARD=1 to enable")
				continue
			}
			return true
		}
		if first == entry {
			return true
		}
	}
	return false
}

func (m *Module) executionContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	timeout := m.timeoutDuration()
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func (m *Module) Execute(ctx context.Context, req *executorv1.ExecuteRequest) (*executorv1.ExecuteResponse, error) {
	return m.executePayload(ctx, string(req.GetPayload()))
}

func (m *Module) executePayload(ctx context.Context, payload string) (*executorv1.ExecuteResponse, error) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return &executorv1.ExecuteResponse{
			Success: false,
			Error:   "empty command payload",
		}, nil
	}

	m.mu.RLock()
	allowed := m.allowedCommands
	m.mu.RUnlock()
	if !m.commandAllowed(payload) {
		slog.Warn("rejected command not in allowed list", "command", redactCommandForLog(payload))
		errMsg := "command not allowed: empty allowlist denies all; set EXECUTOR_ALLOWED_COMMANDS to command names or *"
		if strings.TrimSpace(allowed) != "" {
			errMsg = "command not allowed: first token must match one of the configured allowed commands"
		}
		return &executorv1.ExecuteResponse{
			Success: false,
			Error:   errMsg,
		}, nil
	}

	args, err := splitShellArgs(payload)
	if err != nil {
		return &executorv1.ExecuteResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	runCtx, cancel := m.executionContext(ctx)
	defer cancel()

	cmd := exec.CommandContext(runCtx, args[0], args[1:]...)
	stdout := &limitedBuffer{limit: m.maxOutputBytes()}
	stderr := &limitedBuffer{limit: m.maxOutputBytes()}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		output := stderr.String()
		if output == "" {
			output = err.Error()
		}
		if stdout.overflow || stderr.overflow {
			output = "output limit exceeded: " + output
		}
		return &executorv1.ExecuteResponse{
			Success: false,
			Result:  stdout.Bytes(),
			Error:   output,
		}, nil
	}

	if stdout.overflow || stderr.overflow {
		return &executorv1.ExecuteResponse{
			Success: false,
			Result:  stdout.Bytes(),
			Error:   "output limit exceeded",
		}, nil
	}

	return &executorv1.ExecuteResponse{
		Success: true,
		Result:  stdout.Bytes(),
	}, nil
}

type limitedBuffer struct {
	buf      bytes.Buffer
	limit    int64
	overflow bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.overflow {
		return 0, io.ErrShortWrite
	}
	remaining := b.limit - int64(b.buf.Len())
	if remaining <= 0 {
		b.overflow = true
		return 0, io.ErrShortWrite
	}
	if int64(len(p)) > remaining {
		_, _ = b.buf.Write(p[:remaining])
		b.overflow = true
		return len(p[:remaining]), io.ErrShortWrite
	}
	return b.buf.Write(p)
}

func (b *limitedBuffer) Bytes() []byte {
	return b.buf.Bytes()
}

func (b *limitedBuffer) String() string {
	return b.buf.String()
}

func allowlistContainsWildcard(raw string) bool {
	for _, part := range strings.Split(raw, ",") {
		if strings.TrimSpace(part) == "*" {
			return true
		}
	}
	return false
}

func validateAllowedCommandsValue(value string) error {
	if allowlistContainsWildcard(value) && !envFlagTruthy(os.Getenv(envAllowWildcard)) {
		return fmt.Errorf("wildcard * requires EXECUTOR_ALLOW_WILDCARD=1")
	}
	return nil
}
