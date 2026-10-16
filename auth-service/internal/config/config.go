package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		HTTPPort string `yaml:"http_port"`
		GRPCPort string `yaml:"grpc_port"`
	} `yaml:"server"`

	Services struct {
		DBServiceAddr string `yaml:"db_service_addr"`
		RedisAddr     string `yaml:"redis_addr"`
	} `yaml:"services"`

	JWT struct {
		Issuer                  string `yaml:"issuer"`
		AccessExpirationMinutes int    `yaml:"access_expiration_minutes"`
		RefreshExpirationDays   int    `yaml:"refresh_expiration_days"`
	} `yaml:"jwt"`
}

func LoadConfig() *Config {
	cfg := &Config{}

	configPaths := []string{"config/config.yaml", "../config/config.yaml", "config.yaml"}
	var data []byte
	var err error

	for _, path := range configPaths {
		data, err = os.ReadFile(path)
		if err == nil {
			log.Printf("[Config] Loaded YAML configuration from %s", path)
			break
		}
	}

	if err != nil {
		log.Println("[Config] WARNING: Could not find config.yaml, applying default fallback settings")
		cfg.Server.HTTPPort = ":8081"
		cfg.Server.GRPCPort = ":50051"
		cfg.Services.DBServiceAddr = "localhost:50052"
		cfg.Services.RedisAddr = "localhost:6379"
		cfg.JWT.Issuer = "auth-service"
		cfg.JWT.AccessExpirationMinutes = 15
		cfg.JWT.RefreshExpirationDays = 7
		return cfg
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		log.Fatalf("[Config] Failed to parse config.yaml: %v", err)
	}

	return cfg
}

func (c *Config) HTTPPort() string {
	if c.Server.HTTPPort == "" {
		return ":8081"
	}
	return c.Server.HTTPPort
}

func (c *Config) GRPCPort() string {
	if c.Server.GRPCPort == "" {
		return ":50051"
	}
	return c.Server.GRPCPort
}

func (c *Config) DBServiceAddr() string {
	if c.Services.DBServiceAddr == "" {
		return "localhost:50052"
	}
	return c.Services.DBServiceAddr
}

func (c *Config) RedisAddr() string {
	if c.Services.RedisAddr == "" {
		return "localhost:6379"
	}
	return c.Services.RedisAddr
}

func (c *Config) JWTIssuer() string {
	if c.JWT.Issuer == "" {
		return "auth-service"
	}
	return c.JWT.Issuer
}

func (c *Config) AccessExpirationMinutes() int {
	if c.JWT.AccessExpirationMinutes <= 0 {
		return 15
	}
	return c.JWT.AccessExpirationMinutes
}

func (c *Config) RefreshExpirationDays() int {
	if c.JWT.RefreshExpirationDays <= 0 {
		return 7
	}
	return c.JWT.RefreshExpirationDays
}
