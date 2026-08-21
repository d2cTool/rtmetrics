package hash

import (
	"testing"
)

func BenchmarkSign(b *testing.B) {
	body := []byte(`[{"id":"Alloc","type":"gauge","value":1.5}]`)
	const key = "secret"
	b.ReportAllocs()
	for b.Loop() {
		_ = Sign(body, key)
	}
}

func BenchmarkValid(b *testing.B) {
	body := []byte(`[{"id":"Alloc","type":"gauge","value":1.5}]`)
	const key = "secret"
	signature := Sign(body, key)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if !Valid(body, key, signature) {
			b.Fatal("signature must be valid")
		}
	}
}
