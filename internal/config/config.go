package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/stellar/go-stellar-sdk/network"
)

// DefaultPublicDomainsRegistry is the Soroban Domains registry v2 contract
// on pubnet, as published by @creit-tech/sorobandomains-sdk.
const DefaultPublicDomainsRegistry = "CC75Z72OCE667WVPQOROIWDAGBOXFNJ4VQONQEURL74EYIDLWA4F7FEN"

type Config struct {
	DatabaseURL  string
	RedisURL     string
	RPCEndpoint  string
	DataLakePath string
	Network      string // "public", "testnet", "futurenet"
	BatchSize    int
	WorkerCount  int
	MetricsAddr  string // listen address for /metrics and /healthz; disabled when empty
	HTTPAddr     string // listen address for the read API (and metrics/healthz); disabled when empty

	// DomainsRegistryContractID is a comma-separated override for the Soroban
	// Domains registry contract ID(s). Empty means use the network default.
	DomainsRegistryContractID string
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:               getEnv("DATABASE_URL", "postgresql://explorer:explorer_dev@localhost:54320/stellar_explorer?sslmode=disable"),
		RedisURL:                  getEnv("REDIS_URL", "redis://localhost:63790"),
		RPCEndpoint:               getEnv("RPC_ENDPOINT", ""),
		DataLakePath:              getEnv("DATA_LAKE_PATH", "s3://aws-public-blockchain/v1.1/stellar/ledgers/pubnet"),
		Network:                   getEnv("NETWORK", "public"),
		BatchSize:                 getEnvInt("BATCH_SIZE", 100),
		WorkerCount:               getEnvInt("WORKER_COUNT", 8),
		MetricsAddr:               getEnv("METRICS_ADDR", ""),
		HTTPAddr:                  getEnv("HTTP_ADDR", ""),
		DomainsRegistryContractID: getEnv("DOMAINS_REGISTRY_CONTRACT_ID", ""),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	switch c.Network {
	case "public", "testnet", "futurenet":
		// valid
	default:
		return fmt.Errorf("invalid NETWORK %q: must be one of public, testnet, futurenet", c.Network)
	}

	if c.WorkerCount <= 0 {
		return fmt.Errorf("invalid WORKER_COUNT %d: must be > 0", c.WorkerCount)
	}

	if c.BatchSize <= 0 {
		return fmt.Errorf("invalid BATCH_SIZE %d: must be > 0", c.BatchSize)
	}

	return nil
}

// ListenAddr returns the HTTP bind address for metrics, health, and the
// domains read API. HTTP_ADDR takes precedence over METRICS_ADDR.
func (c *Config) ListenAddr() string {
	if c.HTTPAddr != "" {
		return c.HTTPAddr
	}
	return c.MetricsAddr
}

// RegistryContractIDs returns the Soroban Domains registry contract IDs to
// watch on the configured network. An explicit DOMAINS_REGISTRY_CONTRACT_ID
// override always wins; otherwise pubnet uses the SDK-published default and
// other networks require the override (no silent wrong-network default).
func (c *Config) RegistryContractIDs() []string {
	if c.DomainsRegistryContractID != "" {
		return splitCSV(c.DomainsRegistryContractID)
	}
	switch c.Network {
	case "public":
		return []string{DefaultPublicDomainsRegistry}
	default:
		return nil
	}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// NetworkPassphrase returns the Stellar network passphrase for the configured network.
func (c *Config) NetworkPassphrase() (string, error) {
	switch c.Network {
	case "public":
		return network.PublicNetworkPassphrase, nil
	case "testnet":
		return network.TestNetworkPassphrase, nil
	case "futurenet":
		return network.FutureNetworkPassphrase, nil
	default:
		return "", fmt.Errorf("unknown network: %s", c.Network)
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return fallback
}
