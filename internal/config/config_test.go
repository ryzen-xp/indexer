package config

import (
	"os"
	"testing"
)

func TestLoad_RPCEndpointOptional(t *testing.T) {
	os.Unsetenv("RPC_ENDPOINT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RPCEndpoint != "" {
		t.Errorf("expected empty RPCEndpoint, got '%s'", cfg.RPCEndpoint)
	}
}

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Network != "public" {
		t.Errorf("expected network 'public', got '%s'", cfg.Network)
	}
	if cfg.BatchSize != 100 {
		t.Errorf("expected batch size 100, got %d", cfg.BatchSize)
	}
	if cfg.WorkerCount != 8 {
		t.Errorf("expected worker count 8, got %d", cfg.WorkerCount)
	}
	if cfg.MetricsAddr != "" {
		t.Errorf("expected metrics server disabled by default, got MetricsAddr=%q", cfg.MetricsAddr)
	}
}

func TestLoad_HTTPAddrOptIn(t *testing.T) {
	os.Setenv("HTTP_ADDR", ":8080")
	defer os.Unsetenv("HTTP_ADDR")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("expected HTTPAddr ':8080', got %q", cfg.HTTPAddr)
	}
	if cfg.ListenAddr() != ":8080" {
		t.Errorf("expected ListenAddr ':8080', got %q", cfg.ListenAddr())
	}
}

func TestLoad_ListenAddrPrefersHTTPAddr(t *testing.T) {
	os.Setenv("HTTP_ADDR", ":8080")
	os.Setenv("METRICS_ADDR", ":9090")
	defer func() {
		os.Unsetenv("HTTP_ADDR")
		os.Unsetenv("METRICS_ADDR")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ListenAddr() != ":8080" {
		t.Errorf("expected HTTP_ADDR to win, got %q", cfg.ListenAddr())
	}
}

func TestRegistryContractIDs_PublicDefault(t *testing.T) {
	os.Unsetenv("DOMAINS_REGISTRY_CONTRACT_ID")
	os.Unsetenv("NETWORK")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ids := cfg.RegistryContractIDs()
	if len(ids) != 1 || ids[0] != DefaultPublicDomainsRegistry {
		t.Errorf("public default registry IDs = %v, want [%s]", ids, DefaultPublicDomainsRegistry)
	}
}

func TestRegistryContractIDs_TestnetRequiresOverride(t *testing.T) {
	os.Setenv("NETWORK", "testnet")
	os.Unsetenv("DOMAINS_REGISTRY_CONTRACT_ID")
	defer os.Unsetenv("NETWORK")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ids := cfg.RegistryContractIDs(); len(ids) != 0 {
		t.Errorf("testnet without override should have no registry IDs, got %v", ids)
	}
}

func TestRegistryContractIDs_Override(t *testing.T) {
	os.Setenv("NETWORK", "testnet")
	os.Setenv("DOMAINS_REGISTRY_CONTRACT_ID", " CC75Z72OCE667WVPQOROIWDAGBOXFNJ4VQONQEURL74EYIDLWA4F7FEN , CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4 ")
	defer func() {
		os.Unsetenv("NETWORK")
		os.Unsetenv("DOMAINS_REGISTRY_CONTRACT_ID")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ids := cfg.RegistryContractIDs()
	if len(ids) != 2 {
		t.Fatalf("override IDs = %v, want 2 entries", ids)
	}
	if ids[0] != DefaultPublicDomainsRegistry {
		t.Errorf("ids[0] = %s", ids[0])
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	os.Setenv("NETWORK", "testnet")
	os.Setenv("BATCH_SIZE", "200")
	defer func() {
		os.Unsetenv("NETWORK")
		os.Unsetenv("BATCH_SIZE")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Network != "testnet" {
		t.Errorf("expected network 'testnet', got '%s'", cfg.Network)
	}
	if cfg.BatchSize != 200 {
		t.Errorf("expected batch size 200, got %d", cfg.BatchSize)
	}
}

func TestLoad_InvalidNetwork(t *testing.T) {
	os.Setenv("NETWORK", "mainnet")
	defer os.Unsetenv("NETWORK")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid NETWORK, got nil")
	}
}

func TestLoad_ZeroWorkerCount(t *testing.T) {
	os.Setenv("WORKER_COUNT", "0")
	defer os.Unsetenv("WORKER_COUNT")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for WORKER_COUNT=0, got nil")
	}
}

func TestLoad_NegativeWorkerCount(t *testing.T) {
	os.Setenv("WORKER_COUNT", "-1")
	defer os.Unsetenv("WORKER_COUNT")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for WORKER_COUNT=-1, got nil")
	}
}

func TestLoad_ZeroBatchSize(t *testing.T) {
	os.Setenv("BATCH_SIZE", "0")
	defer os.Unsetenv("BATCH_SIZE")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for BATCH_SIZE=0, got nil")
	}
}

func TestLoad_NegativeBatchSize(t *testing.T) {
	os.Setenv("BATCH_SIZE", "-5")
	defer os.Unsetenv("BATCH_SIZE")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for BATCH_SIZE=-5, got nil")
	}
}
