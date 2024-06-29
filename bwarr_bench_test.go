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
	benchmarkAppend(b, elemsOnStart, elemsOnStart)
}

func BenchmarkQA_ReplaceOrInsertNotFound(b *testing.B) {
	benchmarkReplace(b, elemsOnStart, elemsOnStart)
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
		preparedData[i] = rand.Int63()
	}
	b.SetBytes(8) //nolint:exhaustruct
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.Has(preparedData[i])
	}
}

func BenchmarkQA_Min(b *testing.B) {
	bwa := New(int64Cmp, elemsOnStart)

	for range elemsOnStart {
		bwa.Insert(rand.Int63())
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.Min()
	}
}

func BenchmarkQA_Max(b *testing.B) {
	bwa := New(int64Cmp, elemsOnStart)

	for range elemsOnStart {
		bwa.Insert(rand.Int63())
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.Max()
	}
}

func BenchmarkQA_Delete(b *testing.B) {
	bwa := New(int64Cmp, elemsOnStart+b.N)
	for range elemsOnStart {
		bwa.Insert(rand.Int63())
	}

	toDel := make([]int64, b.N)
	for i := range b.N {
		toDel[i] = rand.Int63()
		bwa.Insert(toDel[i])
	}
	rand.Shuffle(len(toDel), func(i, j int) { toDel[i], toDel[j] = toDel[j], toDel[i] })

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, found := bwa.Delete(toDel[i]); !found {
			b.Fail()
		}
	}
}

func BenchmarkQA_DeleteMin(b *testing.B) {
	const elemsOnStart = 4 * 1024 * 1024
	elems := elemsOnStart + b.N
	bwa := New(int64Cmp, elems)

	for range elems {
		bwa.Insert(rand.Int63())
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.DeleteMin()
	}
}

func BenchmarkQA_DeleteMax(b *testing.B) {
	const elemsOnStart = 4 * 1024 * 1024
	elems := elemsOnStart + b.N
	bwa := New(int64Cmp, elems)

	for range elems {
		bwa.Insert(rand.Int63())
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.DeleteMax()
	}
}

func Benchmark_AscendRandom(b *testing.B) {
	const elems = 128*1024 - 1
	bwa := New(int64Cmp, elems)
	for range elems {
		bwa.Insert(rand.Int63())
	}

	s := int64(0)
	iter := func(x int64) bool {
		s += x
		return true
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.Ascend(iter)
	}
}

func Benchmark_AscendInc(b *testing.B) {
	const elems = 128*1024 - 1
	bwa := New(int64Cmp, elems)
	for i := range elems {
		bwa.Insert(int64(i))
	}

	s := int64(0)
	iter := func(x int64) bool {
		s += x
		return true
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.Ascend(iter)
	}
}

func Benchmark_AscendDec(b *testing.B) {
	const elems = 128*1024 - 1
	bwa := New(int64Cmp, elems)
	for i := range elems {
		bwa.Insert(int64(elems - i))
	}

	s := int64(0)
	iter := func(x int64) bool {
		s += x
		return true
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.Ascend(iter)
	}
}

func benchmarkAppend(b *testing.B, elemsOnStart, capacity int) {
	bwa := New(int64Cmp, capacity)

	for i := 0; i < elemsOnStart; i++ {
		bwa.Insert(rand.Int63())
	}
	preparedData := make([]int64, b.N)
	for i := 0; i < b.N; i++ {
		preparedData[i] = rand.Int63()
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.Insert(preparedData[i])
	}
}

func benchmarkReplace(b *testing.B, elemsOnStart, capacity int) {
	bwa := New(int64Cmp, capacity)

	for i := 0; i < elemsOnStart; i++ {
		bwa.Insert(rand.Int63())
	}
	preparedData := make([]int64, b.N)
	for i := 0; i < b.N; i++ {
		preparedData[i] = rand.Int63()
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.ReplaceOrInsert(preparedData[i])
	}
}
