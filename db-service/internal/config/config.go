package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Name     string `yaml:"name"`
	} `yaml:"database"`

	Server struct {
		GRPCPort string `yaml:"grpc_port"`
	} `yaml:"server"`
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
		cfg.Database.Host = "localhost"
		cfg.Database.Port = 5433
		cfg.Database.User = "postgres"
		cfg.Database.Password = "postgres"
		cfg.Database.Name = "microservices_db"
		cfg.Server.GRPCPort = ":50052"
		return cfg
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		log.Fatalf("[Config] Failed to parse config.yaml: %v", err)
	}

	return cfg
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.Database.Host, c.Database.Port, c.Database.User, c.Database.Password, c.Database.Name)
}

func (c *Config) GRPCPort() string {
	if c.Server.GRPCPort == "" {
		return ":50052"
	}
	return c.Server.GRPCPort
}
