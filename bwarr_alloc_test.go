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
		// Count words (8 bytes):
		// whiteSegments 3, // total 1, cmp 1 --> // 3 + 1 + 1 = 5;
		// 5 * 8 = 40 bytes;
		{
			name:         "Empty",
			bwarr:        &BWArr[int64]{},
			expectedSize: 40,
		},
		{
			name:         "New(0)",
			bwarr:        New[int64](int64Cmp, 0),
			expectedSize: 40,
		},
		{
			name:         "New(testAllocsSize)",
			bwarr:        New[int64](int64Cmp, testAllocsSize),
			expectedSize: 463,
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
	// Total: 2 + 8 = 10;
	const expectedAllocs = 10

	allocs := testing.AllocsPerRun(100, func() {
		bwarr := New[int64](int64Cmp, testAllocsSize)
		_ = bwarr
	})

	assert.Equal(t, expectedAllocs, int(allocs), "Expected allocation count does not match actual count")
}

func TestBWArr_Allocs_NewFromSlice(t *testing.T) {
	const expectedAllocs = 8
	// Slice of segments - 1, BWArr struct - 1 --> 2;
	// Allocated only occupied segments: 42 = 32 + 8 + 2 --> 3 segments, 6 allocs;
	// Total: 2 + 6 = 8;
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

func TestBWArr_Allocs_Clone(t *testing.T) {
	bwarr := New[int64](int64Cmp, testAllocsSize)

	for i := range testAllocsSize {
		bwarr.Insert(int64(i))
	}

	const N = 100
	allocs := testing.AllocsPerRun(N, func() {
		c := bwarr.Clone()
		c.Len() // Use the clone to prevent compiler optimizations
	})

	assert.Equal(t, 8.0, allocs, "Expected 8 memory allocations during Clone") // nolint:testifylint
}

func TestBWArr_Allocs_Len(t *testing.T) {
	const N = 100
	bwarr := New[int64](int64Cmp, N)

	for i := range N {
		bwarr.Insert(int64(i))
	}

	allocs := testing.AllocsPerRun(N, func() {
		bwarr.Len()
	})

	assert.Equal(t, 0.0, allocs, "Expected zero memory allocations during Len operations") // nolint:testifylint
}

func TestBWArr_Allocs_Ascend(t *testing.T) {
	bwarr := New[int64](int64Cmp, testAllocsSize)

	for i := range testAllocsSize {
		bwarr.Insert(int64(i))
	}

	const N = 100
	allocs := testing.AllocsPerRun(N, func() {
		s := int64(0)
		bwarr.Ascend(func(item int64) bool {
			s += item
			return true
		})
	})

	assert.Equal(t, 2.0, allocs, "Expected 2 memory allocations during Ascend") // nolint:testifylint
}

func TestBWArr_Allocs_AscendGreaterOrEqual(t *testing.T) {
	bwarr := New[int64](int64Cmp, testAllocsSize)

	for i := range testAllocsSize {
		bwarr.Insert(int64(i))
	}

	const N = 100
	allocs := testing.AllocsPerRun(N, func() {
		s := int64(0)
		bwarr.AscendGreaterOrEqual(int64(5), func(item int64) bool {
			s += item
			return true
		})
	})

	assert.Equal(t, 2.0, allocs, "Expected 2 memory allocations during AscendGreaterOrEqual") // nolint:testifylint
}

func TestBWArr_Allocs_AscendLessThan(t *testing.T) {
	bwarr := New[int64](int64Cmp, testAllocsSize)

	for i := range testAllocsSize {
		bwarr.Insert(int64(i))
	}

	const N = 100
	allocs := testing.AllocsPerRun(N, func() {
		s := int64(0)
		bwarr.AscendLessThan(int64(5), func(item int64) bool {
			s += item
			return true
		})
	})

	assert.Equal(t, 2.0, allocs, "Expected 2 memory allocations during AscendLessThan") // nolint:testifylint
}

func TestBWArr_Allocs_AscendRange(t *testing.T) {
	bwarr := New[int64](int64Cmp, testAllocsSize)

	for i := range testAllocsSize {
		bwarr.Insert(int64(i))
	}

	const N = 100
	allocs := testing.AllocsPerRun(N, func() {
		s := int64(0)
		bwarr.AscendRange(int64(2), int64(8), func(item int64) bool {
			s += item
			return true
		})
	})

	assert.Equal(t, 2.0, allocs, "Expected 2 memory allocations during AscendRange") // nolint:testifylint
}

func TestBWArr_Allocs_Descend(t *testing.T) {
	bwarr := New[int64](int64Cmp, testAllocsSize)

	for i := range testAllocsSize {
		bwarr.Insert(int64(i))
	}

	const N = 100
	allocs := testing.AllocsPerRun(N, func() {
		s := int64(0)
		bwarr.Descend(func(item int64) bool {
			s += item
			return true
		})
	})

	assert.Equal(t, 2.0, allocs, "Expected 2 memory allocations during Descend") // nolint:testifylint
}

func TestBWArr_Allocs_DescendGreaterOrEqual(t *testing.T) {
	bwarr := New[int64](int64Cmp, testAllocsSize)

	for i := range testAllocsSize {
		bwarr.Insert(int64(i))
	}

	const N = 100
	allocs := testing.AllocsPerRun(N, func() {
		s := int64(0)
		bwarr.DescendGreaterOrEqual(int64(5), func(item int64) bool {
			s += item
			return true
		})
	})

	assert.Equal(t, 2.0, allocs, "Expected 2 memory allocations during DescendGreaterOrEqual") // nolint:testifylint
}

func TestBWArr_Allocs_DescendLessThan(t *testing.T) {
	bwarr := New[int64](int64Cmp, testAllocsSize)

	for i := range testAllocsSize {
		bwarr.Insert(int64(i))
	}

	const N = 100
	allocs := testing.AllocsPerRun(N, func() {
		s := int64(0)
		bwarr.DescendLessThan(int64(5), func(item int64) bool {
			s += item
			return true
		})
	})

	assert.Equal(t, 2.0, allocs, "Expected 2 memory allocations during DescendLessThan") // nolint:testifylint
}

func TestBWArr_Allocs_DescendRange(t *testing.T) {
	bwarr := New[int64](int64Cmp, testAllocsSize)

	for i := range testAllocsSize {
		bwarr.Insert(int64(i))
	}

	const N = 100
	allocs := testing.AllocsPerRun(N, func() {
		s := int64(0)
		bwarr.DescendRange(int64(2), int64(8), func(item int64) bool {
			s += item
			return true
		})
	})

	assert.Equal(t, 2.0, allocs, "Expected 2 memory allocations during DescendRange") // nolint:testifylint
}

func TestBWArr_Allocs_UnorderedWalk(t *testing.T) {
	bwarr := New[int64](int64Cmp, testAllocsSize)

	for i := range testAllocsSize {
		bwarr.Insert(int64(i))
	}

	const N = 100
	allocs := testing.AllocsPerRun(N, func() {
		s := int64(0)
		bwarr.UnorderedWalk(func(item int64) bool {
			s += item
			return true
		})
	})

	assert.Equal(t, 0.0, allocs, "Expected zero memory allocations during UnorderedWalk") // nolint:testifylint
}

func TestBWArr_Allocs_Compact(t *testing.T) {
	const testAllocsSize = 16
	bwarr := New[int64](int64Cmp, testAllocsSize)

	// Pre-populate with test data
	for i := range testAllocsSize {
		bwarr.Insert(int64(i))
	}

	allocs := testing.AllocsPerRun(1, func() {
		bwarr.Compact()
	})

	assert.Equal(t, 0.0, allocs, "Expected zero memory allocations during Compact operations") // nolint:testifylint
}

func TestBWArr_Allocs_ShouldBeLessAfterCompact(t *testing.T) {
	const testAllocsSize = 1024
	bwarr := New[int64](int64Cmp, testAllocsSize)

	for i := range testAllocsSize {
		bwarr.Insert(int64(i))
	}

	szBefore := calculateBWArrSize(bwarr)
	bwarr.Compact()
	szAfter := calculateBWArrSize(bwarr)

	const expectedReductionFactor = 1.9 // Struct root fields remain unchanged so a bit less than 2x reduction is expected

	assert.Greater(t, float64(szBefore)/float64(szAfter), expectedReductionFactor, "BWArr size should be ~ twice less after Compact")
}

// calculateBWArrSize calculates the total size of a BWArr struct including all nested fields
func calculateBWArrSize[T any](bwarr *BWArr[T]) int {
	size := int(unsafe.Sizeof(*bwarr))
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
