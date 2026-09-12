package bwarr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolsToLayeredBitSet(bools []bool) *layeredBitSet {
	bs := newLayeredBitSet(len(bools))
	for i, b := range bools {
		if b {
			bs.Set(i)
		}
	}
	return bs
}

//nolint:exhaustruct
func Test_demoteSegment(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		from     segment[int64]
		to       *segment[int64]
		expected *segment[int64]
	}{
		{
			name:     "demote 4 to 2",
			from:     segment[int64]{elements: []int64{23, 0, 0, 42}, deleted: boolsToLayeredBitSet([]bool{false, true, true, false}), deletedNum: 2},
			to:       &segment[int64]{elements: []int64{16, 32}, deleted: boolsToLayeredBitSet([]bool{true, true}), deletedNum: 2},
			expected: &segment[int64]{elements: []int64{23, 42}, deleted: newLayeredBitSet(2), deletedNum: 0},
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			demoteSegment(&tt.from, tt.to)
		})
	}
}

func Test_calculateWhiteSegmentsQuantity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		capacity  int
		want      int
		wantPanic bool
	}{
		{
			name:      "zero capacity",
			capacity:  0,
			want:      0,
			wantPanic: false,
		},
		{
			name:      "power of two capacity",
			capacity:  8,
			want:      4,
			wantPanic: false,
		},
		{
			name:      "border capacity",
			capacity:  31,
			want:      5,
			wantPanic: false,
		},
		{
			name:      "negative capacity",
			capacity:  -1,
			want:      0,
			wantPanic: true,
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantPanic {
				require.Panics(t, func() { calculateWhiteSegmentsQuantity(tt.capacity) })
			} else {
				require.Equal(t, tt.want, calculateWhiteSegmentsQuantity(tt.capacity))
			}
		})
	}
}

//nolint:exhaustruct
func Test_mergeSegments(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		seg1     segment[int64]
		seg2     segment[int64]
		result   *segment[int64]
		expected segment[int64]
	}{
		{
			name:     "two elements",
			seg1:     segment[int64]{elements: []int64{23, 42}, deleted: newLayeredBitSet(2)},
			seg2:     segment[int64]{elements: []int64{17, 37}, deleted: newLayeredBitSet(2)},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: newLayeredBitSet(4)},
			expected: segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: newLayeredBitSet(4)},
		},
		{
			name:     "rewind from first",
			seg1:     segment[int64]{elements: []int64{3, 4}, deleted: newLayeredBitSet(2)},
			seg2:     segment[int64]{elements: []int64{17, 37}, deleted: newLayeredBitSet(2)},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: newLayeredBitSet(4)},
			expected: segment[int64]{elements: []int64{3, 4, 17, 37}, deleted: newLayeredBitSet(4)},
		},
		{
			name:     "two with one deleted element",
			seg1:     segment[int64]{elements: []int64{23, 42}, deleted: newLayeredBitSet(2)},
			seg2:     segment[int64]{elements: []int64{17, 37}, deleted: boolsToLayeredBitSet([]bool{false, true}), deletedNum: 1},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: newLayeredBitSet(4)},
			expected: segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: boolsToLayeredBitSet([]bool{false, false, true, false}), deletedNum: 1},
		},
		{
			name:     "two with two deleted elements",
			seg1:     segment[int64]{elements: []int64{23, 42}, deleted: boolsToLayeredBitSet([]bool{true, false}), deletedNum: 1},
			seg2:     segment[int64]{elements: []int64{17, 37}, deleted: boolsToLayeredBitSet([]bool{false, true}), deletedNum: 1},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: newLayeredBitSet(4)},
			expected: segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: boolsToLayeredBitSet([]bool{false, true, true, false}), deletedNum: 2},
		},
		{
			name:     "if elements are equal, non-deleted must be first",
			seg1:     segment[int64]{elements: []int64{23, 42}, deleted: boolsToLayeredBitSet([]bool{true, false}), deletedNum: 1},
			seg2:     segment[int64]{elements: []int64{23, 42}, deleted: boolsToLayeredBitSet([]bool{false, true}), deletedNum: 1},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: newLayeredBitSet(4)},
			expected: segment[int64]{elements: []int64{23, 23, 42, 42}, deleted: boolsToLayeredBitSet([]bool{false, true, false, true}), deletedNum: 2},
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Copy seg2 into the second half of result
			seg2Len := len(tt.seg2.elements)
			copy(tt.result.elements[seg2Len:], tt.seg2.elements)
			for i := range seg2Len {
				if tt.seg2.deleted.Get(i) {
					tt.result.deleted.Set(seg2Len + i)
				}
			}
			tt.result.deletedNum = tt.seg2.deletedNum
			// Merge seg1 into result starting at position seg2Len
			mergeSegments(&tt.seg1, tt.result, int64Cmp, seg2Len)
			segmentsEqual(t, tt.expected, *tt.result)
		})
	}
}

