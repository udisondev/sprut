package router

import (
	"testing"

	"github.com/udisondev/sprut/pkg/protocol"
)

func TestIsValidHexPubKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid lowercase hex",
			input: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			want:  true,
		},
		{
			name:  "valid uppercase hex",
			input: "0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
			want:  true,
		},
		{
			name:  "valid mixed case hex",
			input: "0123456789AbCdEf0123456789AbCdEf0123456789AbCdEf0123456789AbCdEf",
			want:  true,
		},
		{
			name:  "NATS wildcard asterisk",
			input: "*123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			want:  false,
		},
		{
			name:  "NATS wildcard greater than",
			input: ">123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			want:  false,
		},
		{
			name:  "NATS subject separator dot",
			input: "0123456789abcdef.123456789abcdef0123456789abcdef0123456789abcdef",
			want:  false,
		},
		{
			name:  "too short",
			input: "0123456789abcdef",
			want:  false,
		},
		{
			name:  "too long",
			input: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef00",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "contains space",
			input: "0123456789abcdef 123456789abcdef0123456789abcdef0123456789abcdef",
			want:  false,
		},
		{
			name:  "contains invalid char g",
			input: "0123456789abcdeg0123456789abcdef0123456789abcdef0123456789abcdef",
			want:  false,
		},
		{
			name:  "all wildcards",
			input: "****************************************************************",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidHexPubKey(tt.input)
			if got != tt.want {
				t.Errorf("isValidHexPubKey(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidHexPubKey_Length(t *testing.T) {
	// Проверяем что функция ожидает ровно 64 символа (2 * PublicKeySize)
	expectedLen := protocol.PublicKeySize * 2
	if expectedLen != 64 {
		t.Errorf("expected key length to be 64, got %d", expectedLen)
	}
}
