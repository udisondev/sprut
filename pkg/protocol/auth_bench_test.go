package protocol

import (
	"crypto/rand"
	"testing"
)

func BenchmarkBuildSignedData(b *testing.B) {
	var challenge [ChallengeSize]byte
	var serverID [ServerIDSize]byte
	var clientPubKey [PublicKeySize]byte
	var channelBinding [ChannelBindingSize]byte

	// Заполняем тестовыми данными
	_, _ = rand.Read(challenge[:])
	_, _ = rand.Read(serverID[:])
	_, _ = rand.Read(clientPubKey[:])
	_, _ = rand.Read(channelBinding[:])

	timestamp := uint64(1706000000)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = BuildSignedData(challenge, timestamp, serverID, clientPubKey, channelBinding)
	}
}

func BenchmarkBuildSignedDataTo(b *testing.B) {
	var challenge [ChallengeSize]byte
	var serverID [ServerIDSize]byte
	var clientPubKey [PublicKeySize]byte
	var channelBinding [ChannelBindingSize]byte
	var buf [SignedDataSize]byte

	_, _ = rand.Read(challenge[:])
	_, _ = rand.Read(serverID[:])
	_, _ = rand.Read(clientPubKey[:])
	_, _ = rand.Read(channelBinding[:])

	timestamp := uint64(1706000000)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = BuildSignedDataTo(buf[:], challenge, timestamp, serverID, clientPubKey, channelBinding)
	}
}

func BenchmarkBuildSignedData_Parallel(b *testing.B) {
	var challenge [ChallengeSize]byte
	var serverID [ServerIDSize]byte
	var clientPubKey [PublicKeySize]byte
	var channelBinding [ChannelBindingSize]byte

	_, _ = rand.Read(challenge[:])
	_, _ = rand.Read(serverID[:])
	_, _ = rand.Read(clientPubKey[:])
	_, _ = rand.Read(channelBinding[:])

	timestamp := uint64(1706000000)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = BuildSignedData(challenge, timestamp, serverID, clientPubKey, channelBinding)
		}
	})
}

func BenchmarkBuildSignedDataTo_Parallel(b *testing.B) {
	var challenge [ChallengeSize]byte
	var serverID [ServerIDSize]byte
	var clientPubKey [PublicKeySize]byte
	var channelBinding [ChannelBindingSize]byte

	_, _ = rand.Read(challenge[:])
	_, _ = rand.Read(serverID[:])
	_, _ = rand.Read(clientPubKey[:])
	_, _ = rand.Read(channelBinding[:])

	timestamp := uint64(1706000000)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		var buf [SignedDataSize]byte
		for pb.Next() {
			_ = BuildSignedDataTo(buf[:], challenge, timestamp, serverID, clientPubKey, channelBinding)
		}
	})
}