func Test_mergeSegmentsForDel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		seg1     segment[int64]
		seg2     segment[int64]
		result   *segment[int64]
		expected segment[int64]
	}{
		{
			name:     "two elements",
			seg1:     segment[int64]{elements: []int64{23, 42}, deleted: newLayeredBitSet(2)},
			seg2:     segment[int64]{elements: []int64{17, 37}, deleted: newLayeredBitSet(2)},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: newLayeredBitSet(4)},
			expected: segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: newLayeredBitSet(4)},
		},
		{
			name:     "rewind from first",
			seg1:     segment[int64]{elements: []int64{3, 4}, deleted: newLayeredBitSet(2)},
			seg2:     segment[int64]{elements: []int64{17, 37}, deleted: newLayeredBitSet(2)},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: newLayeredBitSet(4)},
			expected: segment[int64]{elements: []int64{3, 4, 17, 37}, deleted: newLayeredBitSet(4)},
		},
		{
			name:     "two with one deleted element",
			seg1:     segment[int64]{elements: []int64{23, 42}, deleted: newLayeredBitSet(2)},
			seg2:     segment[int64]{elements: []int64{17, 37}, deleted: boolsToLayeredBitSet([]bool{false, true}), deletedNum: 1},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: newLayeredBitSet(4)},
			expected: segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: boolsToLayeredBitSet([]bool{false, false, true, false}), deletedNum: 1},
		},
		{
			name:     "two with two deleted elements",
			seg1:     segment[int64]{elements: []int64{23, 42}, deleted: boolsToLayeredBitSet([]bool{true, false}), deletedNum: 1},
			seg2:     segment[int64]{elements: []int64{17, 37}, deleted: boolsToLayeredBitSet([]bool{false, true}), deletedNum: 1},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: newLayeredBitSet(4)},
			expected: segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: boolsToLayeredBitSet([]bool{false, true, true, false}), deletedNum: 2},
		},
		{
			name:     "if elements are equal, non-deleted must be first",
			seg1:     segment[int64]{elements: []int64{23, 42}, deleted: boolsToLayeredBitSet([]bool{true, false}), deletedNum: 1},
			seg2:     segment[int64]{elements: []int64{23, 42}, deleted: boolsToLayeredBitSet([]bool{false, true}), deletedNum: 1},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: newLayeredBitSet(4)},
			expected: segment[int64]{elements: []int64{23, 23, 42, 42}, deleted: boolsToLayeredBitSet([]bool{false, true, false, true}), deletedNum: 2},
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Copy seg2 into the second half of result
			seg2Len := len(tt.seg2.elements)
			copy(tt.result.elements[seg2Len:], tt.seg2.elements)
			for i := range seg2Len {
				if tt.seg2.deleted.Get(i) {
					tt.result.deleted.Set(seg2Len + i)
				}
			}
			tt.result.deletedNum = tt.seg2.deletedNum
			// Merge seg1 into result starting at position seg2Len
			mergeSegmentsDirty(&tt.seg1, tt.result, int64Cmp, seg2Len, true)
			segmentsEqual(t, tt.expected, *tt.result)
		})
	}
}

