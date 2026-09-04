// Package server wires configuration and the Wish SSH server.
package server

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"ieee-ctf/internal/models"
)

// Config mirrors configs/server.yaml.
type Config struct {
	Server struct {
		Host        string `yaml:"host"`
		Port        string `yaml:"port"`
		HostKeyPath string `yaml:"host_key_path"`
	} `yaml:"server"`

	Database struct {
		URL           string `yaml:"url"`
		AuthToken     string `yaml:"auth_token"`
		Path          string `yaml:"path"`
		MigrationsDir string `yaml:"migrations_dir"`
	} `yaml:"database"`

	Security struct {
		RateLimit struct {
			RequestsPerSecond float64 `yaml:"requests_per_second"`
			Burst             int     `yaml:"burst"`
		} `yaml:"rate_limit"`
		Sessions struct {
			MaxGlobal int `yaml:"max_global"`
			MaxPerIP  int `yaml:"max_per_ip"`
		} `yaml:"sessions"`
		Timeouts struct {
			MaxSessionMinutes  int `yaml:"max_session_minutes"`
			IdleTimeoutMinutes int `yaml:"idle_timeout_minutes"`
		} `yaml:"timeouts"`
		Submissions struct {
			PerMinutePerTeam float64 `yaml:"per_minute_per_team"`
			CooldownSeconds  int     `yaml:"cooldown_seconds"`
		} `yaml:"submissions"`
		Input struct {
			MaxFlagLength       int `yaml:"max_flag_length"`
			MaxPGPMessageLength int `yaml:"max_pgp_message_length"`
			MaxTeamNameLength   int `yaml:"max_team_name_length"`
		} `yaml:"input"`
	} `yaml:"security"`

	Gameplay struct {
		AllowHints *bool `yaml:"allow_hints"`
		AllowSkips *bool `yaml:"allow_skips"`
	} `yaml:"gameplay"`

	Hints struct {
		ChallengeValidityMinutes int `yaml:"challenge_validity_minutes"`
	} `yaml:"hints"`
}

// Load reads server.yaml, applies defaults and env overrides.
func Load(path string) (*Config, error) {
	cfg := &Config{}
	if path != "" {
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config %s: %w", path, err)
		}
		if err := yaml.Unmarshal(body, cfg); err != nil {
			return nil, fmt.Errorf("parse config %s: %w", path, err)
		}
	}
	applyDefaults(cfg)

	// Environment overrides.
	if v := os.Getenv("CTF_DB_URL"); v != "" {
		cfg.Database.URL = v
	} else if v := os.Getenv("TURSO_DATABASE_URL"); v != "" {
		cfg.Database.URL = v
	}
	if v := os.Getenv("CTF_DB_AUTH_TOKEN"); v != "" {
		cfg.Database.AuthToken = v
	} else if v := os.Getenv("TURSO_AUTH_TOKEN"); v != "" {
		cfg.Database.AuthToken = v
	} else if v := os.Getenv("LIBSQL_AUTH_TOKEN"); v != "" {
		cfg.Database.AuthToken = v
	}
	if v := os.Getenv("CTF_DB_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("CTF_HOST_KEY"); v != "" {
		cfg.Server.HostKeyPath = v
	}
	return cfg, nil
}

// DatabaseTarget returns the configured database URL/path and authentication token.
func (c *Config) DatabaseTarget() (string, string) {
	if c.Database.URL != "" {
		return c.Database.URL, c.Database.AuthToken
	}
	return c.Database.Path, c.Database.AuthToken
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == "" {
		cfg.Server.Port = "2222"
	}
	if cfg.Server.HostKeyPath == "" {
		cfg.Server.HostKeyPath = ".ssh/id_ed25519"
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = "ctf.db"
	}
	if cfg.Database.MigrationsDir == "" {
		cfg.Database.MigrationsDir = "migrations"
	}
	s := &cfg.Security
	if s.RateLimit.RequestsPerSecond <= 0 {
		s.RateLimit.RequestsPerSecond = 1.0
	}
	if s.RateLimit.Burst <= 0 {
		s.RateLimit.Burst = 5
	}
	if s.Sessions.MaxGlobal <= 0 {
		s.Sessions.MaxGlobal = 100
	}
	if s.Sessions.MaxPerIP <= 0 {
		s.Sessions.MaxPerIP = 3
	}
	if s.Timeouts.MaxSessionMinutes <= 0 {
		s.Timeouts.MaxSessionMinutes = 30
	}
	if s.Timeouts.IdleTimeoutMinutes <= 0 {
		s.Timeouts.IdleTimeoutMinutes = 5
	}
	if s.Submissions.PerMinutePerTeam <= 0 {
		s.Submissions.PerMinutePerTeam = 2
	}
	if s.Submissions.CooldownSeconds <= 0 {
		s.Submissions.CooldownSeconds = 30
	}
	if s.Input.MaxFlagLength <= 0 {
		s.Input.MaxFlagLength = 256
	}
	if s.Input.MaxPGPMessageLength <= 0 {
		s.Input.MaxPGPMessageLength = 8192
	}
	if s.Input.MaxTeamNameLength <= 0 {
		s.Input.MaxTeamNameLength = 32
	}
	if cfg.Hints.ChallengeValidityMinutes <= 0 {
		cfg.Hints.ChallengeValidityMinutes = 10
	}
}

// LoadRounds parses configs/rounds.yaml.
func LoadRounds(path string) (*models.RoundsConfig, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rounds config %s: %w", path, err)
	}
	rc, err := models.ParseRounds(body)
	if err != nil {
		return nil, fmt.Errorf("parse rounds config %s: %w", path, err)
	}
	return rc, nil
}
