// Package identity предоставляет работу с ed25519 ключами.
package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// KeyPair содержит пару ed25519 ключей.
type KeyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// Generate создаёт новую пару ключей.
func Generate() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key: %w", err)
	}
	return &KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// LoadFromFile загружает ключи из файла.
// Файл должен содержать 64 байта приватного ключа в raw формате.
func LoadFromFile(path string) (*KeyPair, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	if len(data) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid key file size: expected %d, got %d", ed25519.PrivateKeySize, len(data))
	}

	priv := ed25519.PrivateKey(data)
	pub := priv.Public().(ed25519.PublicKey)

	return &KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// SaveToFile сохраняет приватный ключ в файл.
func (k *KeyPair) SaveToFile(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create key directory: %w", err)
	}

	if err := os.WriteFile(path, k.PrivateKey, 0600); err != nil {
		return fmt.Errorf("write key file: %w", err)
	}
	return nil
}

// LoadOrGenerate загружает ключи из файла или генерирует новые.
func LoadOrGenerate(path string) (*KeyPair, error) {
	// Сначала пробуем загрузить существующий файл
	kp, err := LoadFromFile(path)
	if err == nil {
		return kp, nil
	}

	// Если файл не существует — генерируем новый
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	kp, err = Generate()
	if err != nil {
		return nil, err
	}

	if err := kp.SaveToFile(path); err != nil {
		return nil, err
	}

	return kp, nil
}

// PublicKeyHex возвращает публичный ключ в hex-encoded формате.
func (k *KeyPair) PublicKeyHex() string {
	return hex.EncodeToString(k.PublicKey)
}

// Sign подписывает данные приватным ключом.
func (k *KeyPair) Sign(data []byte) []byte {
	return ed25519.Sign(k.PrivateKey, data)
}
