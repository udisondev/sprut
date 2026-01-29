package protocol

import (
	"crypto/tls"
	"fmt"
)

// ChannelBindingLabel метка для TLS channel binding (RFC 5705).
const ChannelBindingLabel = "EXPORTER-goro-auth-v1"

// GetChannelBinding извлекает TLS channel binding из состояния соединения.
// Используется tls-exporter согласно RFC 5705.
func GetChannelBinding(state tls.ConnectionState) ([ChannelBindingSize]byte, error) {
	var binding [ChannelBindingSize]byte

	if !state.HandshakeComplete {
		return binding, fmt.Errorf("TLS handshake not complete")
	}

	exported, err := state.ExportKeyingMaterial(ChannelBindingLabel, nil, ChannelBindingSize)
	if err != nil {
		return binding, fmt.Errorf("export keying material: %w", err)
	}

	copy(binding[:], exported)
	return binding, nil
}
