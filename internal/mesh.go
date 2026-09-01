package internal

import (
	"context"
	"encoding/json"
	"fmt"

	meshv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/mesh/v1"
	"github.com/Muxcore-Media/core/pkg/contracts"
	modulesdk "github.com/Muxcore-Media/core/sdk/go/module"
	"google.golang.org/grpc"
)

const (
	meshMethodSettings      = "Settings"
	meshMethodUpdateSetting = "UpdateSetting"
	meshMethodExecute       = "Execute"
)

type executorMeshServer struct {
	meshv1.UnimplementedModuleMeshServer
	moduleID string
	mod      *Module
	settings modulesdk.SettingsHandler
}

// RegisterExecutorMesh wires ModuleMesh Settings, UpdateSetting, and worker-pool Execute.
func RegisterExecutorMesh(srv *grpc.Server, moduleID string, mod *Module) {
	meshv1.RegisterModuleMeshServer(srv, &executorMeshServer{
		moduleID: moduleID,
		mod:      mod,
		settings: modulesdk.SettingsHandlerFromProvider(mod),
	})
}

func (s *executorMeshServer) Call(ctx context.Context, req *meshv1.CallRequest) (*meshv1.CallResponse, error) {
	if req.GetTargetModule() != "" && req.GetTargetModule() != s.moduleID {
		return &meshv1.CallResponse{Error: fmt.Sprintf("wrong target module %q", req.GetTargetModule())}, nil
	}
	switch req.GetMethod() {
	case meshMethodExecute:
		return s.handleExecute(ctx, req.GetPayload())
	case meshMethodSettings:
		return s.callSettings(ctx, req)
	case meshMethodUpdateSetting:
		return s.callUpdateSetting(ctx, req)
	default:
		return &meshv1.CallResponse{Error: fmt.Sprintf("unknown method %q", req.GetMethod())}, nil
	}
}

func (s *executorMeshServer) callSettings(ctx context.Context, req *meshv1.CallRequest) (*meshv1.CallResponse, error) {
	_ = ctx
	if s.settings.List == nil {
		return &meshv1.CallResponse{Error: "settings list not implemented"}, nil
	}
	raw, err := json.Marshal(s.settings.List())
	if err != nil {
		return &meshv1.CallResponse{Error: err.Error()}, nil
	}
	return &meshv1.CallResponse{Payload: raw}, nil
}

func (s *executorMeshServer) callUpdateSetting(ctx context.Context, req *meshv1.CallRequest) (*meshv1.CallResponse, error) {
	_ = ctx
	if s.settings.Update == nil {
		return &meshv1.CallResponse{Error: "settings update not implemented"}, nil
	}
	var body struct {
		Key   string `json:"Key"`
		Value string `json:"Value"`
	}
	if len(req.GetPayload()) > 0 {
		if err := json.Unmarshal(req.GetPayload(), &body); err != nil {
			return &meshv1.CallResponse{Error: fmt.Sprintf("invalid UpdateSetting payload: %v", err)}, nil
		}
	}
	if body.Key == "" {
		return &meshv1.CallResponse{Error: "UpdateSetting requires Key"}, nil
	}
	if err := s.settings.Update(body.Key, body.Value); err != nil {
		return &meshv1.CallResponse{Error: err.Error()}, nil
	}
	return &meshv1.CallResponse{Payload: []byte(`{"ok":true}`)}, nil
}

func (s *executorMeshServer) handleExecute(ctx context.Context, payload []byte) (*meshv1.CallResponse, error) {
	var task contracts.WorkerTask
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &task); err != nil {
			return &meshv1.CallResponse{Error: fmt.Sprintf("invalid WorkerTask payload: %v", err)}, nil
		}
	}
	resp, err := s.mod.executePayload(ctx, string(task.Payload))
	if err != nil {
		return &meshv1.CallResponse{Error: err.Error()}, nil
	}
	if !resp.GetSuccess() {
		return &meshv1.CallResponse{Error: resp.GetError()}, nil
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		return &meshv1.CallResponse{Error: err.Error()}, nil
	}
	return &meshv1.CallResponse{Payload: raw}, nil
}

func (s *executorMeshServer) StreamCall(stream meshv1.ModuleMesh_StreamCallServer) error {
	return fmt.Errorf("StreamCall not supported for executor-shell mesh handler")
}
