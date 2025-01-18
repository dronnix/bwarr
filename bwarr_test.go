package bwarr

import (
	"math/rand"
	"slices"
	"sort"
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
			t.Parallel()
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
			t.Parallel()
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
			t.Parallel()
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
			t.Parallel()
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
			t.Parallel()
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
			t.Parallel()
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
			t.Parallel()
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

	for i := range elems {
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
			t.Parallel()
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

func TestBWArr_DeleteFromEmpty(t *testing.T) {
	t.Parallel()
	bwa := New(int64Cmp, 0)

	elem, found := bwa.DeleteMin()
	assert.False(t, found)
	assert.Equal(t, int64(0), elem)

	elem, found = bwa.DeleteMax()
	assert.False(t, found)
	assert.Equal(t, int64(0), elem)
}

func TestBWArr_RandomDelete(t *testing.T) {
	t.Parallel()
	rand.Seed(42) //nolint:staticcheck
	const elements = 63
	bwa := New(int64Cmp, 0)
	toDel := make([]int64, elements)
	for i := range elements {
		toDel[i] = int64(i)
	}
	rand.Shuffle(len(toDel), func(i, j int) { toDel[i], toDel[j] = toDel[j], toDel[i] })

	for i := range toDel {
		bwa.Insert(toDel[i])
	}
	rand.Shuffle(len(toDel), func(i, j int) { toDel[i], toDel[j] = toDel[j], toDel[i] })

	for i := range toDel {
		if elem, found := bwa.Delete(toDel[i]); !found || elem != toDel[i] {
			t.Logf("failed to delete %d on %d iteration", toDel[i], i)
			t.Fail()
		}
	}
}

func TestBWArr_Len(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		add  []int64
		del  []int64
		want int
	}{
		{
			name: "add one to empty",
			add:  []int64{23},
			del:  nil,
			want: 1,
		},
		{
			name: "add one, del one",
			add:  []int64{23},
			del:  []int64{23},
			want: 0,
		},
		{
			name: "add four, del one",
			add:  []int64{23, 42, 37, 17},
			del:  []int64{42},
			want: 3,
		},
		{
			name: "add four, del two",
			add:  []int64{23, 42, 37, 17},
			del:  []int64{42, 23},
			want: 2,
		},
		{
			name: "add four, del three",
			add:  []int64{23, 42, 37, 17},
			del:  []int64{42, 23, 17},
			want: 1,
		},
		{
			name: "add four, del all",
			add:  []int64{23, 42, 37, 17},
			del:  []int64{42, 23, 17, 37},
			want: 0,
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bwa := New(int64Cmp, 0)
			for _, elem := range tt.add {
				bwa.Insert(elem)
			}
			for _, elem := range tt.del {
				bwa.Delete(elem)
			}
			assert.Equal(t, tt.want, bwa.Len())
		})
	}
}

func TestBWArr_Clear(t *testing.T) {
	t.Parallel()
	bwa := New(int64Cmp, 0)
	for i := range 15 {
		bwa.Insert(int64(i))
	}
	validateBWArr(t, bwa)

	bwa.Clear(true)
	validateBWArr(t, bwa)
	assert.Equal(t, 0, bwa.Len())
	assert.Equal(t, 0, bwa.total)
	assert.Len(t, bwa.whiteSegments, 2)

	for i := range 15 {
		bwa.Insert(int64(i))
	}
	validateBWArr(t, bwa)
}

func TestBWArr_Clone(t *testing.T) {
	t.Parallel()
	bwa := New(int64Cmp, 0)
	for i := range 15 {
		bwa.Insert(int64(i))
	}
	bwa.Delete(3)
	bwa.Delete(11)
	validateBWArr(t, bwa)

	newBwa := bwa.Clone()
	validateBWArr(t, newBwa)
	bwaEqual(t, bwa, newBwa)

	newBwa.Delete(5)
	newBwa.Delete(6)
	newBwa.Delete(7)
	validateBWArr(t, newBwa)

	for i := range 7 {
		newBwa.Insert(int64(i))
	}
	validateBWArr(t, newBwa)
}