// Fully deleted segments cannot be active in a BWArr (occupancy invariant), but the
// find methods guard against them so they stay safe for any segment state.
//
//nolint:exhaustruct
func Test_findsOnFullyDeletedSegment(t *testing.T) {
	t.Parallel()
	seg := segment[int64]{elements: []int64{23, 42}, deleted: boolsToLayeredBitSet([]bool{true, true}), deletedNum: 2} //nolint:exhaustruct
	assert.Equal(t, -1, seg.findRightmostNotDeleted(int64Cmp, 23))
	assert.Equal(t, -1, seg.findGTOE(int64Cmp, 23))
	assert.Equal(t, -1, seg.findLess(int64Cmp, 43))
}

// Exercises the defensive guard in the duplicate-handling branch of the binary search:
// reachable only when the bitset's firstUnset cache understates the first live element.
func Test_findRightmostNotDeletedCorruptedCache(t *testing.T) {
	t.Parallel()
	seg := segment[int64]{elements: []int64{7, 7, 7, 7}, deleted: boolsToLayeredBitSet([]bool{true, true, true, false}), deletedNum: 3} //nolint:exhaustruct
	seg.deleted.firstUnset = 0                                                                                                          // Below the real first live element (3).
	assert.Equal(t, -1, seg.findRightmostNotDeleted(int64Cmp, 7))
}

func Test_findRightmostNotDeleted(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		seg  segment[int64]
		val  int64
		want int
	}{
		{
			name: "one match",
			seg:  segment[int64]{elements: []int64{23}, deleted: newLayeredBitSet(1)},
			val:  23,
			want: 0,
		},
		{
			name: "one not match",
			seg:  segment[int64]{elements: []int64{23}, deleted: newLayeredBitSet(1)},
			val:  42,
			want: -1,
		},
		{
			name: "in the middle",
			seg: segment[int64]{
				elements: []int64{17, 23, 37, 42},
				deleted:  newLayeredBitSet(4),
			},
			val:  23,
			want: 1,
		},
		{
			name: "in the beginning",
			seg: segment[int64]{
				elements: []int64{17, 23, 37, 42},
				deleted:  newLayeredBitSet(4),
			},
			val:  17,
			want: 0,
		},
		{
			name: "in the end",
			seg: segment[int64]{
				elements: []int64{17, 23, 37, 42},
				deleted:  newLayeredBitSet(4),
			},
			val:  42,
			want: 3,
		},
		{
			name: "with deleted",
			seg: segment[int64]{
				elements: []int64{17, 23, 37, 42},
				deleted:  boolsToLayeredBitSet([]bool{true, true, false, true}),
			},
			val:  37,
			want: 2,
		},
		{
			name: "with deleted not match",
			seg: segment[int64]{
				elements: []int64{17, 23, 37, 42},
				deleted:  boolsToLayeredBitSet([]bool{false, true, false, false}),
			},
			val:  23,
			want: -1,
		},
		{
			name: "with deleted postfix",
			seg: segment[int64]{
				elements: []int64{17, 23, 37, 42, 49, 51, 69, 88},
				deleted:  boolsToLayeredBitSet([]bool{false, false, false, true, true, true, true, true}),
			},
			val:  37,
			want: 2,
		},
		{
			name: "should find rightmost",
			seg: segment[int64]{
				elements: []int64{17, 23, 23, 23, 37, 42, 49, 51},
				deleted:  newLayeredBitSet(8),
			},
			val:  23,
			want: 3,
		},
		{
			name: "should find rightmost not deleted",
			seg: segment[int64]{
				elements: []int64{17, 23, 23, 23, 37, 42, 49, 51},
				deleted:  boolsToLayeredBitSet([]bool{false, false, true, true, false, false, false, false}),
			},
			val:  23,
			want: 1,
		},
		{
			name: "should find rightmost not deleted in the middle",
			seg: segment[int64]{
				elements: []int64{17, 23, 23, 23, 37, 42, 49, 51},
				deleted:  boolsToLayeredBitSet([]bool{false, true, false, true, false, false, false, false}),
			},
			val:  23,
			want: 2,
		},
		{
			name: "live equal before a long deleted equal run",
			seg: segment[int64]{
				elements: []int64{1, 2, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 9, 9},
				deleted: boolsToLayeredBitSet([]bool{
					false, false, false, false, true, true, true, true, true, true, true, true, true, true, false, false,
				}),
				deletedNum: 10,
			},
			val:  5,
			want: 3,
		},
		{
			name: "only smaller live before a deleted equal run",
			seg: segment[int64]{
				elements: []int64{1, 2, 5, 5, 5, 5, 5, 5, 9, 9},
				deleted: boolsToLayeredBitSet([]bool{
					false, false, true, true, true, true, true, true, false, false,
				}),
				deletedNum: 6,
			},
			val:  5,
			want: -1,
		},
		{
			name: "deleted equal run at the very beginning",
			seg: segment[int64]{
				elements:   []int64{5, 5, 5, 5, 9, 9, 9, 9},
				deleted:    boolsToLayeredBitSet([]bool{true, true, true, true, false, false, false, false}),
				deletedNum: 4,
			},
			val:  5,
			want: -1,
		},
		{
			name: "live equal, deleted smaller, then deleted equal run",
			seg: segment[int64]{
				elements:   []int64{4, 5, 5, 5, 5, 5, 5, 8},
				deleted:    boolsToLayeredBitSet([]bool{true, false, true, true, true, true, true, false}),
				deletedNum: 6,
			},
			val:  5,
			want: 1,
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tt.want, tt.seg.findRightmostNotDeleted(int64Cmp, tt.val), "searchInSegment(%v, %v)", tt.seg, tt.val)
		})
	}
}

