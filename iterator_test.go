package bwarr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateDescIteratorEnd(t *testing.T) {
	t.Parallel()

	// Initialize a BWArr with some segments
	bwa := New(int64Cmp, 0)
	for i := range 7 {
		bwa.Insert(int64(i))
	}

	// Mark some elements as deleted
	bwa.whiteSegments[2].deleted.Set(0) // Delete first element of third segment
	bwa.whiteSegments[2].deleted.Set(3) // Delete last element of third segment
	bwa.whiteSegments[2].deletedNum += 2

	iter := createDescIteratorEnd(bwa)

	expectedIndices := []int{0, 1, 2}
	expectedLengths := []int{1, 2, 4}
	expectedEnds := []int{0, 0, 1}

	for i, si := range iter.segIters {
		assert.Equal(t, expectedIndices[i], si.index)
		assert.Len(t, si.seg.elements, expectedLengths[i])
		assert.Equal(t, expectedEnds[i], si.end)
	}
}

func TestCreateAscIteratorBegin(t *testing.T) {
	t.Parallel()

	// Initialize a BWArr with some segments
	bwa := New(int64Cmp, 0)
	for i := range 7 {
		bwa.Insert(int64(i))
	}

	// Mark some elements as deleted
	bwa.whiteSegments[2].deleted.Set(0) // Delete first element of third segment
	bwa.whiteSegments[2].deleted.Set(3) // Delete last element of third segment
	bwa.whiteSegments[2].deletedNum += 2

	iter := createAscIteratorBegin(bwa)

	expectedIndices := []int{1, 0, 0}
	expectedLengths := []int{4, 2, 1}
	expectedEnds := []int{2, 1, 0}

	for i, si := range iter.segIters {
		assert.Equal(t, expectedIndices[i], si.index)
		assert.Len(t, si.seg.elements, expectedLengths[i])
		assert.Equal(t, expectedEnds[i], si.end)
	}
}

// The skip guards handle segmentIterator states outside the constructors' contract
// (end pointing at a deleted element, fully deleted words). Locked in here so the
// slow paths stay panic-free for any segment state.
//
//nolint:exhaustruct
func Test_segmentIteratorSkipGuards(t *testing.T) {
	t.Parallel()

	t.Run("nextSlow: in-word skip lands beyond end", func(t *testing.T) {
		t.Parallel()
		seg := makeSegment[int64](3) // 8 elements; 1..7 deleted, phantom zeros above.
		for i := 1; i < 8; i++ {
			seg.deleted.Set(i)
		}
		it := segmentIterator[int64]{seg: seg, end: 7, word: seg.deleted.layers[0][0]}
		assert.False(t, it.nextSlow(1))
	})

	t.Run("nextSlow: layered walk finds nothing", func(t *testing.T) {
		t.Parallel()
		seg := makeSegment[int64](6) // One fully set word.
		for i := range 64 {
			seg.deleted.Set(i)
		}
		it := segmentIterator[int64]{seg: seg, end: 63, word: seg.deleted.layers[0][0]}
		assert.False(t, it.nextSlow(1))
	})

	t.Run("prevSlow: in-word skip lands before end", func(t *testing.T) {
		t.Parallel()
		seg := makeSegment[int64](3)
		seg.deleted.Set(2)
		it := segmentIterator[int64]{seg: seg, end: 2, word: seg.deleted.layers[0][0]}
		assert.False(t, it.prevSlow(2))
	})

	t.Run("prevSlow: layered walk finds nothing", func(t *testing.T) {
		t.Parallel()
		seg := makeSegment[int64](6)
		for i := range 64 {
			seg.deleted.Set(i)
		}
		it := segmentIterator[int64]{seg: seg, end: 0, word: seg.deleted.layers[0][0]}
		assert.False(t, it.prevSlow(62))
	})
}
