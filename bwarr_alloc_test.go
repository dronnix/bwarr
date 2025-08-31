package bwarr

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAllocsSize = 11 // 4 segments to fit;

func TestBWArr_SizeOfEmpty(t *testing.T) {
	tests := []struct {
		name         string
		bwarr        *BWArr[int64]
		expectedSize int
	}{
		// Count worlds (8 bytes):
		// whiteSegments 3, // highBlackSeg 9, lowBlackSeg 9, // total 1, cmp 1 --> // 3 + 9 + 9 + 1 + 1 = 23;
		// 23 * 8 = 184 bytes;
		{
			name:         "Empty",
			bwarr:        &BWArr[int64]{},
			expectedSize: 184,
		},
		{
			name:         "New(0)",
			bwarr:        New[int64](int64Cmp, 0),
			expectedSize: 184,
		},
		{
			name:         "New(testAllocsSize)",
			bwarr:        New[int64](int64Cmp, testAllocsSize),
			expectedSize: 661,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calculatedSize := calculateBWArrSize(tt.bwarr)
			assert.Equal(t, tt.expectedSize, calculatedSize, "Expected size of empty BWArr does not match actual size")
		})
	}
}

func TestBWArr_Allocs_New(t *testing.T) {
	// Slice of segments - 1, BWArr struct - 1 --> 2;
	// 4 segments, each contains two slices: elements and deleted flags --> 8;
	// 2 black segments, each contains two slices: elements and deleted flags --> 4;
	// Total: 2 + 8 + 4 = 14;
	const expectedAllocs = 14

	allocs := testing.AllocsPerRun(100, func() {
		bwarr := New[int64](int64Cmp, testAllocsSize)
		_ = bwarr
	})

	assert.Equal(t, expectedAllocs, int(allocs), "Expected allocation count does not match actual count")
}

func TestBWArr_Allocs_NewFromSlice(t *testing.T) {
	const expectedAllocs = 12
	// Slice of segments - 1, BWArr struct - 1 --> 2;
	// 2 black segments, each contains two slices: elements and deleted flags --> 4;
	// Allocated only occupied segments: 42 = 32 + 8 + 2 --> 3 segments, 6 allocs;
	// Total: 2 + 4 + 6 = 12;
	testSlice := make([]int64, testAllocsSize)
	for i := range testSlice {
		testSlice[i] = int64(i)
	}

	allocs := testing.AllocsPerRun(100, func() {
		bwarr := NewFromSlice[int64](int64Cmp, testSlice)
		_ = bwarr
	})

	require.Equal(t, expectedAllocs, int(allocs), "Expected allocation count does not match actual count")
}

func TestBWArr_Allocs_Insert(t *testing.T) {
	const N = 100
	bwarrs := make([]*BWArr[int64], N+1)
	for i := range bwarrs {
		bwarrs[i] = New[int64](int64Cmp, testAllocsSize)
	}

	idx := 0
	allocs := testing.AllocsPerRun(N, func() {
		for i := range testAllocsSize {
			bwarrs[idx].Insert(int64(i))
		}
		idx++
	})

	assert.Equal(t, 0.0, allocs, "Expected zero memory allocations during insertions") // nolint:testifylint
}

func TestBWArr_Allocs_ReplaceOrInsert(t *testing.T) {
	const N = 100
	bwarrs := make([]*BWArr[int64], N+1)
	for i := range bwarrs {
		bwarrs[i] = New[int64](int64Cmp, testAllocsSize)
	}

	idx := 0
	allocs := testing.AllocsPerRun(N, func() {
		for i := range testAllocsSize {
			bwarrs[idx].ReplaceOrInsert(int64(i))
		}
		idx++
	})

	assert.Equal(t, 0.0, allocs, "Expected zero memory allocations during insertions") // nolint:testifylint
}

func TestBWArr_Allocs_Has(t *testing.T) {
	const N = 100
	bwarr := New[int64](int64Cmp, testAllocsSize)

	// Pre-populate with test data
	for i := range testAllocsSize {
		bwarr.Insert(int64(i))
	}

	allocs := testing.AllocsPerRun(N, func() {
		for i := range testAllocsSize * 2 { // Multiply by 2 to include some misses
			bwarr.Has(int64(i))
		}
	})

	assert.Equal(t, 0.0, allocs, "Expected zero memory allocations during Has operations") // nolint:testifylint
}

func TestBWArr_Allocs_Get(t *testing.T) {
	const N = 100
	bwarr := New[int64](int64Cmp, testAllocsSize)

	// Pre-populate with test data
	for i := range testAllocsSize {
		bwarr.Insert(int64(i))
	}

	allocs := testing.AllocsPerRun(N, func() {
		for i := range testAllocsSize * 2 { // Multiply by 2 to include some misses
			bwarr.Get(int64(i))
		}
	})

	assert.Equal(t, 0.0, allocs, "Expected zero memory allocations during Get operations") // nolint:testifylint
}

