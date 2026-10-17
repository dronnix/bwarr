package bwarr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Cmp(a, b int64) int {
	return int(a - b)
}

func makeSegmentHelper1(elements []int64, deleted []bool) *segment[int64] {
	l := len(elements)
	s := segment[int64]{
		elements: elements,
		deleted:  deleted,
		fenwick:  make([]int, l+1),
	}
	s.initFenwick(int64Cmp)
	return &s
}

func makeSegmentHelper2(elements []int64) *segment[int64] {
	return makeSegmentHelper1(elements, make([]bool, len(elements)))
}

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
			from:     *makeSegmentHelper1([]int64{23, 0, 0, 42}, []bool{false, true, true, false}),
			to:       makeSegmentHelper1([]int64{16, 32}, []bool{true, true}),
			expected: makeSegmentHelper2([]int64{23, 42}),
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
			seg1:     *makeSegmentHelper2([]int64{23, 42}),
			seg2:     *makeSegmentHelper2([]int64{17, 37}),
			result:   makeSegmentHelper2(make([]int64, 4)),
			expected: *makeSegmentHelper2([]int64{17, 23, 37, 42}),
		},
		{
			name:     "rewind from first",
			seg1:     *makeSegmentHelper2([]int64{3, 4}),
			seg2:     *makeSegmentHelper2([]int64{17, 37}),
			result:   makeSegmentHelper2(make([]int64, 4)),
			expected: *makeSegmentHelper2([]int64{3, 4, 17, 37}),
		},
		{
			name:     "two with one deleted element",
			seg1:     *makeSegmentHelper2([]int64{23, 42}),
			seg2:     *makeSegmentHelper1([]int64{17, 37}, []bool{false, true}),
			result:   makeSegmentHelper2(make([]int64, 4)),
			expected: *makeSegmentHelper1([]int64{17, 23, 37, 42}, []bool{false, false, true, false}),
		},
		{
			name:     "two with two deleted elements",
			seg1:     *makeSegmentHelper1([]int64{23, 42}, []bool{true, false}),
			seg2:     *makeSegmentHelper1([]int64{17, 37}, []bool{false, true}),
			result:   makeSegmentHelper2(make([]int64, 4)),
			expected: *makeSegmentHelper1([]int64{17, 23, 37, 42}, []bool{false, true, true, false}),
		},
		{
			name:     "if elements are equal, non-deleted must be first",
			seg1:     *makeSegmentHelper1([]int64{23, 42}, []bool{true, false}),
			seg2:     *makeSegmentHelper1([]int64{23, 42}, []bool{false, true}),
			result:   makeSegmentHelper2(make([]int64, 4)),
			expected: *makeSegmentHelper1([]int64{23, 23, 42, 42}, []bool{false, true, false, true}),
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Copy seg2 into the second half of result
			seg2Len := len(tt.seg2.elements)
			copy(tt.result.elements[seg2Len:], tt.seg2.elements)
			copy(tt.result.deleted[seg2Len:], tt.seg2.deleted)
			// Merge seg1 into result starting at position seg2Len
			mergeSegmentsDirty(&tt.seg1, tt.result, int64Cmp, seg2Len, false)
			tt.result.initFenwick(int64Cmp) // The merge leaves the fenwick sums dirty, exactly as in Insert.
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
			seg1:     *makeSegmentHelper2([]int64{23, 42}),
			seg2:     *makeSegmentHelper2([]int64{17, 37}),
			result:   makeSegmentHelper2(make([]int64, 4)),
			expected: *makeSegmentHelper2([]int64{17, 23, 37, 42}),
		},
		{
			name:     "rewind from first",
			seg1:     *makeSegmentHelper2([]int64{3, 4}),
			seg2:     *makeSegmentHelper2([]int64{17, 37}),
			result:   makeSegmentHelper2(make([]int64, 4)),
			expected: *makeSegmentHelper2([]int64{3, 4, 17, 37}),
		},
		{
			name:     "two with one deleted element",
			seg1:     *makeSegmentHelper2([]int64{23, 42}),
			seg2:     *makeSegmentHelper1([]int64{17, 37}, []bool{false, true}),
			result:   makeSegmentHelper2(make([]int64, 4)),
			expected: *makeSegmentHelper1([]int64{17, 23, 37, 42}, []bool{false, false, true, false}),
		},
		{
			name:     "two with two deleted elements",
			seg1:     *makeSegmentHelper1([]int64{23, 42}, []bool{true, false}),
			seg2:     *makeSegmentHelper1([]int64{17, 37}, []bool{false, true}),
			result:   makeSegmentHelper2(make([]int64, 4)),
			expected: *makeSegmentHelper1([]int64{17, 23, 37, 42}, []bool{false, true, true, false}),
		},
		{
			name:     "if elements are equal, non-deleted must be first",
			seg1:     *makeSegmentHelper1([]int64{23, 42}, []bool{true, false}),
			seg2:     *makeSegmentHelper1([]int64{23, 42}, []bool{false, true}),
			result:   makeSegmentHelper2(make([]int64, 4)),
			expected: *makeSegmentHelper1([]int64{23, 23, 42, 42}, []bool{false, true, false, true}),
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Copy seg2 into the second half of result
			seg2Len := len(tt.seg2.elements)
			copy(tt.result.elements[seg2Len:], tt.seg2.elements)
			copy(tt.result.deleted[seg2Len:], tt.seg2.deleted)
			// Merge seg1 into result starting at position seg2Len
			mergeSegmentsDirty(&tt.seg1, tt.result, int64Cmp, seg2Len, true)
			tt.result.initFenwick(int64Cmp) // The merge leaves the fenwick sums dirty, exactly as in the delete consolidation.
			segmentsEqual(t, tt.expected, *tt.result)
		})
	}
}

