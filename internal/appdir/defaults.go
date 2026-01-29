package appdir

import (
	_ "embed"
	"fmt"
	"os"
)

//go:embed default_config.yaml
var defaultConfigYAML []byte

// writeDefaultConfig записывает дефолтный конфиг в указанный путь.
func writeDefaultConfig(path string) error {
	if err := os.WriteFile(path, defaultConfigYAML, 0644); err != nil {
		return fmt.Errorf("write default config: %w", err)
	}
	return nil
}

// DefaultConfigYAML возвращает содержимое дефолтного конфига.
func DefaultConfigYAML() []byte {
	return defaultConfigYAML
}
