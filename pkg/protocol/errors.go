package protocol

import "errors"

var (
	// ErrAuthFailed — общая ошибка аутентификации.
	ErrAuthFailed = errors.New("authentication failed")

	// ErrInvalidSignature — неверная подпись.
	ErrInvalidSignature = errors.New("invalid signature")

	// ErrChallengeExpired — challenge истёк (replay attack protection).
	ErrChallengeExpired = errors.New("challenge expired")

	// ErrConnectionClosed — соединение закрыто.
	ErrConnectionClosed = errors.New("connection closed")
)