func TestBWArr_Allocs_Delete(t *testing.T) {
	const N = 100
	bwarrs := make([]*BWArr[int64], N+1)
	for i := range bwarrs {
		bwarrs[i] = New[int64](int64Cmp, testAllocsSize)
		// Pre-populate with test data
		for j := range testAllocsSize {
			bwarrs[i].Insert(int64(j))
		}
	}

	idx := 0
	allocs := testing.AllocsPerRun(N, func() {
		for i := range testAllocsSize * 2 { // Multiply by 2 to include some misses
			bwarrs[idx].Delete(int64(i))
		}
		idx++
	})

	assert.Equal(t, 0.0, allocs, "Expected zero memory allocations during Delete operations") // nolint:testifylint
}

func TestBWArr_Allocs_DeleteMin(t *testing.T) {
	const N = 100
	bwarr := New(int64Cmp, N)
	for j := range N {
		bwarr.Insert(int64(j))
	}

	idx := 0
	allocs := testing.AllocsPerRun(N, func() {
		bwarr.DeleteMin()
		idx++
	})

	assert.Equal(t, 0.0, allocs, "Expected zero memory allocations during DeleteMin operations") // nolint:testifylint
}

func TestBWArr_Allocs_DeleteMax(t *testing.T) {
	const N = 100
	bwarr := New(int64Cmp, N)
	for j := range N {
		bwarr.Insert(int64(j))
	}

	idx := 0
	allocs := testing.AllocsPerRun(N, func() {
		bwarr.DeleteMax()
		idx++
	})

	assert.Equal(t, 0.0, allocs, "Expected zero memory allocations during DeleteMax operations") // nolint:testifylint
}

func TestBWArr_Allocs_Min(t *testing.T) {
	const N = 100
	bwarr := New[int64](int64Cmp, N)

	// Pre-populate with test data
	for i := range N {
		bwarr.Insert(int64(i))
	}

	allocs := testing.AllocsPerRun(N, func() {
		bwarr.Min()
	})

	assert.Equal(t, 0.0, allocs, "Expected zero memory allocations during Min operations") // nolint:testifylint
}

func TestBWArr_Allocs_Max(t *testing.T) {
	const N = 100
	bwarr := New[int64](int64Cmp, N)

	// Pre-populate with test data
	for i := range N {
		bwarr.Insert(int64(i))
	}

	allocs := testing.AllocsPerRun(N, func() {
		bwarr.Max()
	})

	assert.Equal(t, 0.0, allocs, "Expected zero memory allocations during Max operations") // nolint:testifylint
}

func TestBWArr_Allocs_Clear(t *testing.T) {
	bwarr := New[int64](int64Cmp, testAllocsSize)

	// Pre-populate with test data
	for i := range testAllocsSize {
		bwarr.Insert(int64(i))
	}

	allocs := testing.AllocsPerRun(1, func() {
		bwarr.Clear(false)
		bwarr.Clear(true)
	})

	assert.Equal(t, 0.0, allocs, "Expected zero memory allocations during Clear operations") // nolint:testifylint
}

func TestBWArr_Allocs_ShouldBeEmptyAfterClear(t *testing.T) {
	bwarr := New[int64](int64Cmp, 0)
	szBefore := calculateBWArrSize(bwarr)

	const N = 100
	for i := range N {
		bwarr.Insert(int64(i))
	}

	bwarr.Clear(true)

	szAfter := calculateBWArrSize(bwarr)
	assert.Equal(t, szBefore, szAfter, "BWArr size should match initial size after Clear with dropSegments")
}

// calculateBWArrSize calculates the total size of a BWArr struct including all nested fields
func calculateBWArrSize[T any](bwarr *BWArr[T]) int {
	size := int(unsafe.Sizeof(*bwarr))
	// Add size of black segs:
	size += calculateSegmentSize(&bwarr.highBlackSeg)
	size += calculateSegmentSize(&bwarr.lowBlackSeg)
	// Add size of each segment in whiteSegments
	for _, seg := range bwarr.whiteSegments {
		size += int(unsafe.Sizeof(seg))
		size += calculateSegmentSize(&seg)
	}
	return size
}

// calculateSegmentSize calculates the size of a segment including its slices
func calculateSegmentSize[T any](seg *segment[T]) (size int) {
	// Add size of elements slice
	if len(seg.elements) > 0 {
		size += len(seg.elements) * int(unsafe.Sizeof(seg.elements[0]))
	}
	// Add size of deleted slice
	if len(seg.deleted) > 0 {
		size += len(seg.deleted) * int(unsafe.Sizeof(seg.deleted[0]))
	}
	return size
}
