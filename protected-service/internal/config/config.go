package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		HTTPPort string `yaml:"http_port"`
	} `yaml:"server"`

	Services struct {
		AuthHTTPURL  string `yaml:"auth_http_url"`
		AuthGRPCAddr string `yaml:"auth_grpc_addr"`
		RedisAddr    string `yaml:"redis_addr"`
	} `yaml:"services"`
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
		cfg.Server.HTTPPort = ":8082"
		cfg.Services.AuthHTTPURL = "http://localhost:8081"
		cfg.Services.AuthGRPCAddr = "localhost:50051"
		cfg.Services.RedisAddr = "localhost:6379"
		return cfg
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		log.Fatalf("[Config] Failed to parse config.yaml: %v", err)
	}

	return cfg
}

func (c *Config) HTTPPort() string {
	if c.Server.HTTPPort == "" {
		return ":8082"
	}
	return c.Server.HTTPPort
}

func (c *Config) AuthHTTPURL() string {
	if c.Services.AuthHTTPURL == "" {
		return "http://localhost:8081"
	}
	return c.Services.AuthHTTPURL
}

func (c *Config) AuthGRPCAddr() string {
	if c.Services.AuthGRPCAddr == "" {
		return "localhost:50051"
	}
	return c.Services.AuthGRPCAddr
}

func (c *Config) RedisAddr() string {
	if c.Services.RedisAddr == "" {
		return "localhost:6379"
	}
	return c.Services.RedisAddr
}
