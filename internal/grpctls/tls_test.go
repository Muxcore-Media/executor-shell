package grpctls_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Muxcore-Media/executor-shell/internal/grpctls"
)

func TestServerConfigTLSByDefault(t *testing.T) {
	t.Setenv("MUXCORE_INSECURE_DISABLE_TLS", "")
	t.Setenv("MUXCORE_GRPC_INSECURE", "")
	dir := t.TempDir()
	t.Setenv("EXECUTOR_TLS_DIR", dir)

	cfg, err := grpctls.ServerConfig()
	if err != nil {
		t.Fatalf("ServerConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected TLS config by default")
	}
	if len(cfg.Certificates) != 1 {
		t.Fatalf("expected one certificate, got %d", len(cfg.Certificates))
	}
	for _, name := range []string{"server.crt", "server.key", "ca.crt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}

func TestServerConfigInsecureDisable(t *testing.T) {
	t.Setenv("MUXCORE_INSECURE_DISABLE_TLS", "true")
	cfg, err := grpctls.ServerConfig()
	if err != nil {
		t.Fatalf("ServerConfig: %v", err)
	}
	if cfg != nil {
		t.Fatal("expected nil TLS config when insecure is enabled")
	}
}
