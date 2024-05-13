package bwarr

import (
	"math/rand"
	"slices"
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

func TestBWArr_Insert(t *testing.T) {
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
			tt.bwaBefore.Insert(tt.addedElement)
			validateBWArr(t, tt.bwaBefore)
			bwaEqual(t, tt.bwaAfter, tt.bwaBefore)
		})
	}
}

//nolint:exhaustruct
func TestBWArr_ReplaceOrInsert(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		toInsertBefore    []testStruct
		toReplaceOrInsert testStruct
		expectedFound     bool
		expectedOld       testStruct
	}{
		{
			name:              "empty",
			toInsertBefore:    []testStruct{},
			toReplaceOrInsert: testStruct{I: 23},
			expectedFound:     false,
		},
		{
			name:              "no match",
			toInsertBefore:    []testStruct{{I: 23}, {I: 42}},
			toReplaceOrInsert: testStruct{I: 37},
			expectedFound:     false,
		},
		{
			name:              "one match",
			toInsertBefore:    []testStruct{{I: 23}, {I: 42, Lbl: "Foo"}, {I: 37}},
			toReplaceOrInsert: testStruct{I: 42, Lbl: "Bar"},
			expectedFound:     true,
			expectedOld:       testStruct{I: 42, Lbl: "Foo"},
		},
		{
			name:              "two matches",
			toInsertBefore:    []testStruct{{I: 23}, {I: 42, Lbl: "Foo"}, {I: 42, Lbl: "Bar"}, {I: 37}},
			toReplaceOrInsert: testStruct{I: 42, Lbl: "Baz"},
			expectedFound:     true,
			expectedOld:       testStruct{I: 42, Lbl: "Foo"},
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			bwa := New(testStructCmp, 0)
			for _, elem := range tt.toInsertBefore {
				bwa.Insert(elem)
			}
			old, found := bwa.ReplaceOrInsert(tt.toReplaceOrInsert)
			validateBWArr(t, bwa)
			assert.Equal(t, tt.expectedFound, found)
			if tt.expectedFound {
				assert.Equal(t, tt.expectedOld, old)
			} else {
				assert.Equal(t, testStruct{}, old)
			}
			assert.True(t, bwa.Has(tt.toReplaceOrInsert))
		})
	}
}

func TestBWArr_HasAndGet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		elemsToAdd      []int64
		elementToSearch int64
		want            bool
	}{
		{
			name:            "empty",
			elemsToAdd:      []int64{},
			elementToSearch: 23,
			want:            false,
		},
		{
			name:            "one match",
			elemsToAdd:      []int64{23},
			elementToSearch: 23,
			want:            true,
		},
		{
			name:            "match from two",
			elemsToAdd:      []int64{23, 42},
			elementToSearch: 42,
			want:            true,
		},
		{
			name:            "match from three",
			elemsToAdd:      []int64{23, 42, 37},
			elementToSearch: 37,
			want:            true,
		},
		{
			name:            "match from seven",
			elemsToAdd:      []int64{23, 42, 37, 17, 31, 29, 41},
			elementToSearch: 37,
			want:            true,
		},
		{
			name:            "not match",
			elemsToAdd:      []int64{23, 42, 37, 17, 31, 29, 41},
			elementToSearch: 13,
			want:            false,
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			bwa := New(int64Cmp, 0)
			for _, elem := range tt.elemsToAdd {
				bwa.Insert(elem)
			}
			assert.Equalf(t, tt.want, bwa.Has(tt.elementToSearch), "Contains(%v)", tt.elementToSearch)
			elem, found := bwa.Get(tt.elementToSearch)
			assert.Equal(t, tt.want, found)
			if tt.want {
				assert.Equal(t, tt.elementToSearch, elem)
			}
		})
	}
}

func TestBWArr_Min(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		elemsToAdd []int64
		want       int64
		found      bool
	}{
		{
			name:       "empty",
			elemsToAdd: []int64{},
			want:       0,
			found:      false,
		},
		{
			name:       "one",
			elemsToAdd: []int64{23},
			want:       23,
			found:      true,
		},
		{
			name:       "two",
			elemsToAdd: []int64{42, 23},
			want:       23,
			found:      true,
		},
		{
			name:       "7",
			elemsToAdd: []int64{24, 42, 23, 27, 23, 7, 61},
			want:       7,
			found:      true,
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			bwa := New(int64Cmp, 0)
			for _, elem := range tt.elemsToAdd {
				bwa.Insert(elem)
			}
			validateBWArr(t, bwa)
			elem, found := bwa.Min()
			validateBWArr(t, bwa)
			assert.Equal(t, tt.found, found)
			assert.Equal(t, tt.want, elem)
		})
	}
}