//nolint:exhaustruct
func Test_segment_findGTOE(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		seg  segment[int64]
		val  int64
		want int
	}{
		{
			name: "one match",
			seg:  segment[int64]{elements: []int64{23}, deleted: newLayeredBitSet(1)},
			val:  23,
			want: 0,
		},
		{
			name: "one greater",
			seg:  segment[int64]{elements: []int64{23}, deleted: newLayeredBitSet(1)},
			val:  11,
			want: 0,
		},
		{
			name: "first",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: newLayeredBitSet(4)},
			val:  11,
			want: 0,
		},
		{
			name: "last",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: newLayeredBitSet(4)},
			val:  42,
			want: 3,
		},
		{
			name: "in the middle",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: newLayeredBitSet(4)},
			val:  30,
			want: 2,
		},
		{
			name: "in the middle with deleted",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: boolsToLayeredBitSet([]bool{false, false, true, false})},
			val:  30,
			want: 3,
		},
		{
			name: "all less",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: newLayeredBitSet(4)},
			val:  101,
			want: -1,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tt.want, tt.seg.findGTOE(int64Cmp, tt.val), "searchInSegment(%v, %v)", tt.seg, tt.val)
		})
	}
}

//nolint:exhaustruct
func Test_segment_findLess(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		seg  segment[int64]
		val  int64
		want int
	}{
		{
			name: "one equal",
			seg:  segment[int64]{elements: []int64{23}, deleted: newLayeredBitSet(1)},
			val:  23,
			want: -1,
		},
		{
			name: "one less",
			seg:  segment[int64]{elements: []int64{23}, deleted: newLayeredBitSet(1)},
			val:  42,
			want: 0,
		},
		{
			name: "one greater",
			seg:  segment[int64]{elements: []int64{23}, deleted: newLayeredBitSet(1)},
			val:  11,
			want: -1,
		},
		{
			name: "last",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: newLayeredBitSet(4)},
			val:  77,
			want: 3,
		},
		{
			name: "first",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: newLayeredBitSet(4)},
			val:  11,
			want: -1,
		},
		{
			name: "last2",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: newLayeredBitSet(4)},
			val:  77,
			want: 3,
		},
		{
			name: "in the middle",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: newLayeredBitSet(4)},
			val:  30,
			want: 1,
		},
		{
			name: "in the middle with deleted",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: boolsToLayeredBitSet([]bool{false, true, false, false})},
			val:  30,
			want: 0,
		},
		{
			name: "all deleted",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: boolsToLayeredBitSet([]bool{true, true, true, true})},
			val:  23,
			want: -1,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tt.want, tt.seg.findLess(int64Cmp, tt.val), "searchInSegment(%v, %v)", tt.seg, tt.val)
		})
	}
}

