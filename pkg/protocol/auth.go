package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ClientHello — первое сообщение от клиента с публичным ключом.
type ClientHello struct {
	PubKey [PublicKeySize]byte
}

// Encode записывает ClientHello в writer.
func (m *ClientHello) Encode(w io.Writer) error {
	if _, err := w.Write([]byte{TypeClientHello}); err != nil {
		return fmt.Errorf("write type: %w", err)
	}
	if _, err := w.Write(m.PubKey[:]); err != nil {
		return fmt.Errorf("write pubkey: %w", err)
	}
	return nil
}

// DecodeClientHello читает ClientHello из reader (без байта типа).
func DecodeClientHello(r io.Reader) (*ClientHello, error) {
	var m ClientHello
	if _, err := io.ReadFull(r, m.PubKey[:]); err != nil {
		return nil, fmt.Errorf("read pubkey: %w", err)
	}
	return &m, nil
}

// ServerChallenge — challenge от сервера для аутентификации.
type ServerChallenge struct {
	Challenge [ChallengeSize]byte
	Timestamp uint64
	ServerID  [ServerIDSize]byte
}

// Encode записывает ServerChallenge в writer.
func (m *ServerChallenge) Encode(w io.Writer) error {
	if _, err := w.Write([]byte{TypeServerChallenge}); err != nil {
		return fmt.Errorf("write type: %w", err)
	}
	if _, err := w.Write(m.Challenge[:]); err != nil {
		return fmt.Errorf("write challenge: %w", err)
	}
	var ts [TimestampSize]byte
	binary.BigEndian.PutUint64(ts[:], m.Timestamp)
	if _, err := w.Write(ts[:]); err != nil {
		return fmt.Errorf("write timestamp: %w", err)
	}
	if _, err := w.Write(m.ServerID[:]); err != nil {
		return fmt.Errorf("write server_id: %w", err)
	}
	return nil
}

// DecodeServerChallenge читает ServerChallenge из reader (без байта типа).
func DecodeServerChallenge(r io.Reader) (*ServerChallenge, error) {
	var m ServerChallenge
	if _, err := io.ReadFull(r, m.Challenge[:]); err != nil {
		return nil, fmt.Errorf("read challenge: %w", err)
	}
	var ts [TimestampSize]byte
	if _, err := io.ReadFull(r, ts[:]); err != nil {
		return nil, fmt.Errorf("read timestamp: %w", err)
	}
	m.Timestamp = binary.BigEndian.Uint64(ts[:])
	if _, err := io.ReadFull(r, m.ServerID[:]); err != nil {
		return nil, fmt.Errorf("read server_id: %w", err)
	}
	return &m, nil
}

// ClientResponse — ответ клиента с подписью.
type ClientResponse struct {
	Signature [SignatureSize]byte
}

// Encode записывает ClientResponse в writer.
func (m *ClientResponse) Encode(w io.Writer) error {
	if _, err := w.Write([]byte{TypeClientResponse}); err != nil {
		return fmt.Errorf("write type: %w", err)
	}
	if _, err := w.Write(m.Signature[:]); err != nil {
		return fmt.Errorf("write signature: %w", err)
	}
	return nil
}

// DecodeClientResponse читает ClientResponse из reader (без байта типа).
func DecodeClientResponse(r io.Reader) (*ClientResponse, error) {
	var m ClientResponse
	if _, err := io.ReadFull(r, m.Signature[:]); err != nil {
		return nil, fmt.Errorf("read signature: %w", err)
	}
	return &m, nil
}

// AuthResult — результат аутентификации от сервера.
type AuthResult struct {
	Status   byte
	ErrorMsg string
}

// Encode записывает AuthResult в writer.
func (m *AuthResult) Encode(w io.Writer) error {
	if _, err := w.Write([]byte{TypeAuthResult}); err != nil {
		return fmt.Errorf("write type: %w", err)
	}
	if _, err := w.Write([]byte{m.Status}); err != nil {
		return fmt.Errorf("write status: %w", err)
	}
	if m.Status != AuthStatusOK {
		errBytes := []byte(m.ErrorMsg)
		if len(errBytes) > MaxErrorMsgLen {
			errBytes = errBytes[:MaxErrorMsgLen]
		}
		var lenBuf [2]byte
		binary.BigEndian.PutUint16(lenBuf[:], uint16(len(errBytes)))
		if _, err := w.Write(lenBuf[:]); err != nil {
			return fmt.Errorf("write error len: %w", err)
		}
		if _, err := w.Write(errBytes); err != nil {
			return fmt.Errorf("write error msg: %w", err)
		}
	}
	return nil
}

// DecodeAuthResult читает AuthResult из reader (без байта типа).
func DecodeAuthResult(r io.Reader) (*AuthResult, error) {
	var m AuthResult
	var status [1]byte
	if _, err := io.ReadFull(r, status[:]); err != nil {
		return nil, fmt.Errorf("read status: %w", err)
	}
	m.Status = status[0]

	if m.Status != AuthStatusOK {
		var lenBuf [2]byte
		if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
			return nil, fmt.Errorf("read error len: %w", err)
		}
		errLen := binary.BigEndian.Uint16(lenBuf[:])
		if errLen > MaxErrorMsgLen {
			return nil, fmt.Errorf("error message too long: %d", errLen)
		}
		errBytes := make([]byte, errLen)
		if _, err := io.ReadFull(r, errBytes); err != nil {
			return nil, fmt.Errorf("read error msg: %w", err)
		}
		m.ErrorMsg = string(errBytes)
	}
	return &m, nil
}

// ReadMessageType читает тип сообщения из reader.
func ReadMessageType(r io.Reader) (byte, error) {
	var t [1]byte
	if _, err := io.ReadFull(r, t[:]); err != nil {
		return 0, err
	}
	return t[0], nil
}

// SignedDataSize — размер буфера для BuildSignedDataTo.
const SignedDataSize = len(ProtocolVersion) + ChallengeSize + TimestampSize + ServerIDSize + PublicKeySize + ChannelBindingSize

// BuildSignedData собирает данные для подписи согласно протоколу.
// Deprecated: используйте BuildSignedDataTo для zero-allocation.
func BuildSignedData(challenge [ChallengeSize]byte, timestamp uint64, serverID [ServerIDSize]byte, clientPubKey [PublicKeySize]byte, channelBinding [ChannelBindingSize]byte) []byte {
	var buf [SignedDataSize]byte
	return BuildSignedDataTo(buf[:], challenge, timestamp, serverID, clientPubKey, channelBinding)
}

// BuildSignedDataTo записывает данные для подписи в переданный буфер.
// Буфер должен иметь размер >= SignedDataSize.
// Возвращает slice с записанными данными.
// Zero-allocation: не выделяет память на heap.
func BuildSignedDataTo(
	buf []byte,
	challenge [ChallengeSize]byte,
	timestamp uint64,
	serverID [ServerIDSize]byte,
	clientPubKey [PublicKeySize]byte,
	channelBinding [ChannelBindingSize]byte,
) []byte {
	_ = buf[SignedDataSize-1] // bounds check hint

	offset := copy(buf, ProtocolVersion)
	offset += copy(buf[offset:], challenge[:])
	binary.BigEndian.PutUint64(buf[offset:], timestamp)
	offset += TimestampSize
	offset += copy(buf[offset:], serverID[:])
	offset += copy(buf[offset:], clientPubKey[:])
	offset += copy(buf[offset:], channelBinding[:])

	return buf[:offset]
}
