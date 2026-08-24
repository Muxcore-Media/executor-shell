package internal

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"

	"google.golang.org/grpc"

	"github.com/Muxcore-Media/core/pkg/contracts"
	modulesdk "github.com/Muxcore-Media/core/sdk/go/module"
	executorv1 "github.com/Muxcore-Media/executor-shell/muxcore/executor/v1"
)

type Module struct {
	executorv1.UnimplementedExecutorServiceServer

	grpcSrv *grpc.Server
	lis     net.Listener

	mu              sync.RWMutex
	id              string
	grpcAddr        string
	allowedCommands string
}

type Config struct {
	ID              string
	GRPCAddr        string
	AllowedCommands string
}

func NewModule(cfg Config) *Module {
	if cfg.ID == "" {
		cfg.ID = "executor-shell"
	}
	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = ":9602"
	}
	if v := os.Getenv("EXECUTOR_GRPC_ADDR"); v != "" {
		cfg.GRPCAddr = v
	}
	if v := os.Getenv("EXECUTOR_ALLOWED_COMMANDS"); v != "" {
		cfg.AllowedCommands = v
	}
	return &Module{
		id:              cfg.ID,
		grpcAddr:        cfg.GRPCAddr,
		allowedCommands: cfg.AllowedCommands,
	}
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
	lis, err := net.Listen("tcp", m.grpcAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", m.grpcAddr, err)
	}
	m.lis = lis
	slog.Info("executor-shell initialized", "addr", m.grpcAddr, "allowed_commands", m.allowedCommands)
	return nil
}

func (m *Module) Start(ctx context.Context) error {
	m.grpcSrv = grpc.NewServer()
	executorv1.RegisterExecutorServiceServer(m.grpcSrv, m)
	modulesdk.RegisterSettings(m.grpcSrv, m.id, m)
	go func() {
		slog.Info("executor-shell gRPC service started", "addr", m.grpcAddr)
		if err := m.grpcSrv.Serve(m.lis); err != nil {
			slog.Error("executor-shell gRPC serve error", "error", err)
		}
	}()
	return nil
}

func (m *Module) Stop(ctx context.Context) error {
	if m.grpcSrv != nil {
		m.grpcSrv.GracefulStop()
	}
	slog.Info("executor-shell stopped")
	return nil
}

func (m *Module) Health(ctx context.Context) error {
	if m.lis == nil {
		return fmt.Errorf("not initialized")
	}
	return nil
}

func (m *Module) CanHandle(ctx context.Context, req *executorv1.CanHandleRequest) (*executorv1.CanHandleResponse, error) {
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

func (m *Module) commandAllowed(payload string) bool {
	m.mu.RLock()
	raw := strings.TrimSpace(m.allowedCommands)
	m.mu.RUnlock()
	if raw == "" {
		return false
	}
	for _, prefix := range strings.Split(raw, ",") {
		prefix = strings.TrimSpace(prefix)
		if prefix == "" {
			continue
		}
		if prefix == "*" {
			if !envFlagTruthy(os.Getenv("EXECUTOR_ALLOW_WILDCARD")) {
				slog.Warn("executor-shell: wildcard allowlist rejected; set EXECUTOR_ALLOW_WILDCARD=1 to enable")
				continue
			}
			return true
		}
		if strings.HasPrefix(payload, prefix) {
			return true
		}
	}
	return false
}

func (m *Module) Execute(ctx context.Context, req *executorv1.ExecuteRequest) (*executorv1.ExecuteResponse, error) {
	payload := string(req.GetPayload())
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
		slog.Warn("rejected command not in allowed list", "command", payload)
		errMsg := "command not allowed: empty allowlist denies all; set EXECUTOR_ALLOWED_COMMANDS to prefixes or *"
		if strings.TrimSpace(allowed) != "" {
			errMsg = fmt.Sprintf("command not allowed: must start with one of: %s", allowed)
		}
		return &executorv1.ExecuteResponse{
			Success: false,
			Error:   errMsg,
		}, nil
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		output := stderr.String()
		if output == "" {
			output = err.Error()
		}
		return &executorv1.ExecuteResponse{
			Success: false,
			Result:  stdout.Bytes(),
			Error:   output,
		}, nil
	}

	return &executorv1.ExecuteResponse{
		Success: true,
		Result:  stdout.Bytes(),
	}, nil
}
