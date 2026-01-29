package identity

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerate(t *testing.T) {
	kp, err := Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if len(kp.PublicKey) != ed25519.PublicKeySize {
		t.Errorf("public key size: got %d, want %d", len(kp.PublicKey), ed25519.PublicKeySize)
	}
	if len(kp.PrivateKey) != ed25519.PrivateKeySize {
		t.Errorf("private key size: got %d, want %d", len(kp.PrivateKey), ed25519.PrivateKeySize)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.key")

	original, err := Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if err := original.SaveToFile(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if !original.PublicKey.Equal(loaded.PublicKey) {
		t.Error("public keys don't match")
	}
	if !original.PrivateKey.Equal(loaded.PrivateKey) {
		t.Error("private keys don't match")
	}
}

func TestLoadOrGenerate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys", "test.key")

	// Первый вызов — генерирует
	kp1, err := LoadOrGenerate(path)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Второй вызов — загружает
	kp2, err := LoadOrGenerate(path)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if !kp1.PublicKey.Equal(kp2.PublicKey) {
		t.Error("should return same keys")
	}
}

func TestLoadInvalidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.key")

	// Записываем файл неправильного размера
	if err := os.WriteFile(path, []byte("too short"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := LoadFromFile(path)
	if err == nil {
		t.Error("expected error for invalid file")
	}
}

func TestPublicKeyHex(t *testing.T) {
	kp, err := Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	hex := kp.PublicKeyHex()
	if len(hex) != ed25519.PublicKeySize*2 {
		t.Errorf("hex length: got %d, want %d", len(hex), ed25519.PublicKeySize*2)
	}
}

func TestSign(t *testing.T) {
	kp, err := Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	data := []byte("test message")
	sig := kp.Sign(data)

	if len(sig) != ed25519.SignatureSize {
		t.Errorf("signature size: got %d, want %d", len(sig), ed25519.SignatureSize)
	}

	if !ed25519.Verify(kp.PublicKey, data, sig) {
		t.Error("signature verification failed")
	}
}
