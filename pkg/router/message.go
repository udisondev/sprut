package router

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/udisondev/sprut/pkg/message"
	"github.com/udisondev/sprut/pkg/protocol"
)

// minMessageSize — минимальный размер сообщения:
// To (64 hex chars) + MsgIDLen (2 bytes) = 66 bytes.
const minMessageSize = protocol.PublicKeySize*2 + 2

// errInvalidRecipient возвращается при невалидном формате адресата.
var errInvalidRecipient = errors.New("invalid recipient pubkey format")

// messagePool — пул для переиспользования protobuf Message объектов.
// Снижает нагрузку на GC при высоком throughput.
var messagePool = sync.Pool{
	New: func() any {
		return &message.Message{}
	},
}

// isValidHexPubKey проверяет, что строка является валидным hex-представлением
// публичного ключа (64 hex символа = 32 байта).
// Это предотвращает NATS subject injection через wildcard символы (*, >, .).
// Zero-allocation реализация: не использует hex.DecodeString.
func isValidHexPubKey(s string) bool {
	if len(s) != protocol.PublicKeySize*2 {
		return false
	}
	for i := range len(s) {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}


// handleMessage читает и обрабатывает одно сообщение от клиента.
func handleMessage(peer *Peer, pool *sync.Pool, maxMessageSize int) error {
	bufPtr := pool.Get().(*[]byte)
	defer pool.Put(bufPtr)
	buf := *bufPtr

	// 1. Читаем длину сообщения (4 bytes)
	if _, err := io.ReadFull(peer.conn, buf[:4]); err != nil {
		return fmt.Errorf("read length: %w", err)
	}

	totalLen := binary.BigEndian.Uint32(buf[:4])
	if totalLen > uint32(maxMessageSize) {
		slog.Warn("message: too large", "client", peer.pubKeyHex, "size", totalLen, "max", maxMessageSize)
		return fmt.Errorf("message too large: %d bytes", totalLen)
	}

	// Проверка минимальной длины для предотвращения buffer underflow
	if totalLen < minMessageSize {
		slog.Warn("message: too small", "client", peer.pubKeyHex, "size", totalLen, "min", minMessageSize)
		return fmt.Errorf("message too small: %d bytes, minimum %d", totalLen, minMessageSize)
	}

	slog.Debug("message: received", "client", peer.pubKeyHex, "size", totalLen)

	// 2. Читаем остаток сообщения
	// totalLen = To(64) + MsgIDLen(2) + MsgID + Payload
	if int(totalLen) > len(buf) {
		return fmt.Errorf("message too large for buffer: %d", totalLen)
	}

	if _, err := io.ReadFull(peer.conn, buf[:totalLen]); err != nil {
		return fmt.Errorf("read message body: %w", err)
	}

	// 3. Парсим заголовок
	// To - 64 hex символа pubkey получателя
	to := string(buf[:protocol.PublicKeySize*2])

	// Валидация hex для предотвращения NATS subject injection
	if !isValidHexPubKey(to) {
		slog.Warn("message: invalid recipient", "client", peer.pubKeyHex, "to_raw", to)
		return errInvalidRecipient
	}

	msgIDLen := binary.BigEndian.Uint16(buf[protocol.PublicKeySize*2 : protocol.PublicKeySize*2+2])
	if int(msgIDLen) > protocol.MaxMsgIDLen {
		slog.Warn("message: msgID too long", "client", peer.pubKeyHex, "len", msgIDLen, "max", protocol.MaxMsgIDLen)
		return fmt.Errorf("msgID too long: %d", msgIDLen)
	}

	// 4. Вычисляем позиции MsgID и Payload
	msgIDStart := protocol.PublicKeySize*2 + 2
	msgIDEnd := msgIDStart + int(msgIDLen)

	if msgIDEnd > int(totalLen) {
		return fmt.Errorf("invalid message structure: msgID exceeds total length")
	}

	msgID := string(buf[msgIDStart:msgIDEnd])
	payload := buf[msgIDEnd:totalLen]

	slog.Debug("message: parsed", "client", peer.pubKeyHex, "to", to, "msg_id", msgID, "payload_size", len(payload))

	// 5. Получаем Message из пула (zero-allocation hot path)
	msg := messagePool.Get().(*message.Message)
	defer func() {
		msg.Reset()
		messagePool.Put(msg)
	}()

	msg.From = peer.pubKeyHex
	msg.To = to
	msg.Id = msgID
	msg.Payload = payload
	msg.UnixDateTime = time.Now().Unix()

	// 6. Сериализуем
	data, err := proto.Marshal(msg)
	if err != nil {
		slog.Error("message: marshal failed", "client", peer.pubKeyHex, "error", err)
		return fmt.Errorf("marshal message: %w", err)
	}

	// 7. Публикуем в NATS
	if err := peer.publisher.Publish(to, data); err != nil {
		slog.Error("message: publish failed", "client", peer.pubKeyHex, "to", to, "error", err)
		return fmt.Errorf("publish to NATS: %w", err)
	}

	slog.Debug("message: published", "client", peer.pubKeyHex, "to", to, "subject", "goro.msg."+to)

	return nil
}
