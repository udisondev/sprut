package router

import (
	"testing"

	"github.com/udisondev/sprut/pkg/protocol"
)

func BenchmarkIsValidHexPubKey(b *testing.B) {
	// Валидный hex публичный ключ (64 символа = 32 байта)
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	b.ReportAllocs()

	for b.Loop() {
		_ = isValidHexPubKey(key)
	}
}

func BenchmarkIsValidHexPubKey_Invalid(b *testing.B) {
	// Невалидный ключ (содержит 'g')
	key := "0123456789abcdef0123456789abcdefg123456789abcdef0123456789abcdef"

	b.ReportAllocs()

	for b.Loop() {
		_ = isValidHexPubKey(key)
	}
}

func BenchmarkIsValidHexPubKey_WrongLength(b *testing.B) {
	// Неправильная длина
	key := "0123456789abcdef"

	b.ReportAllocs()

	for b.Loop() {
		_ = isValidHexPubKey(key)
	}
}

// BenchmarkIsValidHexPubKey_Parallel проверяет производительность при параллельном доступе.
func BenchmarkIsValidHexPubKey_Parallel(b *testing.B) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = isValidHexPubKey(key)
		}
	})
}

// BenchmarkMinMessageSizeConstant проверяет, что константа вычисляется корректно.
func BenchmarkMinMessageSizeConstant(b *testing.B) {
	expected := protocol.PublicKeySize*2 + 2

	b.ReportAllocs()

	for b.Loop() {
		if minMessageSize != expected {
			b.Fatal("minMessageSize mismatch")
		}
	}
}