func TestBWArr_Max(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		elemsToAdd []int64
		want       int64
		found      bool
	}{
		{
			name:       "empty",
			elemsToAdd: []int64{},
			want:       0,
			found:      false,
		},
		{
			name:       "one",
			elemsToAdd: []int64{23},
			want:       23,
			found:      true,
		},
		{
			name:       "two",
			elemsToAdd: []int64{42, 23},
			want:       42,
			found:      true,
		},
		{
			name:       "61",
			elemsToAdd: []int64{24, 42, 23, 27, 23, 7, 61},
			want:       61,
			found:      true,
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			bwa := New(int64Cmp, 0)
			for _, elem := range tt.elemsToAdd {
				bwa.Insert(elem)
			}
			validateBWArr(t, bwa)
			elem, found := bwa.Max()
			validateBWArr(t, bwa)
			assert.Equal(t, tt.found, found)
			assert.Equal(t, tt.want, elem)
		})
	}
}

func TestBWArr_MinStability(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		elemsToAdd []int64
		segment    int
		index      int
	}{
		{
			name:       "two",
			elemsToAdd: []int64{23, 42, 23, 27, 23, 29, 61},
			segment:    2,
			index:      0,
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			bwa := New(int64Cmp, 0)
			for _, elem := range tt.elemsToAdd {
				bwa.Insert(elem)
			}
			validateBWArr(t, bwa)
			seg, ind := bwa.min()
			assert.Equal(t, tt.segment, seg)
			assert.Equal(t, tt.index, ind)
			validateBWArr(t, bwa)
		})
	}
}

func TestBWArr_MaxStability(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		elemsToAdd []int64
		segment    int
		index      int
	}{
		{
			name:       "two",
			elemsToAdd: []int64{61, 42, 23, 27, 61, 29, 61},
			segment:    2,
			index:      3,
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			bwa := New(int64Cmp, 0)
			for _, elem := range tt.elemsToAdd {
				bwa.Insert(elem)
			}
			validateBWArr(t, bwa)
			seg, ind := bwa.max()
			assert.Equal(t, tt.segment, seg)
			assert.Equal(t, tt.index, ind)
			validateBWArr(t, bwa)
		})
	}
}

func TestBWArr_DeleteMin(t *testing.T) {
	t.Parallel()
	elems := []int64{87, 42, 23, 27, 23, 29, 61, 17, 51, 50, 11, 90}
	bwa := New(int64Cmp, len(elems))
	for _, elem := range elems {
		bwa.Insert(elem)
	}
	validateBWArr(t, bwa)
	slices.Sort(elems)

	for i := range elems {
		elem, found := bwa.DeleteMin()
		validateBWArr(t, bwa)
		assert.True(t, found)
		assert.Equal(t, elems[i], elem, "DeleteMin() on %d iteration", i)
	}
}

func TestBWArr_DeleteMax(t *testing.T) {
	t.Parallel()
	elems := []int64{87, 42, 23, 27, 23, 29, 61, 17, 51, 50, 11, 90}
	bwa := New(int64Cmp, len(elems))
	for _, elem := range elems {
		bwa.Insert(elem)
	}
	validateBWArr(t, bwa)
	slices.SortFunc(elems, func(a, b int64) int { return int(b - a) })

	for i := 0; i < len(elems); i++ {
		elem, found := bwa.DeleteMax()
		validateBWArr(t, bwa)
		assert.True(t, found)
		assert.Equal(t, elems[i], elem, "DeleteMax() on %d iteration", i)
	}
}

