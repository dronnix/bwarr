package bwarr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const testAllocsSize = 42 // 5 Segments should be enough;

func TestBWArr_Allocs_New(t *testing.T) {
	const expectedAllocs = 18
	// Slice of segments - 1, BWArr struct - 1 --> 2;
	// 6 segments, each contains two slices: elements and deleted flags --> 12;
	// 2 black segments, each contains two slices: elements and deleted flags --> 4;
	// Total: 2 + 12 + 4 = 18;
	allocs := testing.AllocsPerRun(1, func() {
		bwarr := New[int64](int64Cmp, testAllocsSize)
		_ = bwarr
	})

	assert.InDelta(t, float64(expectedAllocs), allocs, 0, "Allocation count mismatch")
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

	allocs := testing.AllocsPerRun(1, func() {
		bwarr := NewFromSlice[int64](int64Cmp, testSlice)
		_ = bwarr
	})

	assert.InDelta(t, float64(expectedAllocs), allocs, 0, "Allocation count mismatch")
}
