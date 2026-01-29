package client

import (
	"crypto/tls"
	"fmt"

	"github.com/udisondev/sprut/pkg/identity"
	"github.com/udisondev/sprut/pkg/protocol"
)

// signChallenge подписывает challenge от сервера.
func signChallenge(keys *identity.KeyPair, challenge *protocol.ServerChallenge, conn *tls.Conn) ([protocol.SignatureSize]byte, error) {
	var sig [protocol.SignatureSize]byte

	// Получаем channel binding из TLS соединения
	channelBinding, err := protocol.GetChannelBinding(conn.ConnectionState())
	if err != nil {
		return sig, fmt.Errorf("get channel binding: %w", err)
	}

	// Собираем данные для подписи
	var clientPubKey [protocol.PublicKeySize]byte
	copy(clientPubKey[:], keys.PublicKey)

	signedData := protocol.BuildSignedData(
		challenge.Challenge,
		challenge.Timestamp,
		challenge.ServerID,
		clientPubKey,
		channelBinding,
	)

	// Подписываем
	signature := keys.Sign(signedData)
	copy(sig[:], signature)

	return sig, nil
}
