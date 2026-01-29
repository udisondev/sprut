package protocol

import (
	"bytes"
	"testing"
)

func TestClientHelloEncodeDecode(t *testing.T) {
	original := &ClientHello{}
	for i := range PublicKeySize {
		original.PubKey[i] = byte(i)
	}

	var buf bytes.Buffer
	if err := original.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	data := buf.Bytes()
	if data[0] != TypeClientHello {
		t.Errorf("type: got %d, want %d", data[0], TypeClientHello)
	}

	reader := bytes.NewReader(data[1:])
	decoded, err := DecodeClientHello(reader)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if decoded.PubKey != original.PubKey {
		t.Errorf("pubkey mismatch")
	}
}

func TestServerChallengeEncodeDecode(t *testing.T) {
	original := &ServerChallenge{
		Timestamp: 1706000000,
	}
	for i := range ChallengeSize {
		original.Challenge[i] = byte(i)
	}
	for i := range ServerIDSize {
		original.ServerID[i] = byte(i + 100)
	}

	var buf bytes.Buffer
	if err := original.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	data := buf.Bytes()
	if data[0] != TypeServerChallenge {
		t.Errorf("type: got %d, want %d", data[0], TypeServerChallenge)
	}

	reader := bytes.NewReader(data[1:])
	decoded, err := DecodeServerChallenge(reader)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if decoded.Challenge != original.Challenge {
		t.Errorf("challenge mismatch")
	}
	if decoded.Timestamp != original.Timestamp {
		t.Errorf("timestamp: got %d, want %d", decoded.Timestamp, original.Timestamp)
	}
	if decoded.ServerID != original.ServerID {
		t.Errorf("server_id mismatch")
	}
}

func TestClientResponseEncodeDecode(t *testing.T) {
	original := &ClientResponse{}
	for i := range SignatureSize {
		original.Signature[i] = byte(i)
	}

	var buf bytes.Buffer
	if err := original.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	data := buf.Bytes()
	if data[0] != TypeClientResponse {
		t.Errorf("type: got %d, want %d", data[0], TypeClientResponse)
	}

	reader := bytes.NewReader(data[1:])
	decoded, err := DecodeClientResponse(reader)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if decoded.Signature != original.Signature {
		t.Errorf("signature mismatch")
	}
}

func TestAuthResultEncodeDecode(t *testing.T) {
	tests := []struct {
		name     string
		status   byte
		errorMsg string
	}{
		{"ok", AuthStatusOK, ""},
		{"invalid_sig", AuthStatusInvalidSig, "invalid signature"},
		{"replay", AuthStatusReplay, "replay detected"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := &AuthResult{
				Status:   tt.status,
				ErrorMsg: tt.errorMsg,
			}

			var buf bytes.Buffer
			if err := original.Encode(&buf); err != nil {
				t.Fatalf("encode: %v", err)
			}

			data := buf.Bytes()
			if data[0] != TypeAuthResult {
				t.Errorf("type: got %d, want %d", data[0], TypeAuthResult)
			}

			reader := bytes.NewReader(data[1:])
			decoded, err := DecodeAuthResult(reader)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}

			if decoded.Status != original.Status {
				t.Errorf("status: got %d, want %d", decoded.Status, original.Status)
			}
			if decoded.ErrorMsg != original.ErrorMsg {
				t.Errorf("error_msg: got %q, want %q", decoded.ErrorMsg, original.ErrorMsg)
			}
		})
	}
}

func TestBuildSignedData(t *testing.T) {
	var challenge [ChallengeSize]byte
	var serverID [ServerIDSize]byte
	var clientPubKey [PublicKeySize]byte
	var channelBinding [ChannelBinding]byte

	for i := range ChallengeSize {
		challenge[i] = byte(i)
	}
	for i := range ServerIDSize {
		serverID[i] = byte(i + 50)
	}
	for i := range PublicKeySize {
		clientPubKey[i] = byte(i + 100)
	}
	for i := range ChannelBinding {
		channelBinding[i] = byte(i + 150)
	}

	timestamp := uint64(1706000000)

	data := BuildSignedData(challenge, timestamp, serverID, clientPubKey, channelBinding)

	// Проверяем, что данные начинаются с версии протокола
	if !bytes.HasPrefix(data, []byte(ProtocolVersion)) {
		t.Errorf("should start with protocol version")
	}

	// Проверяем общую длину
	expectedLen := len(ProtocolVersion) + ChallengeSize + TimestampSize + ServerIDSize + PublicKeySize + ChannelBinding
	if len(data) != expectedLen {
		t.Errorf("length: got %d, want %d", len(data), expectedLen)
	}
}
