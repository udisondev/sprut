package config

import (
	"fmt"
	"os"

	"github.com/udisondev/sprut/internal/appdir"
	"gopkg.in/yaml.v3"
)

// Load загружает конфигурацию из файла.
func Load(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	resolvePaths(cfg)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

// LoadFromAppDir загружает конфигурацию из XDG директории приложения.
func LoadFromAppDir() (*Config, error) {
	return Load(appdir.ConfigPath())
}

// resolvePaths подставляет дефолтные пути для пустых значений.
func resolvePaths(cfg *Config) {
	// TLS сертификаты
	if cfg.TLS.CertFile == "" {
		cfg.TLS.CertFile = appdir.CertPath()
	}
	if cfg.TLS.KeyFile == "" {
		cfg.TLS.KeyFile = appdir.KeyPath()
	}

	// Файл логов
	if cfg.Log.File == "" {
		cfg.Log.File = appdir.LogFilePath()
	}
}
