package bwarr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			from:     segment[int64]{elements: []int64{23, 0, 0, 42}, deleted: []bool{false, true, true, false}, deletedNum: 2},
			to:       &segment[int64]{elements: []int64{16, 32}, deleted: []bool{true, true}, deletedNum: 2},
			expected: &segment[int64]{elements: []int64{23, 42}, deleted: []bool{false, false}, deletedNum: 0},
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			demoteSegment(tt.from, tt.to)
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
			seg1:     segment[int64]{elements: []int64{23, 42}, deleted: []bool{false, false}, maxNonDeletedIdx: 1},
			seg2:     segment[int64]{elements: []int64{17, 37}, deleted: []bool{false, false}, maxNonDeletedIdx: 1},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: make([]bool, 4)},
			expected: segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: make([]bool, 4), maxNonDeletedIdx: 3},
		},
		{
			name:     "rewind from first",
			seg1:     segment[int64]{elements: []int64{3, 4}, deleted: []bool{false, false}, maxNonDeletedIdx: 1},
			seg2:     segment[int64]{elements: []int64{17, 37}, deleted: []bool{false, false}, maxNonDeletedIdx: 1},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: make([]bool, 4)},
			expected: segment[int64]{elements: []int64{3, 4, 17, 37}, deleted: make([]bool, 4), maxNonDeletedIdx: 3},
		},
		{
			name:     "two with one deleted element",
			seg1:     segment[int64]{elements: []int64{23, 42}, deleted: []bool{false, false}, maxNonDeletedIdx: 1},
			seg2:     segment[int64]{elements: []int64{17, 37}, deleted: []bool{false, true}, deletedNum: 1},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: make([]bool, 4)},
			expected: segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: []bool{false, false, true, false}, deletedNum: 1, maxNonDeletedIdx: 3},
		},
		{
			name:     "two with two deleted elements",
			seg1:     segment[int64]{elements: []int64{23, 42}, deleted: []bool{true, false}, deletedNum: 1, maxNonDeletedIdx: 1},
			seg2:     segment[int64]{elements: []int64{17, 37}, deleted: []bool{false, true}, deletedNum: 1},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: make([]bool, 4)},
			expected: segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: []bool{false, true, true, false}, deletedNum: 2, maxNonDeletedIdx: 3},
		},
		{
			name:     "if elements are equal, non-deleted must be first",
			seg1:     segment[int64]{elements: []int64{23, 42}, deleted: []bool{true, false}, deletedNum: 1, maxNonDeletedIdx: 1},
			seg2:     segment[int64]{elements: []int64{23, 42}, deleted: []bool{false, true}, deletedNum: 1},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: make([]bool, 4)},
			expected: segment[int64]{elements: []int64{23, 23, 42, 42}, deleted: []bool{false, true, false, true}, deletedNum: 2, maxNonDeletedIdx: 3},
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mergeSegments(tt.seg1, tt.seg2, int64Cmp, tt.result)
			require.Equal(t, tt.expected, *tt.result)
		})
	}
}