func TestBWArr_Delete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		before          *BWArr[int64]
		elementToDelete int64
		after           *BWArr[int64]
		expected        bool
	}{
		{
			name:            "not found",
			before:          makeInt64BWAFromWhite([][]int64{{23}}, 1),
			elementToDelete: 42,
			after:           makeInt64BWAFromWhite([][]int64{{23}}, 1),
			expected:        false,
		},
		{
			name:            "remove from first segment",
			before:          makeInt64BWAFromWhite([][]int64{{17}, {23, 42}}, 3),
			elementToDelete: 17,
			after:           makeInt64BWAFromWhite([][]int64{{0}, {23, 42}}, 2),
			expected:        true,
		},
		{
			name:            "remove from second segment with demote to first",
			before:          makeInt64BWAFromWhite([][]int64{{0}, {23, 42}}, 2),
			elementToDelete: 23,
			after:           makeInt64BWAFromWhite([][]int64{{42}, {0, 0}}, 1),
			expected:        true,
		},
		{
			name:            "remove from second segment with merge",
			before:          makeInt64BWAFromWhite([][]int64{{17}, {23, 42}}, 3),
			elementToDelete: 23,
			after:           makeInt64BWAFromWhite([][]int64{{0}, {17, 42}}, 2),
			expected:        true,
		},
		{
			name:            "remove from third without demote",
			before:          makeInt64BWAFromWhite([][]int64{{0}, {0, 0}, {17, 23, 37, 42}}, 4),
			elementToDelete: 23,
			after:           markDel(makeInt64BWAFromWhite([][]int64{{0}, {0, 0}, {17, 23, 37, 42}}, 4), bwaIdx{2, 1}),
			expected:        true,
		},
		{
			name:            "remove from third with demote to second",
			before:          markDel(makeInt64BWAFromWhite([][]int64{{1}, {0, 0}, {17, 23, 37, 42}}, 5), bwaIdx{2, 2}),
			elementToDelete: 23,
			after:           makeInt64BWAFromWhite([][]int64{{1}, {17, 42}, {0, 0, 0, 0}}, 3),
			expected:        true,
		},
		{
			name:            "remove from third with merge with second",
			before:          markDel(makeInt64BWAFromWhite([][]int64{{0}, {19, 41}, {17, 23, 37, 42}}, 6), bwaIdx{2, 2}),
			elementToDelete: 23,
			after:           makeInt64BWAFromWhite([][]int64{{0}, {0, 0}, {17, 19, 41, 42}}, 4),
			expected:        true,
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			got, found := tt.before.Delete(tt.elementToDelete)
			assert.Equal(t, tt.expected, found)
			if found {
				assert.Equal(t, tt.elementToDelete, got)
			}
			validateBWArr(t, tt.before)
			bwaEqual(t, tt.after, tt.before)
		})
	}
}

func TestBWArr_RandomDelete(t *testing.T) {
	t.Parallel()
	rand.Seed(42) //nolint:staticcheck
	const elements = 63
	bwa := New(int64Cmp, 0)
	toDel := make([]int64, elements)
	for i := 0; i < elements; i++ {
		toDel[i] = int64(i)
	}
	rand.Shuffle(len(toDel), func(i, j int) { toDel[i], toDel[j] = toDel[j], toDel[i] })

	for i := range toDel {
		bwa.Insert(toDel[i])
	}
	rand.Shuffle(len(toDel), func(i, j int) { toDel[i], toDel[j] = toDel[j], toDel[i] })

	for i := 0; i < len(toDel); i++ {
		if elem, found := bwa.Delete(toDel[i]); !found || elem != toDel[i] {
			t.Logf("failed to delete %d on %d iteration", toDel[i], i)
			t.Fail()
		}
	}
}

func int64Cmp(a, b int64) int {
	return int(a - b)
}

type testStruct struct {
	I   int
	S   string
	Arr []int
	Lbl string // Non-comparable label to distinguish elements
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

//nolint:exhaustruct
func makeInt64BWAFromWhite(segs [][]int64, total int) *BWArr[int64] {
	bwa := BWArr[int64]{
		whiteSegments: make([]segment[int64], len(segs)),
		blackSegments: make([]segment[int64], len(segs)),
		cmp:           int64Cmp,
		total:         total,
	}
	for i, seg := range segs {
		l := len(seg)
		bwa.whiteSegments[i] = segment[int64]{elements: seg, deleted: make([]bool, l), maxNonDeletedIdx: l - 1}
		bwa.blackSegments[i] = segment[int64]{elements: make([]int64, l), deleted: make([]bool, l)}
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

type bwaIdx struct {
	segNum int
	idx    int
}

func markDel[T any](bwa *BWArr[T], toDel ...bwaIdx) *BWArr[T] {
	for i := range toDel {
		bwa.whiteSegments[toDel[i].segNum].deleted[toDel[i].idx] = true
		bwa.whiteSegments[toDel[i].segNum].deletedNum++
	}
	return bwa
}