func TestBWArr_Ascend(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name    string
		initSeq []int64
	}
	tests := []testCase{
		{name: "empty", initSeq: []int64{}},
		{name: "one", initSeq: []int64{1}},
		{name: "two", initSeq: []int64{11, 7}},
		{name: "three", initSeq: []int64{11, 7, 13}},
		{name: "four", initSeq: []int64{11, 7, 17, 13}},
		{name: "five", initSeq: []int64{11, 17, 13, 19, 7}},
		{name: "six", initSeq: []int64{11, 17, 13, 19, 7, 4}},
		{name: "seven", initSeq: []int64{23, 7, 17, 13, 19, 7, 4}},
		{name: "eight", initSeq: []int64{23, 7, 42, 13, 19, 7, 4, 5}},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bwa := New(int64Cmp, len(tt.initSeq))
			for _, e := range tt.initSeq {
				bwa.Insert(e)
			}
			expected := make([]int64, len(tt.initSeq))
			copy(expected, tt.initSeq)
			sort.Slice(expected, func(i, j int) bool { return expected[i] < expected[j] })

			got := make([]int64, 0, len(tt.initSeq))
			iter := func(e int64) bool {
				got = append(got, e)
				return true
			}
			bwa.Ascend(iter)
			assert.Equal(t, expected, got)
		})
	}
}

func TestBWArr_AscendRandom(t *testing.T) {
	t.Parallel()
	rand.Seed(2342) //nolint:staticcheck
	const elements = 1023
	bwa := New(int64Cmp, elements)
	for range elements {
		bwa.Insert(int64(rand.Intn(100)))
	}

	prev := int64(0)
	iter := func(e int64) bool {
		assert.GreaterOrEqual(t, e, prev)
		prev = e
		return true
	}
	bwa.Ascend(iter)
}

func TestBWArr_AscendWithDeleted(t *testing.T) {
	t.Parallel()
	const elemsNum, toDel = 1023, 241
	elems := make([]int64, elemsNum)
	for i := range elems {
		elems[i] = int64(i)
	}
	rand.Shuffle(len(elems), func(i, j int) { elems[i], elems[j] = elems[j], elems[i] })

	bwa := New(int64Cmp, elemsNum)
	for _, v := range elems {
		bwa.Insert(v)
	}
	rand.Shuffle(len(elems), func(i, j int) { elems[i], elems[j] = elems[j], elems[i] })
	for i := range toDel {
		bwa.Delete(elems[i])
	}

	iter := func(e int64) bool {
		assert.GreaterOrEqual(t, slices.Index(elems[toDel:], e), 0)
		return true
	}
	bwa.Ascend(iter)
}

func TestBWArr_AscendStability(t *testing.T) {
	t.Parallel()
	const elemsNum = 1023

	bwa := New(stabValCmp, elemsNum)
	for i := range elemsNum {
		bwa.Insert(stabVal{val: rand.Intn(7), seq: i + 1})
	}

	seqs := make(map[int]int, elemsNum)
	iter := func(e stabVal) bool {
		ps := seqs[e.val]
		assert.Greater(t, e.seq, ps)
		return true
	}
	bwa.Ascend(iter)
}

func TestBWArr_AscendGreaterOrEqual(t *testing.T) {
	t.Parallel()
	const elemsNum = 1023
	bwa := New(int64Cmp, elemsNum)
	for i := range elemsNum {
		bwa.Insert(int64(i))
	}

	for i := range elemsNum {
		if i%2 != 0 {
			bwa.Delete(int64(i))
		}
	}

	const pivot = 780
	expected := int64(pivot)
	iter := func(e int64) bool {
		assert.Equal(t, expected, e)
		expected += 2
		return true
	}
	bwa.AscendGreaterOrEqual(pivot, iter)
	assert.Equal(t, expected, int64(elemsNum+1))
}

func TestBWArr_AscendLessThan(t *testing.T) {
	t.Parallel()
	const elemsNum = 1023
	const pivot = 780
	bwa := New(int64Cmp, elemsNum)
	for i := range elemsNum {
		bwa.Insert(int64(i))
	}

	for i := range elemsNum {
		if i%2 != 0 {
			bwa.Delete(int64(i))
		}
	}

	expected := int64(0)
	iter := func(e int64) bool {
		assert.Equal(t, expected, e)
		expected += 2
		return true
	}
	bwa.AscendLessThan(pivot, iter)
	assert.Equal(t, expected, int64(pivot))
}

func TestBWArr_AscendRange(t *testing.T) {
	t.Parallel()
	const elemsNum = 1023
	bwa := New(int64Cmp, elemsNum)
	for i := range elemsNum {
		bwa.Insert(int64(i))
	}
	for i := range elemsNum {
		if i%2 == 0 {
			bwa.Delete(int64(i))
		}
	}

	const from, to = int64(233), int64(781)
	expected := from
	iter := func(e int64) bool {
		require.Equal(t, expected, e)
		expected += 2
		return true
	}
	bwa.AscendRange(from, to, iter)
	assert.Equal(t, expected, to)
}

