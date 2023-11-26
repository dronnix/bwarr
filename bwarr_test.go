package bwarr

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBWArrInt64(t *testing.T) {
	t.Parallel()
	testNewBWArr(t, int64Cmp)
}

func TestNewBWArrTestStruct(t *testing.T) {
	t.Parallel()
	testNewBWArr(t, testStructCmp)
}

func TestBlackWhiteArray_Append(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		bwaBefore    *BWArr[int64]
		addedElement int64
		bwaAfter     *BWArr[int64]
	}{
		{
			name:         "add to empty",
			bwaBefore:    New(int64Cmp, 0),
			addedElement: 23,
			bwaAfter:     makeInt64BWAFromWhite([][]int64{{23}, {0, 0}}, 1),
		},
		{
			name:         "add having one element",
			bwaBefore:    makeInt64BWAFromWhite([][]int64{{23}, {0, 0}}, 1),
			addedElement: 42,
			bwaAfter:     makeInt64BWAFromWhite([][]int64{{0}, {23, 42}}, 2),
		},
		{
			name:         "add to full",
			bwaBefore:    makeInt64BWAFromWhite([][]int64{{31}, {23, 42}}, 3),
			addedElement: 37,
			bwaAfter:     makeInt64BWAFromWhite([][]int64{{0}, {0, 0}, {23, 31, 37, 42}}, 4),
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			tt.bwaBefore.Append(tt.addedElement)
			validateBWArr(t, tt.bwaBefore)
			bwaEqual(t, tt.bwaAfter, tt.bwaBefore)
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
			want:      2,
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
			seg1:     segment[int64]{elements: []int64{23, 42}, deleted: []bool{false, false}},
			seg2:     segment[int64]{elements: []int64{17, 37}, deleted: []bool{false, false}},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: make([]bool, 4)},
			expected: segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: make([]bool, 4)},
		},
		{
			name:     "rewind from first",
			seg1:     segment[int64]{elements: []int64{3, 4}, deleted: []bool{false, false}},
			seg2:     segment[int64]{elements: []int64{17, 37}, deleted: []bool{false, false}},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: make([]bool, 4)},
			expected: segment[int64]{elements: []int64{3, 4, 17, 37}, deleted: make([]bool, 4)},
		},
		{
			name:     "two with one deleted element",
			seg1:     segment[int64]{elements: []int64{23, 42}, deleted: []bool{false, false}},
			seg2:     segment[int64]{elements: []int64{17, 37}, deleted: []bool{false, true}, deletedNum: 1},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: make([]bool, 4)},
			expected: segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: []bool{false, false, true, false}, deletedNum: 1},
		},
		{
			name:     "two with two deleted elements",
			seg1:     segment[int64]{elements: []int64{23, 42}, deleted: []bool{true, false}, deletedNum: 1},
			seg2:     segment[int64]{elements: []int64{17, 37}, deleted: []bool{false, true}, deletedNum: 1},
			result:   &segment[int64]{elements: make([]int64, 4), deleted: make([]bool, 4)},
			expected: segment[int64]{elements: []int64{17, 23, 37, 42}, deleted: []bool{false, true, true, false}, deletedNum: 2},
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			mergeSegments(tt.seg1, tt.seg2, int64Cmp, tt.result)
			require.Equal(t, tt.expected, *tt.result)
		})
	}
}

func int64Cmp(a, b int64) int {
	return int(a - b)
}

type testStruct struct {
	I   int
	S   string
	Arr []int
}

func testStructCmp(a, b testStruct) int {
	iCmp := a.I - b.I
	if iCmp != 0 {
		return iCmp
	}

	sCmp := strings.Compare(a.S, b.S)
	if sCmp != 0 {
		return sCmp
	}

	lArrCmp := len(a.Arr) - len(b.Arr)
	if lArrCmp != 0 {
		return lArrCmp
	}
	for i := 0; i < len(a.Arr); i++ {
		arrCmp := a.Arr[i] - b.Arr[i]
		if arrCmp != 0 {
			return arrCmp
		}
	}
	return 0
}

