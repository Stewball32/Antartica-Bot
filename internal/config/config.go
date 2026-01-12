package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Discord DiscordConfig `yaml:"discord"`
	Dev     DevConfig     `yaml:"dev"`
}

type DiscordConfig struct {
	ClientID string `yaml:"client_id"`
	Secret   string `yaml:"secret"`
	Token    string `yaml:"token"`
}

type DevConfig struct {
	Enabled bool   `yaml:"enabled"`
	GuildID string `yaml:"guild_id"`
}

func Load(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}
