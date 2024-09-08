package bwarr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateDescIteratorEnd(t *testing.T) {
	t.Parallel()

	// Initialize a BWArr with some segments
	bwa := New(int64Cmp, 0)
	for i := 0; i < 7; i++ {
		bwa.Insert(int64(i))
	}

	// Mark some elements as deleted
	bwa.whiteSegments[2].deleted[0] = true // Delete first element of third segment
	bwa.whiteSegments[2].deleted[3] = true // Delete last element of third segment
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
	for i := 0; i < 7; i++ {
		bwa.Insert(int64(i))
	}

	// Mark some elements as deleted
	bwa.whiteSegments[2].deleted[0] = true // Delete first element of third segment
	bwa.whiteSegments[2].deleted[3] = true // Delete last element of third segment
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
