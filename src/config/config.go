package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Network  NetworkConfig  `yaml:"network"`
	Snapshot SnapshotConfig `yaml:"snapshot"`
	WAL      WALConfig      `yaml:"wal"`
	Redis    RedisConfig    `yaml:"redis"`
	Logger   LoggerConfig   `yaml:"logger"`
}

type LoggerConfig struct {
	Level string `yaml:"level"`
}

type PeerConfig struct {
	ID      string `yaml:"id"`
	Address string `yaml:"address"`
}

type NetworkConfig struct {
	Self  PeerConfig   `yaml:",inline"`
	Peers []PeerConfig `yaml:"peers"`
}

type SnapshotConfig struct {
	Path      string `yaml:"path"`
	Interval  int    `yaml:"interval"`  // in seconds
	Threshold int64  `yaml:"threshold"` // number of bytes
}

type WALConfig struct {
	Path string `yaml:"path"`
}

type RedisConfig struct {
	Host                     string `yaml:"host"`
	Port                     int    `yaml:"port"`
	Timeout                  int    `yaml:"timeout"`                     // in seconds
	MaxConnections           int    `yaml:"max_connections"`             // max concurrent connections (workers)
	MaxPending               int    `yaml:"max_pending"`                 // max pending connections in queue
	MaxMessageSize           int64  `yaml:"max_message_size"`            // max bulk string size in bytes
	BaseWorkers              int    `yaml:"base_workers"`                // number of idle workers to keep alive
	WorkerTTL                int    `yaml:"worker_ttl"`                  // in seconds
	IdleConnectionsPerWorker int    `yaml:"idle_connections_per_worker"` // idle connections per worker threshold
}

func DefaultConfig() *Config {
	return &Config{
		Snapshot: SnapshotConfig{
			Path:      ".data/snapshot.db",
			Interval:  3600,
			Threshold: 1024 * 1024, // 1MB
		},
		WAL: WALConfig{
			Path: ".data/wal.log",
		},
		Redis: RedisConfig{
			Host:                     "localhost",
			Port:                     6379,
			Timeout:                  30,
			MaxConnections:           100,
			MaxPending:               1000,
			MaxMessageSize:           10 * 1024 * 1024, // 10MB
			BaseWorkers:              10,
			WorkerTTL:                10,
			IdleConnectionsPerWorker: 3,
		},
		Network: NetworkConfig{
			Self: PeerConfig{
				ID:      "self",
				Address: "localhost:5000",
			},
			Peers: []PeerConfig{
				{
					ID:      "self",
					Address: "localhost:5000",
				},
			},
		},
		Logger: LoggerConfig{
			Level: "INFO",
		},
	}
}

// LoadConfig loads configuration from a list of files.
// Files are processed in order, so later files override values from earlier ones.
// For example, LoadConfig("config.yaml", "override.yaml") will load config.yaml first,
// then apply any overrides from override.yaml.
func LoadConfig(files ...string) (*Config, error) {
	cfg := DefaultConfig()

	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			continue
		}

		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		decoder := yaml.NewDecoder(f)
		if err := decoder.Decode(cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}
