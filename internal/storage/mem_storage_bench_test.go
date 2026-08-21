package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/d2cTool/rtmetrics/internal/model"
)

func BenchmarkSaveCounter(b *testing.B) {
	s := New()
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := s.SaveCounter(ctx, "PollCount", 1); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSaveGauge(b *testing.B) {
	s := New()
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := s.SaveGauge(ctx, "Alloc", 1.5); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSaveBatch(b *testing.B) {
	s := New()
	ctx := context.Background()
	alloc := 1.5
	delta := int64(1)
	batch := []model.Metrics{
		{ID: "Alloc", MType: model.Gauge, Value: &alloc},
		{ID: "PollCount", MType: model.Counter, Delta: &delta},
	}
	b.ReportAllocs()
	for b.Loop() {
		if err := s.SaveBatch(ctx, batch); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAll(b *testing.B) {
	s := New()
	ctx := context.Background()
	for i := range 64 {
		_, _ = s.SaveGauge(ctx, fmt.Sprintf("g%d", i), float64(i))
		_, _ = s.SaveCounter(ctx, fmt.Sprintf("c%d", i), int64(i))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := s.GetAllGauges(ctx); err != nil {
			b.Fatal(err)
		}
		if _, err := s.GetAllCounters(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