func Test_segment_nextNonDeletedAfter(t *testing.T) {
	t.Parallel()
	seg := segment[int64]{ // nolint:exhaustruct
		elements: []int64{17, 23, 23, 23, 37, 42, 49, 51},
		deleted:  boolsToLayeredBitSet([]bool{false, true, false, true, false, false, false, false}),
	}
	tests := []struct {
		name string
		idx  int
		want int
	}{
		{
			name: "zero",
			idx:  -1,
			want: 0,
		},
		{
			name: "zero to second",
			idx:  0,
			want: 2,
		},
		{
			name: "first to second",
			idx:  1,
			want: 2,
		},
		{
			name: "second to forth",
			idx:  2,
			want: 4,
		},
		{
			name: "third to forth",
			idx:  3,
			want: 4,
		},
		{
			name: "the very last",
			idx:  7,
			want: 8,
		},
		{
			name: "after the very last",
			idx:  8,
			want: 8,
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, seg.nextNonDeletedAfter(tt.idx))
		})
	}
}

func Test_segment_AllDeleted(t *testing.T) {
	t.Parallel()
	seg := segment[int64]{ // nolint:exhaustruct
		elements: []int64{17, 23, 42, 51},
		deleted:  boolsToLayeredBitSet([]bool{true, true, true, true}),
	}
	assert.Equal(t, -1, seg.minNonDeletedIndex())
	assert.Equal(t, -1, seg.maxNonDeletedIndex())
}