// Fully deleted segments cannot be active in a BWArr (occupancy invariant), but the
// find methods guard against them so they stay safe for any segment state.
func Test_findsOnFullyDeletedSegment(t *testing.T) {
	t.Parallel()
	seg := makeSegmentHelper1([]int64{23, 42}, []bool{true, true})
	assert.Equal(t, -1, seg.findRightmostNotDeleted(int64Cmp, 23))
	assert.Equal(t, -1, seg.findGTOE(int64Cmp, 23))
	assert.Equal(t, -1, seg.findLess(int64Cmp, 43))
}

func Test_findRightmostNotDeleted(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		seg  *segment[int64]
		val  int64
		want int
	}{
		{
			name: "one match",
			seg:  makeSegmentHelper2([]int64{23}),
			val:  23,
			want: 0,
		},
		{
			name: "one not match",
			seg:  makeSegmentHelper2([]int64{23}),
			val:  42,
			want: -1,
		},
		{
			name: "in the middle",
			seg:  makeSegmentHelper2([]int64{17, 23, 37, 42}),
			val:  23,
			want: 1,
		},
		{
			name: "in the beginning",
			seg:  makeSegmentHelper2([]int64{17, 23, 37, 42}),
			val:  17,
			want: 0,
		},
		{
			name: "in the end",
			seg:  makeSegmentHelper2([]int64{17, 23, 37, 42}),
			val:  42,
			want: 3,
		},
		{
			name: "with deleted",
			seg:  makeSegmentHelper1([]int64{17, 23, 37, 42}, []bool{true, true, false, true}),
			val:  37,
			want: 2,
		},
		{
			name: "with deleted not match",
			seg:  makeSegmentHelper1([]int64{17, 23, 37, 42}, []bool{false, true, false, false}),
			val:  23,
			want: -1,
		},
		{
			name: "with deleted postfix",
			seg: makeSegmentHelper1(
				[]int64{17, 23, 37, 42, 49, 51, 69, 88},
				[]bool{false, false, false, true, true, true, true, true},
			),
			val:  37,
			want: 2,
		},
		{
			name: "should find rightmost",
			seg:  makeSegmentHelper2([]int64{17, 23, 23, 23, 37, 42, 49, 51}),
			val:  23,
			want: 3,
		},
		{
			name: "should find rightmost not deleted",
			seg: makeSegmentHelper1(
				[]int64{17, 23, 23, 23, 37, 42, 49, 51},
				[]bool{false, false, true, true, false, false, false, false},
			),
			val:  23,
			want: 1,
		},
		{
			name: "should find rightmost not deleted in the middle",
			seg: makeSegmentHelper1(
				[]int64{17, 23, 23, 23, 37, 42, 49, 51},
				[]bool{false, true, false, true, false, false, false, false},
			),
			val:  23,
			want: 2,
		},
		{
			name: "live equal before a long deleted equal run",
			seg: makeSegmentHelper1(
				[]int64{1, 2, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 9, 9},
				[]bool{
					false, false, false, false, true, true, true, true, true, true, true, true, true, true, false, false,
				},
			),
			val:  5,
			want: 3,
		},
		{
			name: "only smaller live before a deleted equal run",
			seg: makeSegmentHelper1(
				[]int64{1, 2, 5, 5, 5, 5, 5, 5, 9, 9},
				[]bool{false, false, true, true, true, true, true, true, false, false},
			),
			val:  5,
			want: -1,
		},
		{
			name: "deleted equal run at the very beginning",
			seg: makeSegmentHelper1(
				[]int64{5, 5, 5, 5, 9, 9, 9, 9},
				[]bool{true, true, true, true, false, false, false, false},
			),
			val:  5,
			want: -1,
		},
		{
			name: "live equal, deleted smaller, then deleted equal run",
			seg: makeSegmentHelper1(
				[]int64{4, 5, 5, 5, 5, 5, 5, 8},
				[]bool{true, false, true, true, true, true, true, false},
			),
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

func Test_segment_findGTOE(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		seg  *segment[int64]
		val  int64
		want int
	}{
		{
			name: "one match",
			seg:  makeSegmentHelper2([]int64{23}),
			val:  23,
			want: 0,
		},
		{
			name: "one greater",
			seg:  makeSegmentHelper2([]int64{23}),
			val:  11,
			want: 0,
		},
		{
			name: "first",
			seg:  makeSegmentHelper2([]int64{17, 23, 37, 42}),
			val:  11,
			want: 0,
		},
		{
			name: "last",
			seg:  makeSegmentHelper2([]int64{17, 23, 37, 42}),
			val:  42,
			want: 3,
		},
		{
			name: "in the middle",
			seg:  makeSegmentHelper2([]int64{17, 23, 37, 42}),
			val:  30,
			want: 2,
		},
		{
			name: "in the middle with deleted",
			seg:  makeSegmentHelper1([]int64{17, 23, 37, 42}, []bool{false, false, true, false}),
			val:  30,
			want: 3,
		},
		{
			name: "all less",
			seg:  makeSegmentHelper2([]int64{17, 23, 37, 42}),
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

func Test_segment_findLess(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		seg  *segment[int64]
		val  int64
		want int
	}{
		{
			name: "one equal",
			seg:  makeSegmentHelper2([]int64{23}),
			val:  23,
			want: -1,
		},
		{
			name: "one less",
			seg:  makeSegmentHelper2([]int64{23}),
			val:  42,
			want: 0,
		},
		{
			name: "one greater",
			seg:  makeSegmentHelper2([]int64{23}),
			val:  11,
			want: -1,
		},
		{
			name: "last",
			seg:  makeSegmentHelper2([]int64{17, 23, 37, 42}),
			val:  77,
			want: 3,
		},
		{
			name: "first",
			seg:  makeSegmentHelper2([]int64{17, 23, 37, 42}),
			val:  11,
			want: -1,
		},
		{
			name: "last2",
			seg:  makeSegmentHelper2([]int64{17, 23, 37, 42}),
			val:  77,
			want: 3,
		},
		{
			name: "in the middle",
			seg:  makeSegmentHelper2([]int64{17, 23, 37, 42}),
			val:  30,
			want: 1,
		},
		{
			name: "in the middle with deleted",
			seg:  makeSegmentHelper1([]int64{17, 23, 37, 42}, []bool{false, true, false, false}),
			val:  30,
			want: 0,
		},
		{
			name: "all deleted",
			seg:  makeSegmentHelper1([]int64{17, 23, 37, 42}, []bool{true, true, true, true}),
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
	seg := makeSegmentHelper1(
		[]int64{17, 23, 23, 23, 37, 42, 49, 51},
		[]bool{false, true, false, true, false, false, false, false},
	)
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
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, seg.nextNonDeletedAfter(tt.idx))
		})
	}
}

func Test_segment_min(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		seg  *segment[int64]
		want int
	}{
		{
			name: "single element",
			seg:  makeSegmentHelper2([]int64{42}),
			want: 0,
		},
		{
			name: "two elements - first is min",
			seg:  makeSegmentHelper2([]int64{17, 42}),
			want: 0,
		},
		{
			name: "two equal elements - should return rightmost (FIFO)",
			seg:  makeSegmentHelper2([]int64{23, 23}),
			want: 1,
		},
		{
			name: "three equal elements - should return rightmost",
			seg:  makeSegmentHelper2([]int64{23, 23, 23}),
			want: 2,
		},
		{
			name: "equal elements with deleted after - should return last non-deleted",
			seg:  makeSegmentHelper1([]int64{23, 23, 23}, []bool{false, false, true}),
			want: 1,
		},
		{
			name: "equal elements with multiple deleted after - should return last non-deleted",
			seg:  makeSegmentHelper1([]int64{23, 23, 23, 23}, []bool{false, false, true, true}),
			want: 1,
		},
		{
			name: "sorted array - minimum is first",
			seg:  makeSegmentHelper2([]int64{17, 23, 37, 42}),
			want: 0,
		},
		{
			name: "sorted array with equal minimums",
			seg:  makeSegmentHelper2([]int64{17, 17, 23, 37, 42}),
			want: 1,
		},
		{
			name: "sorted array with equal minimums and deleted after",
			seg: makeSegmentHelper1(
				[]int64{17, 17, 17, 23, 37, 42},
				[]bool{false, false, true, false, false, false},
			),
			want: 1,
		},
		{
			name: "sorted array with larger elements after equal mins",
			seg:  makeSegmentHelper2([]int64{5, 5, 5, 10, 20, 30}),
			want: 2,
		},
		{
			name: "first element deleted - min is second",
			seg:  makeSegmentHelper1([]int64{17, 23, 37, 42}, []bool{true, false, false, false}),
			want: 1,
		},
		{
			name: "sparse deleted elements",
			seg:  makeSegmentHelper1([]int64{17, 23, 37, 42}, []bool{true, false, true, false}),
			want: 1,
		},
		{
			name: "all equal non-deleted with deleted suffix",
			seg:  makeSegmentHelper1([]int64{10, 10, 10, 10, 10}, []bool{false, false, false, true, true}),
			want: 2,
		},
		{
			name: "complex case - equal mins, then deleted, then larger values",
			seg: makeSegmentHelper1(
				[]int64{5, 5, 5, 5, 10, 15, 20},
				[]bool{false, false, false, true, false, false, false},
			),
			want: 2,
		},
		{
			name: "single non-deleted in middle of deleted",
			seg:  makeSegmentHelper1([]int64{17, 23, 37, 42}, []bool{true, true, false, true}),
			want: 2,
		},
		{
			name: "equal elements at start with larger after",
			seg:  makeSegmentHelper2([]int64{1, 1, 2, 3, 4}),
			want: 1,
		},
		{
			name: "invariant test - deleted only after equal non-deleted",
			seg: makeSegmentHelper1(
				[]int64{10, 10, 10, 10, 20, 30},
				[]bool{false, false, true, true, false, false},
			),
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
				assert.Falsef(t, tt.seg.deleted[got], "min() returned deleted element at index %d", got)
			}
		})
	}
}

