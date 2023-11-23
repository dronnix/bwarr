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
		require.Len(t, len(bwa.whiteSegments[i].elements), 1<<i)
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
