// Package protocol определяет wire protocol для goro.
package protocol

// Типы сообщений аутентификации
const (
	TypeClientHello     byte = 0x01
	TypeServerChallenge byte = 0x02
	TypeClientResponse  byte = 0x03
	TypeAuthResult      byte = 0x04
)

// Размеры полей
const (
	PublicKeySize      = 32
	ChallengeSize      = 32
	TimestampSize      = 8
	ServerIDSize       = 32
	SignatureSize      = 64
	ChannelBindingSize = 32
)

// ChannelBinding — alias для обратной совместимости.
// Deprecated: используйте ChannelBindingSize.
const ChannelBinding = ChannelBindingSize

// Статусы аутентификации
const (
	AuthStatusOK         byte = 0x00
	AuthStatusInvalidSig byte = 0x02
	AuthStatusReplay     byte = 0x03
)

// Версия протокола для подписи
const ProtocolVersion = "goro-auth-v1"

// Максимальные размеры
const (
	MaxMessageSize  = 65536        // 64KB
	MaxMsgIDLen     = 256
	MaxErrorMsgLen  = 1024
)