//nolint:exhaustruct
func Test_segment_min(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		seg  segment[int64]
		want int
	}{
		{
			name: "single element",
			seg: segment[int64]{
				elements: []int64{42},
				deleted:  newLayeredBitSet(1),
			},
			want: 0,
		},
		{
			name: "two elements - first is min",
			seg: segment[int64]{
				elements: []int64{17, 42},
				deleted:  newLayeredBitSet(2),
			},
			want: 0,
		},
		{
			name: "two equal elements - should return rightmost (FIFO)",
			seg: segment[int64]{
				elements: []int64{23, 23},
				deleted:  newLayeredBitSet(2),
			},
			want: 1,
		},
		{
			name: "three equal elements - should return rightmost",
			seg: segment[int64]{
				elements: []int64{23, 23, 23},
				deleted:  newLayeredBitSet(3),
			},
			want: 2,
		},
		{
			name: "equal elements with deleted after - should return last non-deleted",
			seg: segment[int64]{
				elements: []int64{23, 23, 23},
				deleted:  boolsToLayeredBitSet([]bool{false, false, true}),
			},
			want: 1,
		},
		{
			name: "equal elements with multiple deleted after - should return last non-deleted",
			seg: segment[int64]{
				elements: []int64{23, 23, 23, 23},
				deleted:  boolsToLayeredBitSet([]bool{false, false, true, true}),
			},
			want: 1,
		},
		{
			name: "sorted array - minimum is first",
			seg: segment[int64]{
				elements: []int64{17, 23, 37, 42},
				deleted:  newLayeredBitSet(4),
			},
			want: 0,
		},
		{
			name: "sorted array with equal minimums",
			seg: segment[int64]{
				elements: []int64{17, 17, 23, 37, 42},
				deleted:  newLayeredBitSet(5),
			},
			want: 1,
		},
		{
			name: "sorted array with equal minimums and deleted after",
			seg: segment[int64]{
				elements: []int64{17, 17, 17, 23, 37, 42},
				deleted:  boolsToLayeredBitSet([]bool{false, false, true, false, false, false}),
			},
			want: 1,
		},
		{
			name: "sorted array with larger elements after equal mins",
			seg: segment[int64]{
				elements: []int64{5, 5, 5, 10, 20, 30},
				deleted:  newLayeredBitSet(6),
			},
			want: 2,
		},
		{
			name: "first element deleted - min is second",
			seg: segment[int64]{
				elements: []int64{17, 23, 37, 42},
				deleted:  boolsToLayeredBitSet([]bool{true, false, false, false}),
			},
			want: 1,
		},
		{
			name: "sparse deleted elements",
			seg: segment[int64]{
				elements: []int64{17, 23, 37, 42},
				deleted:  boolsToLayeredBitSet([]bool{true, false, true, false}),
			},
			want: 1,
		},
		{
			name: "all equal non-deleted with deleted suffix",
			seg: segment[int64]{
				elements: []int64{10, 10, 10, 10, 10},
				deleted:  boolsToLayeredBitSet([]bool{false, false, false, true, true}),
			},
			want: 2,
		},
		{
			name: "complex case - equal mins, then deleted, then larger values",
			seg: segment[int64]{
				elements: []int64{5, 5, 5, 5, 10, 15, 20},
				deleted:  boolsToLayeredBitSet([]bool{false, false, false, true, false, false, false}),
			},
			want: 2,
		},
		{
			name: "single non-deleted in middle of deleted",
			seg: segment[int64]{
				elements: []int64{17, 23, 37, 42},
				deleted:  boolsToLayeredBitSet([]bool{true, true, false, true}),
			},
			want: 2,
		},
		{
			name: "equal elements at start with larger after",
			seg: segment[int64]{
				elements: []int64{1, 1, 2, 3, 4},
				deleted:  newLayeredBitSet(5),
			},
			want: 1,
		},
		{
			name: "invariant test - deleted only after equal non-deleted",
			seg: segment[int64]{
				elements: []int64{10, 10, 10, 10, 20, 30},
				deleted:  boolsToLayeredBitSet([]bool{false, false, true, true, false, false}),
			},
			want: 1,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.seg.min(int64Cmp)
			assert.Equalf(t, tt.want, got, "min() returned index %d, want %d", got, tt.want)

			// Verify the returned index is not deleted
			if got >= 0 && got < len(tt.seg.elements) {
				assert.Falsef(t, tt.seg.deleted.Get(got), "min() returned deleted element at index %d", got)
			}
		})
	}
}

func validateSegment[T any](t *testing.T, seg segment[T], cmp CmpFunc[T]) {
	deleted, firstNonDelIdx, lastNonDelIdx := 0, 0, len(seg.elements)-1
	metNonDel := false
	for i := range seg.elements {
		if seg.deleted.Get(i) {
			deleted++
			continue
		}
		// If elements are equal, deleted must be after non-deleted;
		if i != 0 && cmp(seg.elements[i-1], seg.elements[i]) == 0 {
			if seg.deleted.Get(i - 1) {
				assert.Failf(t, "Order constraint", "at index %d and %d: equal elements %d, but deleted comes before non-deleted", i-1, i, seg.elements[i])
			}
		}
		lastNonDelIdx = i
		if !metNonDel {
			firstNonDelIdx = i
			metNonDel = true
		}

		if i >= len(seg.elements)-1 || seg.deleted.Get(i+1) {
			continue
		}
		assert.LessOrEqual(t, cmp(seg.elements[i], seg.elements[i+1]), 0)
	}
	assert.Equal(t, deleted, seg.deletedNum)
	assert.GreaterOrEqual(t, firstNonDelIdx, seg.minNonDeletedIndex())
	assert.LessOrEqual(t, lastNonDelIdx, seg.maxNonDeletedIndex())
}

func segmentsEqual[T any](t *testing.T, expected, actual segment[T]) {
	require.Len(t, actual.elements, len(expected.elements))
	require.Equal(t, expected.deletedNum, actual.deletedNum)
	for i := range expected.elements {
		assert.Equal(t, expected.deleted.Get(i), actual.deleted.Get(i))
		if !expected.deleted.Get(i) {
			assert.Equal(t, expected.elements[i], actual.elements[i])
		}
	}
}
