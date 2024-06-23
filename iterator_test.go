package bwarr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

//nolint:exhaustruct
func Test_createIterator(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		bwa  *BWArr[int64]
		want iterator[int64]
	}
	tests := []testCase{
		{
			name: "empty",
			bwa:  New(int64Cmp, 0),
			want: iterator[int64]{segIters: make([]segmentIterator[int64], 0), curValPtrs: make([]*int64, 0)},
		},
		{
			name: "one element",
			bwa:  makeInt64BWAFromWhite([][]int64{{23}}, 1),
			want: iterator[int64]{
				segIters:   []segmentIterator[int64]{{seg: segment[int64]{elements: []int64{23}, deleted: []bool{false}}, index: 0}},
				curValPtrs: []*int64{&[]int64{23}[0]},
			},
		},
		{
			name: "two elements",
			bwa:  makeInt64BWAFromWhite([][]int64{{17}, {23, 42}}, 3),
			want: iterator[int64]{
				segIters: []segmentIterator[int64]{
					{seg: segment[int64]{elements: []int64{17}, deleted: []bool{false}}, index: 0},
					{seg: segment[int64]{elements: []int64{23, 42}, deleted: []bool{false, false}, maxNonDeletedIdx: 1}, index: 0},
				},
				curValPtrs: []*int64{&[]int64{17}[0], &[]int64{23, 42}[0]},
			},
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			iter := createIterator(tt.bwa)
			assert.Equal(t, tt.want, iter)
		})
	}
}
