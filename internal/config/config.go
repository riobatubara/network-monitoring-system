package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Collector CollectorConfig `mapstructure:"collector"`
}

type ServerConfig struct {
	GRPCPort int      `mapstructure:"grpc_port"`
	DB       DBConfig `mapstructure:"db"`
}

type DBConfig struct {
	PostgresDSN        string `mapstructure:"postgres_dsn"`
	RedisAddr          string `mapstructure:"redis_addr"`
	VictoriaMetricsURL string `mapstructure:"victoriametrics_url"`
}

type CollectorConfig struct {
	ID                 string `mapstructure:"id"`
	ServerAddress      string `mapstructure:"server_address"`
	WorkerCount        int    `mapstructure:"worker_count"`
	BufferSize         int    `mapstructure:"buffer_size"`
	RateLimitMs        int    `mapstructure:"rate_limit_ms"`
	PollingIntervalSec int    `mapstructure:"polling_interval_sec"`
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv() // Allows environment variables to override files (e.g., SERVER_DB_REDIS_ADDR)

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
