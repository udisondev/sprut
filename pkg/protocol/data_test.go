package protocol

import (
	"bytes"
	"testing"
)

func TestClientMessageEncodeDecode(t *testing.T) {
	// 64 символа hex = 32 байта pubkey
	to := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	original := &ClientMessage{
		To:      to,
		MsgID:   "test-msg-123",
		Payload: []byte("Hello, world!"),
	}

	var buf bytes.Buffer
	if err := original.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	decoded, err := DecodeClientMessage(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if decoded.To != original.To {
		t.Errorf("to: got %q, want %q", decoded.To, original.To)
	}
	if decoded.MsgID != original.MsgID {
		t.Errorf("msg_id: got %q, want %q", decoded.MsgID, original.MsgID)
	}
	if !bytes.Equal(decoded.Payload, original.Payload) {
		t.Errorf("payload: got %q, want %q", decoded.Payload, original.Payload)
	}
}

func TestClientMessageInvalidTo(t *testing.T) {
	msg := &ClientMessage{
		To:      "too_short",
		MsgID:   "test",
		Payload: []byte("test"),
	}

	var buf bytes.Buffer
	err := msg.Encode(&buf)
	if err == nil {
		t.Error("expected error for invalid to length")
	}
}

func TestClientMessageTooLong(t *testing.T) {
	to := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	msg := &ClientMessage{
		To:      to,
		MsgID:   "test",
		Payload: make([]byte, MaxMessageSize+1),
	}

	var buf bytes.Buffer
	err := msg.Encode(&buf)
	if err == nil {
		t.Error("expected error for too large message")
	}
}

func TestServerMessageEncodeDecode(t *testing.T) {
	original := &ServerMessage{
		Data: []byte("test protobuf data"),
	}

	var buf bytes.Buffer
	if err := original.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	decoded, err := DecodeServerMessage(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !bytes.Equal(decoded.Data, original.Data) {
		t.Errorf("data: got %q, want %q", decoded.Data, original.Data)
	}
}

func TestServerMessageTooLarge(t *testing.T) {
	msg := &ServerMessage{
		Data: make([]byte, MaxMessageSize+1),
	}

	var buf bytes.Buffer
	err := msg.Encode(&buf)
	if err == nil {
		t.Error("expected error for too large message")
	}
}