//nolint:exhaustruct
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
			seg:  segment[int64]{elements: []int64{23}, deleted: []bool{false}},
			val:  23,
			want: 0,
		},
		{
			name: "one not match",
			seg:  segment[int64]{elements: []int64{23}, deleted: []bool{false}},
			val:  42,
			want: -1,
		},
		{
			name: "in the middle",
			seg: segment[int64]{
				elements:         []int64{17, 23, 37, 42},
				deleted:          []bool{false, false, false, false},
				maxNonDeletedIdx: 3,
			},
			val:  23,
			want: 1,
		},
		{
			name: "in the beginning",
			seg: segment[int64]{
				elements:         []int64{17, 23, 37, 42},
				deleted:          []bool{false, false, false, false},
				maxNonDeletedIdx: 3,
			},
			val:  17,
			want: 0,
		},
		{
			name: "in the end",
			seg: segment[int64]{
				elements:         []int64{17, 23, 37, 42},
				deleted:          []bool{false, false, false, false},
				maxNonDeletedIdx: 3,
			},
			val:  42,
			want: 3,
		},
		{
			name: "with deleted",
			seg: segment[int64]{
				elements:         []int64{17, 23, 37, 42},
				deleted:          []bool{true, true, false, true},
				maxNonDeletedIdx: 2,
			},
			val:  37,
			want: 2,
		},
		{
			name: "with deleted not match",
			seg: segment[int64]{
				elements:         []int64{17, 23, 37, 42},
				deleted:          []bool{false, true, false, false},
				maxNonDeletedIdx: 3,
			},
			val:  23,
			want: -1,
		},
		{
			name: "with deleted postfix",
			seg: segment[int64]{
				elements:         []int64{17, 23, 37, 42, 49, 51, 69, 88},
				deleted:          []bool{false, false, false, true, true, true, true, true},
				maxNonDeletedIdx: 2,
			},
			val:  37,
			want: 2,
		},
		{
			name: "should find rightmost",
			seg: segment[int64]{
				elements:         []int64{17, 23, 23, 23, 37, 42, 49, 51},
				deleted:          []bool{false, false, false, false, false, false, false, false},
				maxNonDeletedIdx: 7,
			},
			val:  23,
			want: 3,
		},
		{
			name: "should find rightmost not deleted",
			seg: segment[int64]{
				elements:         []int64{17, 23, 23, 23, 37, 42, 49, 51},
				deleted:          []bool{false, false, true, true, false, false, false, false},
				maxNonDeletedIdx: 7,
			},
			val:  23,
			want: 1,
		},
		{
			name: "should find rightmost not deleted in the middle",
			seg: segment[int64]{
				elements:         []int64{17, 23, 23, 23, 37, 42, 49, 51},
				deleted:          []bool{false, true, false, true, false, false, false, false},
				maxNonDeletedIdx: 7,
			},
			val:  23,
			want: 2,
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
			seg:  segment[int64]{elements: []int64{23}, deleted: []bool{false}},
			val:  23,
			want: 0,
		},
		{
			name: "one greater",
			seg:  segment[int64]{elements: []int64{23}, deleted: []bool{false}},
			val:  11,
			want: 0,
		},
		{
			name: "first",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: []bool{false, false, false, false}, maxNonDeletedIdx: 3},
			val:  11,
			want: 0,
		},
		{
			name: "last",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: []bool{false, false, false, false}, maxNonDeletedIdx: 3},
			val:  42,
			want: 3,
		},
		{
			name: "in the middle",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: []bool{false, false, false, false}, maxNonDeletedIdx: 3},
			val:  30,
			want: 2,
		},
		{
			name: "in the middle with deleted",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: []bool{false, false, true, false}, maxNonDeletedIdx: 3},
			val:  30,
			want: 3,
		},
		{
			name: "all less",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: []bool{false, false, false, false}},
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
			seg:  segment[int64]{elements: []int64{23}, deleted: []bool{false}},
			val:  23,
			want: -1,
		},
		{
			name: "one less",
			seg:  segment[int64]{elements: []int64{23}, deleted: []bool{false}},
			val:  42,
			want: 0,
		},
		{
			name: "one greater",
			seg:  segment[int64]{elements: []int64{23}, deleted: []bool{false}},
			val:  11,
			want: -1,
		},
		{
			name: "last",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: []bool{false, false, false, false}, maxNonDeletedIdx: 3},
			val:  77,
			want: 3,
		},
		{
			name: "first",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: []bool{false, false, false, false}, maxNonDeletedIdx: 3},
			val:  11,
			want: -1,
		},
		{
			name: "last",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: []bool{false, false, false, false}, maxNonDeletedIdx: 3},
			val:  77,
			want: 3,
		},
		{
			name: "in the middle",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: []bool{false, false, false, false}, maxNonDeletedIdx: 3},
			val:  30,
			want: 1,
		},
		{
			name: "in the middle with deleted",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: []bool{false, true, false, false}, maxNonDeletedIdx: 3},
			val:  30,
			want: 0,
		},
		{
			name: "all deleted",
			seg:  segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: []bool{true, true, true, true}, maxNonDeletedIdx: -1},
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
		elements:         []int64{17, 23, 23, 23, 37, 42, 49, 51},
		deleted:          []bool{false, true, false, true, false, false, false, false},
		maxNonDeletedIdx: 7,
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
		deleted:  []bool{true, true, true, true},
	}
	assert.Equal(t, -1, seg.minNonDeletedIndex())
	assert.Equal(t, -1, seg.maxNonDeletedIndex())
}

func validateSegment[T any](t *testing.T, seg segment[T], cmp CmpFunc[T]) {
	require.Len(t, seg.elements, len(seg.deleted))
	deleted, firstNonDelIdx, lastNonDelIdx := 0, 0, len(seg.elements)-1
	metNonDel := false
	for i := range seg.elements {
		if seg.deleted[i] {
			deleted++
			continue
		}
		// If elements are equal, deleted must be after non-deleted;
		if i != 0 && cmp(seg.elements[i-1], seg.elements[i]) == 0 {
			if seg.deleted[i-1] {
				assert.False(t, seg.deleted[i], "At index %d: equal elements, but deleted comes before non-deleted", i)
			}
		}
		lastNonDelIdx = i
		if !metNonDel {
			firstNonDelIdx = i
			metNonDel = true
		}

		if i >= len(seg.elements)-1 || seg.deleted[i+1] {
			continue
		}
		assert.LessOrEqual(t, cmp(seg.elements[i], seg.elements[i+1]), 0)
	}
	assert.Equal(t, deleted, seg.deletedNum)
	assert.GreaterOrEqual(t, firstNonDelIdx, seg.minNonDeletedIdx)
	assert.LessOrEqual(t, lastNonDelIdx, seg.maxNonDeletedIdx)
}

func segmentsEqual[T any](t *testing.T, expected, actual segment[T]) {
	require.Len(t, expected.elements, len(expected.deleted))
	require.Len(t, actual.elements, len(expected.elements))
	require.Len(t, actual.deleted, len(expected.deleted))
	require.Equal(t, expected.deletedNum, actual.deletedNum)
	for i := range expected.elements {
		assert.Equal(t, expected.deleted[i], actual.deleted[i])
		if !expected.deleted[i] {
			assert.Equal(t, expected.elements[i], actual.elements[i])
		}
	}
	require.Equal(t, expected.minNonDeletedIdx, actual.minNonDeletedIdx)
	require.Equal(t, expected.maxNonDeletedIdx, actual.maxNonDeletedIdx)
}
