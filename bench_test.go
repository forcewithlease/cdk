package cdk_test

import (
	"testing"

	"github.com/forcewithlease/cdk"
)

var (
	sinkCode string
	sinkN    uint64
)

func BenchmarkFixedEncode(b *testing.B) {
	c, err := cdk.NewFixed(10, cdk.Config{Key: []byte("bench-key")})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkCode, _ = c.Encode(uint64(i))
	}
}

func BenchmarkFixedDecode(b *testing.B) {
	c, err := cdk.NewFixed(10, cdk.Config{Key: []byte("bench-key")})
	if err != nil {
		b.Fatal(err)
	}
	codes := make([]string, 1024)
	for i := range codes {
		codes[i], _ = c.Encode(uint64(i))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkN, _ = c.Decode(codes[i&1023])
	}
}

func BenchmarkVariableEncode(b *testing.B) {
	c, err := cdk.NewVariable(10, cdk.Config{Key: []byte("bench-key")})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkCode, _ = c.Encode(uint64(i))
	}
}

func BenchmarkBijectiveEncode(b *testing.B) {
	c, err := cdk.NewBijective(cdk.Config{Key: []byte("bench-key")})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkCode, _ = c.Encode(uint64(i))
	}
}

func BenchmarkBandedEncode(b *testing.B) {
	c, err := cdk.NewBanded(6, 10, cdk.Config{Key: []byte("bench-key")})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkCode, _ = c.Encode(uint64(i))
	}
}

func BenchmarkBijectiveDecode(b *testing.B) {
	c, err := cdk.NewBijective(cdk.Config{Key: []byte("bench-key")})
	if err != nil {
		b.Fatal(err)
	}
	codes := make([]string, 1024)
	for i := range codes {
		codes[i], _ = c.Encode(uint64(i))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkN, _ = c.Decode(codes[i&1023])
	}
}
