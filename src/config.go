package src

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Snapshot SnapshotConfig `yaml:"snapshot"`
	WAL      WALConfig      `yaml:"wal"`
	Redis    RedisConfig    `yaml:"redis"`
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
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Timeout int    `yaml:"timeout"` // in seconds
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
			Host:    "localhost",
			Port:    6379,
			Timeout: 30,
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