func TestBWArr_AscendRangeOutOfBounds(t *testing.T) {
	t.Parallel()
	const elemsNum = 15
	bwa := New(int64Cmp, elemsNum)
	for i := range elemsNum {
		bwa.Insert(int64(i))
	}

	const from, to = int64(17), int64(23)
	iter := func(_ int64) bool {
		t.Fail()
		return true
	}
	bwa.AscendRange(from, to, iter)
}

func TestBWArr_Descend(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		initSeq []int64
	}{
		{name: "empty", initSeq: []int64{}},
		{name: "one", initSeq: []int64{1}},
		{name: "two", initSeq: []int64{11, 7}},
		{name: "three", initSeq: []int64{11, 7, 13}},
		{name: "four", initSeq: []int64{11, 7, 13, 5}},
		{name: "five", initSeq: []int64{11, 7, 13, 5, 9}},
		{name: "six", initSeq: []int64{11, 7, 13, 5, 9, 3}},
		{name: "seven", initSeq: []int64{24, 42, 23, 27, 23, 7, 61}},
		{name: "eight", initSeq: []int64{24, 42, 23, 27, 23, 7, 61, 15}},
		{name: "nine", initSeq: []int64{24, 42, 23, 27, 23, 7, 61, 15, 19}},
		{name: "ten", initSeq: []int64{24, 42, 23, 27, 23, 7, 61, 15, 19, 31}},
		{name: "eleven", initSeq: []int64{24, 42, 23, 27, 23, 7, 61, 15, 19, 31, 29}},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bwa := New(int64Cmp, len(tt.initSeq))
			for _, e := range tt.initSeq {
				bwa.Insert(e)
			}
			expected := make([]int64, len(tt.initSeq))
			copy(expected, tt.initSeq)
			sort.Slice(expected, func(i, j int) bool { return expected[i] > expected[j] })

			got := make([]int64, 0, len(tt.initSeq))
			iter := func(e int64) bool {
				got = append(got, e)
				return true
			}
			bwa.Descend(iter)
			assert.Equal(t, expected, got)
		})
	}
}

func TestBWArr_DescendRandom(t *testing.T) {
	t.Parallel()
	rand.Seed(2342) //nolint:staticcheck
	const elements = 1023
	bwa := New(int64Cmp, elements)
	for range elements {
		bwa.Insert(int64(rand.Intn(100)))
	}

	prev := int64(100)
	iter := func(e int64) bool {
		assert.LessOrEqual(t, e, prev)
		prev = e
		return true
	}
	bwa.Descend(iter)
}

func TestBWArr_DescendWithDeleted(t *testing.T) {
	t.Parallel()
	const elemsNum, toDel = 1023, 241
	elems := make([]int64, elemsNum)
	for i := range elems {
		elems[i] = int64(i)
	}
	rand.Shuffle(len(elems), func(i, j int) { elems[i], elems[j] = elems[j], elems[i] })

	bwa := New(int64Cmp, elemsNum)
	for _, v := range elems {
		bwa.Insert(v)
	}
	rand.Shuffle(len(elems), func(i, j int) { elems[i], elems[j] = elems[j], elems[i] })
	for i := range toDel {
		bwa.Delete(elems[i])
	}

	iter := func(e int64) bool {
		assert.GreaterOrEqual(t, slices.Index(elems[toDel:], e), 0)
		return true
	}
	bwa.Descend(iter)
}

func TestBWArr_DescendStability(t *testing.T) {
	t.Parallel()
	const elemsNum = 1023

	bwa := New(stabValCmp, elemsNum)
	for i := range elemsNum {
		bwa.Insert(stabVal{val: rand.Intn(7), seq: i + 1})
	}

	seqs := make(map[int]int, elemsNum)
	iter := func(e stabVal) bool {
		ps := seqs[e.val]
		assert.Greater(t, e.seq, ps)
		return true
	}
	bwa.Descend(iter)
}

func TestBWArr_DescendGreaterOrEqual(t *testing.T) {
	t.Parallel()
	const elemsNum = 1023
	bwa := New(int64Cmp, elemsNum)
	for i := 8; i < elemsNum; i++ {
		bwa.Insert(int64(i))
	}

	const pivot = 622
	expected := int64(elemsNum - 1)
	iter := func(e int64) bool {
		assert.Equal(t, expected, e)
		expected--
		return true
	}
	bwa.DescendGreaterOrEqual(pivot, iter)
	assert.Equal(t, expected, int64(pivot-1))
}

