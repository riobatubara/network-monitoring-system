package config

import (
	"github.com/spf13/viper"
)

type SchedulerConfig struct {
	GRPCPort int      `mapstructure:"grpc_port"`
	DB       DBConfig `mapstructure:"db"`
}

type CollectorConfig struct {
	ID                 string `mapstructure:"id"`
	SchedulerAddress   string `mapstructure:"scheduler_address"`
	WorkerCount        int    `mapstructure:"worker_count"`
	BufferSize         int    `mapstructure:"buffer_size"`
	RateLimitMs        int    `mapstructure:"rate_limit_ms"`
	PollingIntervalSec int    `mapstructure:"polling_interval_sec"`
}

type Config struct {
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
	Collector CollectorConfig `mapstructure:"collector"`
}

type DBConfig struct {
	PostgresDSN        string `mapstructure:"postgres_dsn"`
	RedisAddr          string `mapstructure:"redis_addr"`
	VictoriaMetricsURL string `mapstructure:"victoriametrics_url"`
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv() // Lets environment variables safely override file keys inside Docker containers

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
