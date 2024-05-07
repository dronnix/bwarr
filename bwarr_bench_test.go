package bwarr

// Benchmarks with prefix QA (quality assurance) is for tracking performance regressions.

import (
	"math/rand"
	"testing"
)

// Less by 1 than segment size to provoke segment allocation
// and to use as much segments as possible for non-modifying operations.
const elemsOnStart = 4*1024*1024 - 1

func BenchmarkQA_Insert(b *testing.B) {
	bwa := New(int64Cmp, elemsOnStart)

	for i := 0; i < elemsOnStart; i++ {
		bwa.Insert(rand.Int63())
	}
	preparedData := make([]int64, b.N)
	for i := 0; i < b.N; i++ {
		preparedData[i] = rand.Int63() //nolint:gosec
	}
	b.SetBytes(8) //nolint:exhaustruct
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.Insert(preparedData[i])
	}
}

func BenchmarkQA_ReplaceOrInsertNotFound(b *testing.B) {
	bwa := New(int64Cmp, elemsOnStart)

	for i := 0; i < elemsOnStart; i++ {
		bwa.Insert(rand.Int63())
	}
	preparedData := make([]int64, b.N)
	for i := 0; i < b.N; i++ {
		preparedData[i] = rand.Int63() //nolint:gosec
	}
	b.SetBytes(8) //nolint:exhaustruct
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.ReplaceOrInsert(preparedData[i])
	}
}

func BenchmarkQA_ReplaceOrInsertFound(b *testing.B) {
	bwa := New(int64Cmp, elemsOnStart)
	preparedData := make([]int64, elemsOnStart)

	for i := 0; i < elemsOnStart; i++ {
		preparedData[i] = rand.Int63()
		bwa.Insert(preparedData[i])
	}
	for i := 0; i < b.N; i++ {
	}
	b.SetBytes(8) //nolint:exhaustruct
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.ReplaceOrInsert(preparedData[i%elemsOnStart])
	}
}

func BenchmarkQA_HasFound(b *testing.B) {
	bwa := New(int64Cmp, elemsOnStart)
	preparedData := make([]int64, elemsOnStart)

	for i := 0; i < elemsOnStart; i++ {
		preparedData[i] = rand.Int63()
		bwa.Insert(preparedData[i])
	}
	for i := 0; i < b.N; i++ {
	}
	b.SetBytes(8) //nolint:exhaustruct
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.Has(preparedData[i%elemsOnStart])
	}
}

func BenchmarkQA_HasNotFoundWorst(b *testing.B) {
	bwa := New(int64Cmp, elemsOnStart)

	for i := 0; i < elemsOnStart; i++ {
		bwa.Insert(rand.Int63())
	}
	preparedData := make([]int64, b.N)
	for i := 0; i < b.N; i++ {
		preparedData[i] = rand.Int63() //nolint:gosec
	}
	b.SetBytes(8) //nolint:exhaustruct
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.Has(preparedData[i])
	}
}