func TestBWArr_DescendLessThan(t *testing.T) {
	t.Parallel()
	const elemsNum = 1023
	bwa := New(int64Cmp, elemsNum)
	for i := range elemsNum - 8 {
		bwa.Insert(int64(i))
	}

	const pivot = 822
	expected := int64(pivot - 1)
	iter := func(e int64) bool {
		assert.Equal(t, expected, e)
		expected--
		return true
	}
	bwa.DescendLessThan(pivot, iter)
	assert.Equal(t, expected, int64(-1))
}

func TestBWArr_DescendRange(t *testing.T) {
	t.Parallel()
	const elemsNum = 1023
	bwa := New(int64Cmp, elemsNum)
	for i := range elemsNum {
		bwa.Insert(int64(i))
	}
	for i := range elemsNum {
		if i%2 == 0 {
			bwa.Delete(int64(i))
		}
	}

	const from, to = int64(23), int64(977)
	expected := to - 2
	iter := func(e int64) bool {
		require.Equal(t, expected, e)
		expected -= 2
		return true
	}
	bwa.DescendRange(from, to, iter)
	assert.Equal(t, expected, from-2)
}

func TestBWArr_DescendRangeOutOfBounds(t *testing.T) {
	t.Parallel()
	const elemsNum = 15
	bwa := New(int64Cmp, elemsNum)
	for i := range elemsNum {
		bwa.Insert(int64(i))
	}

	const from, to = int64(17), int64(23)
	iter := func(_ int64) bool {
		t.Fail()
		return true
	}
	bwa.DescendRange(from, to, iter)
}

func TestBWArr_AscIteratorsShouldStop(t *testing.T) {
	t.Parallel()
	const elemsNum = 15
	bwa := New(int64Cmp, elemsNum)
	for i := range elemsNum {
		bwa.Insert(int64(i))
	}
	iter := func(e int64) bool {
		require.Equal(t, int64(0), e)
		return false
	}
	bwa.Ascend(iter)
	bwa.AscendGreaterOrEqual(0, iter)
	bwa.AscendLessThan(7, iter)
	bwa.AscendRange(0, 7, iter)
}

func TestBWArr_DescIteratorsShouldStop(t *testing.T) {
	t.Parallel()
	const elemsNum = 15
	bwa := New(int64Cmp, elemsNum)
	for i := range elemsNum {
		bwa.Insert(int64(i))
	}
	iter := func(e int64) bool {
		require.Equal(t, int64(elemsNum-1), e)
		return false
	}
	bwa.Descend(iter)
	bwa.DescendGreaterOrEqual(5, iter)
	bwa.DescendLessThan(elemsNum, iter)
	bwa.DescendRange(7, elemsNum, iter)
}

func TestBWArr_Compact(t *testing.T) {
	t.Parallel()

	bwa := New(int64Cmp, 0)
	bwa.Compact()
	validateBWArr(t, bwa)

	const elemsNum = 16
	toInsert := make([]int64, elemsNum)
	for i := range elemsNum {
		toInsert[i] = int64(i)
	}
	rand.Shuffle(len(toInsert), func(i, j int) { toInsert[i], toInsert[j] = toInsert[j], toInsert[i] })
	for i := range elemsNum {
		bwa.Insert(toInsert[i])
		validateBWArr(t, bwa)
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
	for i := range a.Arr {
		arrCmp := a.Arr[i] - b.Arr[i]
		if arrCmp != 0 {
			return arrCmp
		}
	}
	return 0
}

type stabVal struct { // For checking stability
	val int
	seq int
}

func stabValCmp(a, b stabVal) int {
	return a.val - b.val
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
			t.Parallel()
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

	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) == 0 {
			continue
		}
		require.Len(t, bwa.whiteSegments[i].elements, 1<<i)
		validateSegment(t, bwa.whiteSegments[i], bwa.cmp)
	}

	for i := range bwa.blackSegments {
		require.Len(t, bwa.blackSegments[i].elements, len(bwa.blackSegments[i].deleted))
	}
}

//nolint:exhaustruct
func makeInt64BWAFromWhite(segs [][]int64, total int) *BWArr[int64] {
	bwa := BWArr[int64]{
		whiteSegments: make([]segment[int64], len(segs)),
		cmp:           int64Cmp,
		total:         total,
	}
	for i, seg := range segs {
		l := len(seg)
		bwa.whiteSegments[i] = segment[int64]{elements: seg, deleted: make([]bool, l), maxNonDeletedIdx: l - 1}
	}
	return &bwa
}

func bwaEqual[T any](t *testing.T, expected, actual *BWArr[T]) {
	require.GreaterOrEqual(t, len(expected.whiteSegments), len(actual.whiteSegments))
	require.Equal(t, expected.total, actual.total)
	for seg := range expected.whiteSegments {
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
