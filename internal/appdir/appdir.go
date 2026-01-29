// Package appdir управляет директорией приложения с XDG-совместимыми путями.
package appdir

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

const appName = "sprut"

// Dir возвращает путь к директории приложения.
// Linux: ~/.config/sprut
// macOS: ~/Library/Application Support/sprut
// Windows: %AppData%\sprut
func Dir() string {
	return filepath.Join(xdg.ConfigHome, appName)
}

// ConfigPath возвращает путь к файлу конфигурации.
func ConfigPath() string {
	return filepath.Join(Dir(), "config.yaml")
}

// CertsDir возвращает путь к директории сертификатов.
func CertsDir() string {
	return filepath.Join(Dir(), "certs")
}

// LogsDir возвращает путь к директории логов.
func LogsDir() string {
	return filepath.Join(Dir(), "logs")
}

// CertPath возвращает путь к файлу сертификата.
func CertPath() string {
	return filepath.Join(CertsDir(), "server.crt")
}

// KeyPath возвращает путь к файлу ключа.
func KeyPath() string {
	return filepath.Join(CertsDir(), "server.key")
}

// LogFilePath возвращает путь к файлу логов.
func LogFilePath() string {
	return filepath.Join(LogsDir(), "sprut.log")
}

// Init инициализирует директорию приложения.
// Создаёт все необходимые поддиректории, дефолтный конфиг и сертификаты.
func Init() error {
	// Создаём директории
	dirs := []string{Dir(), CertsDir(), LogsDir()}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// Создаём дефолтный конфиг если его нет
	if err := ensureDefaultConfig(); err != nil {
		return fmt.Errorf("ensure default config: %w", err)
	}

	// Генерируем сертификаты если их нет
	if err := ensureCerts(); err != nil {
		return fmt.Errorf("ensure certificates: %w", err)
	}

	return nil
}

// ensureDefaultConfig создаёт дефолтный конфиг если его нет.
func ensureDefaultConfig() error {
	configPath := ConfigPath()

	if _, err := os.Stat(configPath); err == nil {
		// Конфиг уже существует
		return nil
	}

	return writeDefaultConfig(configPath)
}

// ensureCerts генерирует сертификаты если их нет.
func ensureCerts() error {
	certPath := CertPath()
	keyPath := KeyPath()

	// Проверяем существование обоих файлов
	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)

	if certErr == nil && keyErr == nil {
		// Оба файла существуют
		return nil
	}

	// Генерируем новые сертификаты
	return generateSelfSignedCert(certPath, keyPath)
}
