package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ClientMessage — сообщение от клиента к серверу.
type ClientMessage struct {
	To      string // hex-encoded публичный ключ получателя (64 символа)
	MsgID   string
	Payload []byte
}

// Encode записывает ClientMessage в writer.
func (m *ClientMessage) Encode(w io.Writer) error {
	toBytes := []byte(m.To)
	msgIDBytes := []byte(m.MsgID)

	if len(toBytes) != PublicKeySize*2 {
		return fmt.Errorf("invalid to length: expected %d, got %d", PublicKeySize*2, len(toBytes))
	}
	if len(msgIDBytes) > MaxMsgIDLen {
		return fmt.Errorf("msg_id too long: %d > %d", len(msgIDBytes), MaxMsgIDLen)
	}

	totalLen := PublicKeySize*2 + 2 + len(msgIDBytes) + len(m.Payload)
	if totalLen > MaxMessageSize {
		return fmt.Errorf("message too large: %d > %d", totalLen, MaxMessageSize)
	}

	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(totalLen))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return fmt.Errorf("write total len: %w", err)
	}

	if _, err := w.Write(toBytes); err != nil {
		return fmt.Errorf("write to: %w", err)
	}

	var msgIDLenBuf [2]byte
	binary.BigEndian.PutUint16(msgIDLenBuf[:], uint16(len(msgIDBytes)))
	if _, err := w.Write(msgIDLenBuf[:]); err != nil {
		return fmt.Errorf("write msg_id len: %w", err)
	}

	if _, err := w.Write(msgIDBytes); err != nil {
		return fmt.Errorf("write msg_id: %w", err)
	}

	if _, err := w.Write(m.Payload); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	return nil
}

// DecodeClientMessage читает ClientMessage из reader.
func DecodeClientMessage(r io.Reader) (*ClientMessage, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, fmt.Errorf("read total len: %w", err)
	}
	totalLen := binary.BigEndian.Uint32(lenBuf[:])
	if totalLen > MaxMessageSize {
		return nil, fmt.Errorf("message too large: %d > %d", totalLen, MaxMessageSize)
	}

	data := make([]byte, totalLen)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("read message data: %w", err)
	}

	if len(data) < PublicKeySize*2+2 {
		return nil, fmt.Errorf("message too short")
	}

	m := &ClientMessage{
		To: string(data[:PublicKeySize*2]),
	}
	data = data[PublicKeySize*2:]

	msgIDLen := binary.BigEndian.Uint16(data[:2])
	data = data[2:]

	if int(msgIDLen) > len(data) {
		return nil, fmt.Errorf("invalid msg_id length")
	}

	m.MsgID = string(data[:msgIDLen])
	m.Payload = data[msgIDLen:]

	return m, nil
}

// ServerMessage — сообщение от сервера к клиенту (protobuf-wrapped).
type ServerMessage struct {
	Data []byte // marshaled protobuf Message
}

// Encode записывает ServerMessage в writer.
func (m *ServerMessage) Encode(w io.Writer) error {
	if len(m.Data) > MaxMessageSize {
		return fmt.Errorf("message too large: %d > %d", len(m.Data), MaxMessageSize)
	}

	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(m.Data)))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return fmt.Errorf("write total len: %w", err)
	}

	if _, err := w.Write(m.Data); err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	return nil
}

// DecodeServerMessage читает ServerMessage из reader.
func DecodeServerMessage(r io.Reader) (*ServerMessage, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, fmt.Errorf("read total len: %w", err)
	}
	totalLen := binary.BigEndian.Uint32(lenBuf[:])
	if totalLen > MaxMessageSize {
		return nil, fmt.Errorf("message too large: %d > %d", totalLen, MaxMessageSize)
	}

	data := make([]byte, totalLen)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("read data: %w", err)
	}

	return &ServerMessage{Data: data}, nil
}
