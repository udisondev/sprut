package router

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/udisondev/sprut/pkg/protocol"
)

// Смещения в буфере аутентификации.
// Области буфера не перекрываются:
//
//	[0:32]      - pubKey (сохраняется на всё время auth)
//	[32:64]     - challenge (32 bytes)
//	[64:72]     - timestamp (8 bytes)
//	[72:104]    - serverID (32 bytes, записан при инициализации)
//	[104:168]   - signature (64 bytes)
//	[168:168+SignedDataSize] - signedData для верификации (128 bytes)
//	[296:...]   - рабочая область для отправки/чтения
const (
	offPubKey     = 0
	offChallenge  = 32
	offTimestamp  = 64
	offServerID   = 72
	offSignature  = 104
	offSignedData = 168
	offWork       = offSignedData + protocol.SignedDataSize // 168 + 128 = 296
	AuthBufSize   = offWork + 128                           // с запасом для рабочих данных
)

// authenticate выполняет аутентификацию клиента.
// При успехе pubKey остаётся в buf[offPubKey:offPubKey+32].
// ServerID уже записан в buf[offServerID:offServerID+32] при инициализации семафора.
func authenticate(conn net.Conn, timeout, challengeTTL time.Duration, buf []byte) error {
	remote := conn.RemoteAddr().String()

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}
	defer func() {
		if err := conn.SetDeadline(time.Time{}); err != nil {
			slog.Error("auth: reset deadline failed", "error", err, "remote", remote)
		}
	}()

	// 1. Читаем TypeClientHello (1 byte)
	if _, err := io.ReadFull(conn, buf[offWork:offWork+1]); err != nil {
		return fmt.Errorf("read hello type: %w", err)
	}
	if buf[offWork] != protocol.TypeClientHello {
		slog.Warn("auth: unexpected message type", "remote", remote, "expected", protocol.TypeClientHello, "got", buf[offWork])
		return fmt.Errorf("unexpected message type: %d", buf[offWork])
	}

	// 2. Читаем PubKey в отдельную область (останется после return)
	if _, err := io.ReadFull(conn, buf[offPubKey:offPubKey+protocol.PublicKeySize]); err != nil {
		return fmt.Errorf("read pubkey: %w", err)
	}

	pubKeyPrefix := hex.EncodeToString(buf[offPubKey : offPubKey+8])
	slog.Debug("auth: received client hello", "remote", remote, "pubkey_prefix", pubKeyPrefix)

	// 3. Генерируем challenge прямо в буфер
	if _, err := rand.Read(buf[offChallenge : offChallenge+protocol.ChallengeSize]); err != nil {
		return fmt.Errorf("generate challenge: %w", err)
	}
	slog.Debug("auth: challenge generated", "remote", remote)

	// 4. Записываем timestamp в буфер
	timestamp := uint64(time.Now().Unix())
	binary.BigEndian.PutUint64(buf[offTimestamp:offTimestamp+protocol.TimestampSize], timestamp)

	// 5. ServerID уже в буфере (записан при инициализации семафора)

	// 6. Отправляем ServerChallenge: Type(1) + Challenge(32) + Timestamp(8) + ServerID(32) = 73 bytes
	challengeMsg := buf[offWork : offWork+1+protocol.ChallengeSize+protocol.TimestampSize+protocol.ServerIDSize]
	challengeMsg[0] = protocol.TypeServerChallenge
	copy(challengeMsg[1:], buf[offChallenge:offChallenge+protocol.ChallengeSize])
	copy(challengeMsg[1+protocol.ChallengeSize:], buf[offTimestamp:offTimestamp+protocol.TimestampSize])
	copy(challengeMsg[1+protocol.ChallengeSize+protocol.TimestampSize:], buf[offServerID:offServerID+protocol.ServerIDSize])

	if _, err := conn.Write(challengeMsg); err != nil {
		return fmt.Errorf("send challenge: %w", err)
	}
	slog.Debug("auth: challenge sent", "remote", remote)

	// 7. Читаем TypeClientResponse (1 byte)
	if _, err := io.ReadFull(conn, buf[offWork:offWork+1]); err != nil {
		return fmt.Errorf("read response type: %w", err)
	}
	if buf[offWork] != protocol.TypeClientResponse {
		slog.Warn("auth: unexpected message type", "remote", remote, "expected", protocol.TypeClientResponse, "got", buf[offWork])
		return fmt.Errorf("unexpected message type: %d", buf[offWork])
	}
	slog.Debug("auth: received client response", "remote", remote)

	// 8. Читаем Signature
	if _, err := io.ReadFull(conn, buf[offSignature:offSignature+protocol.SignatureSize]); err != nil {
		return fmt.Errorf("read signature: %w", err)
	}

	// 9. Получаем channel binding из TLS соединения
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return fmt.Errorf("not a TLS connection")
	}
	channelBinding, err := protocol.GetChannelBinding(tlsConn.ConnectionState())
	if err != nil {
		slog.Error("auth: channel binding failed", "remote", remote, "error", err)
		return fmt.Errorf("get channel binding: %w", err)
	}

	// 10. Собираем данные для верификации подписи (zero-allocation)
	var challenge [protocol.ChallengeSize]byte
	var serverID [protocol.ServerIDSize]byte
	var pubKey [protocol.PublicKeySize]byte

	copy(challenge[:], buf[offChallenge:offChallenge+protocol.ChallengeSize])
	copy(serverID[:], buf[offServerID:offServerID+protocol.ServerIDSize])
	copy(pubKey[:], buf[offPubKey:offPubKey+protocol.PublicKeySize])

	signedData := protocol.BuildSignedDataTo(buf[offSignedData:offSignedData+protocol.SignedDataSize], challenge, timestamp, serverID, pubKey, channelBinding)

	slog.Debug("auth: verifying signature", "remote", remote)

	// 11. Верифицируем подпись
	if !ed25519.Verify(buf[offPubKey:offPubKey+protocol.PublicKeySize], signedData, buf[offSignature:offSignature+protocol.SignatureSize]) {
		slog.Warn("auth: invalid signature", "remote", remote)
		return protocol.ErrInvalidSignature
	}
	slog.Debug("auth: signature valid", "remote", remote)

	// 12. Проверяем timestamp для защиты от replay attack
	now := uint64(time.Now().Unix())
	if timestamp > now+60 {
		slog.Warn("auth: timestamp in future", "remote", remote, "diff_seconds", timestamp-now)
		return fmt.Errorf("timestamp in future")
	}
	if now-timestamp > uint64(challengeTTL.Seconds()) {
		slog.Warn("auth: challenge expired", "remote", remote, "age_seconds", now-timestamp)
		return protocol.ErrChallengeExpired
	}
	slog.Debug("auth: timestamp valid", "remote", remote, "age_seconds", now-timestamp)

	// 13. Отправляем успешный результат (синхронизация с клиентом)
	// PubKey остаётся в buf[offPubKey:] - вызывающий код возьмёт его оттуда
	buf[offWork] = protocol.TypeAuthResult
	buf[offWork+1] = protocol.AuthStatusOK
	if _, err := conn.Write(buf[offWork : offWork+2]); err != nil {
		return fmt.Errorf("send auth result: %w", err)
	}
	slog.Debug("auth: auth result sent", "remote", remote)

	return nil
}