func validateSegment[T any](t *testing.T, seg segment[T], cmp CmpFunc[T]) {
	deleted, firstNonDelIdx, lastNonDelIdx := 0, 0, len(seg.elements)-1
	metNonDel := false
	var firstNotDeletedValue T
	for i := range seg.elements {
		if seg.deleted[i] {
			deleted++
			continue
		}
		// If elements are equal, deleted must be after non-deleted;
		if i != 0 && cmp(seg.elements[i-1], seg.elements[i]) == 0 {
			if seg.deleted[i-1] {
				assert.Failf(t, "Order constraint", "at index %d and %d: equal elements %d, but deleted comes before non-deleted", i-1, i, seg.elements[i])
			}
		}
		lastNonDelIdx = i
		if !metNonDel {
			firstNonDelIdx = i
			metNonDel = true
			firstNotDeletedValue = seg.elements[i]
		} else if cmp(seg.elements[i], firstNotDeletedValue) == 0 {
			firstNonDelIdx = i
		}

		if i >= len(seg.elements)-1 || seg.deleted[i+1] {
			continue
		}
		assert.LessOrEqual(t, cmp(seg.elements[i], seg.elements[i+1]), 0)
	}
	assert.Equal(t, deleted, seg.deletedNum())
	assert.Equal(t, firstNonDelIdx, seg.minNonDeletedIndex(cmp))
	assert.Equal(t, lastNonDelIdx, seg.maxNonDeletedIndex())
}

func segmentsEqual[T any](t *testing.T, expected, actual segment[T]) {
	require.Len(t, actual.elements, len(expected.elements))
	require.Equal(t, expected.deletedNum(), actual.deletedNum())
	for i := range expected.elements {
		assert.Equal(t, expected.deleted[i], actual.deleted[i])
		if !expected.deleted[i] {
			assert.Equal(t, expected.elements[i], actual.elements[i])
		}
	}
}
