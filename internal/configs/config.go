package configs

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTPAddr string  `yaml:"http_addr"`
	Database Database `yaml:"database"`
	Zap      Zap      `yaml:"zap"`
	Redis    Redis    `yaml:"redis"`
	Auth     Auth     `yaml:"auth"`
}

type Database struct {
	Path string `yaml:"path"`
}

type Zap struct {
	Image          string `yaml:"image"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type Redis struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type Auth struct {
	AccessSecret  string `yaml:"access_secret"`
	RefreshSecret string `yaml:"refresh_secret"`
	AccessTTL     string `yaml:"access_ttl"`
	RefreshTTL    string `yaml:"refresh_ttl"`
}

func Load() Config {
	cfg := defaultConfig()

	// Load from YAML file if exists
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		if err := loadFromYAML(path, &cfg); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to load config from %s: %v\n", path, err)
		}
	} else if _, err := os.Stat("config.yaml"); err == nil {
		if err := loadFromYAML("config.yaml", &cfg); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to load config from config.yaml: %v\n", err)
		}
	}

	// Environment variables override YAML and defaults
	overrideFromEnv(&cfg)

	return cfg
}

func defaultConfig() Config {
	return Config{
		HTTPAddr: ":9090",
		Database: Database{Path: "webtester.db"},
		Zap: Zap{
			Image:          "ghcr.io/zaproxy/zaproxy:stable",
			TimeoutSeconds: 1800,
		},
		Redis: Redis{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		},
		Auth: Auth{
			AccessSecret:  "",
			RefreshSecret: "",
			AccessTTL:     "15m",
			RefreshTTL:    "168h",
		},
	}
}

func loadFromYAML(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, cfg)
}

func overrideFromEnv(cfg *Config) {
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("ZAP_IMAGE"); v != "" {
		cfg.Zap.Image = v
	}
	if v := os.Getenv("ZAP_TIMEOUT_SECONDS"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Zap.TimeoutSeconds)
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Redis.DB)
	}
	if v := os.Getenv("AUTH_ACCESS_SECRET"); v != "" {
		cfg.Auth.AccessSecret = v
	}
	if v := os.Getenv("AUTH_REFRESH_SECRET"); v != "" {
		cfg.Auth.RefreshSecret = v
	}
	if v := os.Getenv("AUTH_ACCESS_TTL"); v != "" {
		cfg.Auth.AccessTTL = v
	}
	if v := os.Getenv("AUTH_REFRESH_TTL"); v != "" {
		cfg.Auth.RefreshTTL = v
	}
}