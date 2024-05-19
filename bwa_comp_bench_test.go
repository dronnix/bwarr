package bwarr // TODO: Move it to the separate repo or refactor to use to track regressions.
import (
	"math/rand"
	"strings"
	"testing"
	"unsafe"

	"github.com/google/btree"
)

func BenchmarkBTreeAdd4M(b *testing.B) {
	bt := createGenericBTree()
	const elemsOnStart = 4 * 1024 * 1024
	for i := 0; i < elemsOnStart; i++ {
		bt.ReplaceOrInsert(rand.Int63()) //nolint:gosec
	}
	preparedData := make([]int64, b.N)
	for i := 0; i < b.N; i++ {
		preparedData[i] = rand.Int63() //nolint:gosec
	}

	b.SetBytes(8)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bt.ReplaceOrInsert(preparedData[i])
	}
}

func BenchmarkBTreeAdd4MHugeStruct(b *testing.B) {
	bt := createGenericBTreeHugeStruct()
	const elemsOnStart = 64 * 1024
	for i := 0; i < elemsOnStart; i++ {
		bt.ReplaceOrInsert(makeHugeStruct()) //nolint:gosec
	}
	preparedData := make([]hugeStruct, b.N)
	for i := 0; i < b.N; i++ {
		preparedData[i] = makeHugeStruct()
	}

	b.SetBytes(int64(unsafe.Sizeof(hugeStruct{}))) //nolint:exhaustruct
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bt.ReplaceOrInsert(preparedData[i])
	}
}

func BenchmarkReplace4MZeroCapacityHugeStruct(b *testing.B) {
	const elemsOnStart = 64 * 1024
	bwa := New(hugeStructCmp, 0)

	for i := 0; i < elemsOnStart; i++ {
		bwa.Insert(makeHugeStruct())
	}
	preparedData := make([]hugeStruct, b.N)
	for i := 0; i < b.N; i++ {
		preparedData[i] = makeHugeStruct()
	}

	b.SetBytes(int64(unsafe.Sizeof(hugeStruct{}))) //nolint:exhaustruct
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.ReplaceOrInsert(preparedData[i])
	}
}

func BenchmarkAppend4MZeroCapacityHugeStruct(b *testing.B) {
	const elemsOnStart = 64 * 1024
	bwa := New(hugeStructCmp, 0)

	for i := 0; i < elemsOnStart; i++ {
		bwa.Insert(makeHugeStruct())
	}
	preparedData := make([]hugeStruct, b.N)
	for i := 0; i < b.N; i++ {
		preparedData[i] = makeHugeStruct()
	}

	b.SetBytes(int64(unsafe.Sizeof(hugeStruct{}))) //nolint:exhaustruct
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bwa.Insert(preparedData[i])
	}
}

func BenchmarkBWArr_Min4M_Fragmented(b *testing.B) {
	const elemsOnStart = 4*1024*1024 - 1
	bwa := New(int64Cmp, elemsOnStart)

	elems := make([]int64, elemsOnStart)
	for i := 0; i < elemsOnStart; i++ {
		x := rand.Int63()
		bwa.Insert(x)
		elems[i] = x
	}

	for range elemsOnStart / 3 {
		bwa.DeleteMin()
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		elem, found := bwa.Min()
		if !found {
			b.Fatalf("Element %d not found", elem)
		}
	}
}

func BenchmarkBTree_Min(b *testing.B) {
	bt := createGenericBTree()
	const elems = 4 * 1024 * 1024
	for i := 0; i < elems; i++ {
		bt.ReplaceOrInsert(rand.Int63()) //nolint:gosec
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		elem, found := bt.Min()
		if !found {
			b.Fatalf("Element %d not found", elem)
		}
	}
}

func BenchmarkBTree_Min_Fragmented(b *testing.B) {
	bt := createGenericBTree()
	const elemsOnStart = 4 * 1024 * 1024
	elems := make([]int64, elemsOnStart)
	for i := 0; i < elemsOnStart; i++ {
		x := rand.Int63()     //nolint:gosec
		bt.ReplaceOrInsert(x) //nolint:gosec
		elems[i] = x
	}

	for i := range elemsOnStart / 3 {
		bt.Delete(elems[i])
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		elem, found := bt.Min()
		if !found {
			b.Fatalf("Element %d not found", elem)
		}
	}
}

func BenchmarkBTree_DeleteMin(b *testing.B) {
	bt := createGenericBTree()
	const elemsOnStart = 4 * 1024 * 1024
	elems := elemsOnStart + b.N
	for i := 0; i < elems; i++ {
		bt.ReplaceOrInsert(rand.Int63()) //nolint:gosec
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bt.DeleteMin()
	}
}

func createGenericBTree() *btree.BTreeG[int64] {
	const degree = 32
	return btree.NewG(degree, func(a, b int64) bool {
		return a < b
	})
}

func createGenericBTreeHugeStruct() *btree.BTreeG[hugeStruct] {
	const degree = 4
	return btree.NewG(degree, func(a, b hugeStruct) bool {
		return hugeStructCmp(a, b) < 0
	})
}

type hugeStruct struct {
	A1 [17]int64
	S1 string
	A2 [41]int64
	I  int
}

func makeHugeStruct() hugeStruct {
	hs := hugeStruct{I: 42, S1: "Some string"} //nolint:exhaustruct
	hs.A2[40] = rand.Int63()
	return hs
}

func hugeStructCmp(a, b hugeStruct) int { // nolint:gocritic
	iCmp := a.I - b.I
	if iCmp != 0 {
		return iCmp
	}

	sCmp := strings.Compare(a.S1, b.S1)
	if sCmp != 0 {
		return sCmp
	}

	for i := 0; i < len(a.A1); i++ {
		arrCmp := a.A1[i] - b.A1[i]
		if arrCmp != 0 {
			return int(arrCmp)
		}
	}

	for i := 0; i < len(a.A2); i++ {
		arrCmp := a.A2[i] - b.A2[i]
		if arrCmp != 0 {
			return int(arrCmp)
		}
	}

	return 0
}