func testNewBWArr[T any](t *testing.T, cmp CmpFunc[T]) {
	tests := []struct {
		name              string
		capacity          int
		wantWhiteSegments int
	}{
		{
			name:              "zero capacity",
			capacity:          0,
			wantWhiteSegments: 2,
		},
		{
			name:              "capacity = 7",
			capacity:          7,
			wantWhiteSegments: 3,
		},
		{
			name:              "capacity = 8",
			capacity:          8,
			wantWhiteSegments: 4,
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			bwa := New(cmp, tt.capacity)
			require.Len(t, bwa.whiteSegments, tt.wantWhiteSegments)
			validateBWArr[T](t, bwa)
		})
	}
}

func validateBWArr[T any](t *testing.T, bwa *BWArr[T]) {
	if len(bwa.whiteSegments) == 0 && len(bwa.blackSegments) == 0 || bwa.total == 0 {
		return
	}
	require.Len(t, bwa.whiteSegments, len(bwa.blackSegments)+1)

	for i := 0; i < len(bwa.whiteSegments); i++ {
		require.Len(t, bwa.whiteSegments[i].elements, 1<<i)
		validateSegment(t, bwa.whiteSegments[i], bwa.cmp)
	}

	for i := 0; i < len(bwa.blackSegments); i++ {
		require.Len(t, bwa.blackSegments[i].elements, 1<<i)
		require.Equal(t, len(bwa.blackSegments[i].elements), len(bwa.blackSegments[i].deleted))
	}
}

func validateSegment[T any](t *testing.T, seg segment[T], cmp CmpFunc[T]) {
	require.Equal(t, len(seg.elements), len(seg.deleted))
	deleted := 0
	for i := 0; i < len(seg.elements); i++ {
		if seg.deleted[i] {
			deleted++
			continue
		}

		if i >= len(seg.elements)-1 || seg.deleted[i+1] {
			continue
		}
		assert.LessOrEqual(t, cmp(seg.elements[i], seg.elements[i+1]), 0)
	}
	assert.Equal(t, deleted, seg.deletedNum)
}

//nolint:exhaustruct
func makeInt64BWAFromWhite(segs [][]int64, total int) *BWArr[int64] {
	bwa := BWArr[int64]{
		whiteSegments: make([]segment[int64], len(segs)),
		blackSegments: make([]segment[int64], len(segs)),
		cmp:           int64Cmp,
		total:         total,
	}
	for i, seg := range segs {
		bwa.whiteSegments[i] = segment[int64]{elements: seg, deleted: make([]bool, len(seg))}
		bwa.blackSegments[i] = segment[int64]{elements: make([]int64, len(seg)), deleted: make([]bool, len(seg))}
	}
	bwa.blackSegments = bwa.blackSegments[:len(bwa.blackSegments)-1]
	return &bwa
}

func bwaEqual[T any](t *testing.T, expected, actual *BWArr[T]) {
	require.GreaterOrEqual(t, len(expected.whiteSegments), len(actual.whiteSegments))
	require.Equal(t, expected.total, actual.total)
	for seg := 0; seg < len(expected.whiteSegments); seg++ {
		if expected.total&(1<<seg) == 0 {
			continue
		}
		segmentsEqual(t, expected.whiteSegments[seg], actual.whiteSegments[seg])
	}
}

func segmentsEqual[T any](t *testing.T, expected, actual segment[T]) {
	require.Equal(t, len(expected.elements), len(expected.deleted))
	require.Equal(t, len(expected.elements), len(actual.elements))
	require.Equal(t, len(expected.deleted), len(actual.deleted))
	for i := range expected.elements {
		assert.Equal(t, expected.deleted[i], actual.deleted[i])
		if !expected.deleted[i] {
			assert.Equal(t, expected.elements[i], actual.elements[i])
		}
	}
}
