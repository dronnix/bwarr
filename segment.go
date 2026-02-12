package bwarr

import (
	"math"
	"math/bits"
)

type segment[T any] struct {
	elements         []T    // Stores user's data.
	deleted          []bool // Stores whether i-th element is deleted.
	deletedNum       int    // Number of deleted elements in the segment.
	minNonDeletedIdx int    // Index of the first non-deleted element in the segment.
	maxNonDeletedIdx int    // Index of the last non-deleted element in the segment.
}

func createSegments[T any](fromRank, toRank int) []segment[T] {
	segments := make([]segment[T], toRank-fromRank)
	for i := fromRank; i < toRank; i++ {
		segments[i-fromRank] = makeSegment[T](i)
	}
	return segments
}

func makeSegment[T any](rank int) segment[T] {
	l := 1 << rank
	return segment[T]{
		elements:         make([]T, l),
		deleted:          make([]bool, l),
		deletedNum:       0,
		minNonDeletedIdx: 0,
		maxNonDeletedIdx: l - 1,
	}
}

// Merge lowSeg and highSeg into highSeg using highSeg free space at the beginning.
func mergeSegments[T any](lowSeg, highSeg *segment[T], cmp CmpFunc[T], highSegReadIdx int) {
	lowSegReadIdx := 0
	lowSegEnd := len(lowSeg.elements)
	heighSegWriteIxd := highSegReadIdx - lowSegEnd
	heighSegEnd := highSegReadIdx + lowSegEnd

	for highSegReadIdx < heighSegEnd && lowSegReadIdx < lowSegEnd {
		cmpResult := cmp(highSeg.elements[highSegReadIdx], lowSeg.elements[lowSegReadIdx])
		if (cmpResult < 0) || (cmpResult == 0 && !highSeg.deleted[highSegReadIdx]) {
			highSeg.elements[heighSegWriteIxd] = highSeg.elements[highSegReadIdx]
			highSeg.deleted[heighSegWriteIxd] = highSeg.deleted[highSegReadIdx]
			highSegReadIdx++
		} else {
			highSeg.elements[heighSegWriteIxd] = lowSeg.elements[lowSegReadIdx]
			highSeg.deleted[heighSegWriteIxd] = lowSeg.deleted[lowSegReadIdx]
			lowSegReadIdx++
		}
		heighSegWriteIxd++
	}

	// Copy remaining elements (only one of the segments contains it):
	copy(highSeg.elements[heighSegWriteIxd:], highSeg.elements[highSegReadIdx:])
	copy(highSeg.deleted[heighSegWriteIxd:], highSeg.deleted[highSegReadIdx:])
	copy(highSeg.elements[heighSegWriteIxd:], lowSeg.elements[lowSegReadIdx:])
	copy(highSeg.deleted[heighSegWriteIxd:], lowSeg.deleted[lowSegReadIdx:])

	highSeg.minNonDeletedIdx = 0
	highSeg.maxNonDeletedIdx = len(highSeg.elements) - 1
	highSeg.deletedNum += lowSeg.deletedNum
}

// Merge lowSeg and highSeg into highSeg using highSeg free space at the beginning.
// Preserve FIFO order for deleting. Maintain min/max non-deleted indexes.
func mergeSegmentsForDel[T any](lowSeg, highSeg *segment[T], cmp CmpFunc[T], highSegReadIdx int) {
	lowSegReadIdx := 0
	lowSegEnd := len(lowSeg.elements)
	highSegWriteIdx := highSegReadIdx - lowSegEnd
	highSegEnd := highSegReadIdx + lowSegEnd

	for highSegReadIdx < highSegEnd && lowSegReadIdx < lowSegEnd {
		cmpResult := cmp(highSeg.elements[highSegReadIdx], lowSeg.elements[lowSegReadIdx])
		if (cmpResult > 0) || (cmpResult == 0 && !lowSeg.deleted[lowSegReadIdx]) {
			highSeg.elements[highSegWriteIdx] = lowSeg.elements[lowSegReadIdx]
			highSeg.deleted[highSegWriteIdx] = lowSeg.deleted[lowSegReadIdx]
			lowSegReadIdx++
		} else {
			highSeg.elements[highSegWriteIdx] = highSeg.elements[highSegReadIdx]
			highSeg.deleted[highSegWriteIdx] = highSeg.deleted[highSegReadIdx]
			highSegReadIdx++
		}
		if !highSeg.deleted[highSegWriteIdx] {
			highSeg.maxNonDeletedIdx = max(highSeg.maxNonDeletedIdx, highSegWriteIdx)
			highSeg.minNonDeletedIdx = min(highSeg.minNonDeletedIdx, highSegWriteIdx)
		}
		highSegWriteIdx++
	}

	for highSegReadIdx < highSegEnd {
		highSeg.elements[highSegWriteIdx] = highSeg.elements[highSegReadIdx]
		highSeg.deleted[highSegWriteIdx] = highSeg.deleted[highSegReadIdx]
		if !highSeg.deleted[highSegWriteIdx] {
			highSeg.maxNonDeletedIdx = max(highSeg.maxNonDeletedIdx, highSegWriteIdx)
			highSeg.minNonDeletedIdx = min(highSeg.minNonDeletedIdx, highSegWriteIdx)
		}
		highSegWriteIdx++
		highSegReadIdx++
	}
	for lowSegReadIdx < lowSegEnd {
		highSeg.elements[highSegWriteIdx] = lowSeg.elements[lowSegReadIdx]
		highSeg.deleted[highSegWriteIdx] = lowSeg.deleted[lowSegReadIdx]
		if !highSeg.deleted[highSegWriteIdx] {
			highSeg.maxNonDeletedIdx = max(highSeg.maxNonDeletedIdx, highSegWriteIdx)
			highSeg.minNonDeletedIdx = min(highSeg.minNonDeletedIdx, highSegWriteIdx)
		}
		highSegWriteIdx++
		lowSegReadIdx++
	}

	highSeg.deletedNum += lowSeg.deletedNum
}

func demoteSegment[T any](from segment[T], to *segment[T]) {
	for r, w := 0, 0; r < len(from.elements); r++ {
		if from.deleted[r] {
			continue
		}
		to.elements[w] = from.elements[r]
		to.deleted[w] = false
		w++
	}
	to.deletedNum = 0 // Since demoteSegment is called only when we have exact len(to.elements) undeleted elements in from.
	to.minNonDeletedIdx, to.maxNonDeletedIdx = 0, len(to.elements)-1
}

// moveNonDeletedValuesToSegmentStart moves all non-deleted values to the beginning of the segment, preserving their order.
// It is used when we have half of the elements in the segment deleted, as preparation for merging with lover segment.
func moveNonDeletedValuesToSegmentStart[T any](seg segment[T]) {
	length := len(seg.elements)
	writePointer := length - 1
	readPointer := length - 1
	for writePointer >= (length >> 1) {
		if !seg.deleted[readPointer] {
			seg.elements[writePointer] = seg.elements[readPointer]
			seg.deleted[writePointer] = false
			writePointer--
		}
		readPointer--
	}
	seg.deletedNum = length >> 1
	seg.minNonDeletedIdx, seg.maxNonDeletedIdx = length>>1, length-1
}

// returns index of the rightmost element equal to val that is not deleted.
func (s *segment[T]) findRightmostNotDeleted(cmp CmpFunc[T], val T) int {
	b, e := s.minNonDeletedIdx, s.maxNonDeletedIdx+1
	elems := s.elements
	del := s.deleted
	for b < e {
		m := (b + e) >> 1
		cmpRes := cmp(val, elems[m])
		switch {
		case cmpRes < 0:
			e = m
		case cmpRes > 0:
			b = m + 1
		default: // elements are equal - follow invariant: deleted elements are to the right (higher index) of non-deleted ones.
			if del[m] {
				e = m
			} else {
				b = m + 1
			}
		}
	}

	idx := b
	if idx == 0 {
		return -1
	}
	idx--
	if s.deleted[idx] {
		return -1
	}
	if cmp(s.elements[idx], val) != 0 {
		return -1
	}
	return idx
}

// returns minimum element index with respect to FIFO constraint: if we have
// several equal minimum elements, returns the rightmost one.
func (s *segment[T]) min(cmp CmpFunc[T]) int {
	minIdx, maxIdx := s.minNonDeletedIndex(), s.maxNonDeletedIndex()
	for i := minIdx + 1; i <= maxIdx; i++ {
		if s.deleted[i] { // deleted elements can appear only after non-deleted equal ones;
			return minIdx
		}
		if cmp(s.elements[i], s.elements[minIdx]) != 0 {
			return minIdx
		}
		minIdx = i
	}
	return minIdx
}

// returns index of the first element that is greater or equal to val and is not deleted.
// If all elements are less than val, returns -1.
func (s *segment[T]) findGTOE(cmp CmpFunc[T], val T) int {
	elems := s.elements
	b, e := s.minNonDeletedIdx, s.maxNonDeletedIdx+1
	for b < e {
		m := (b + e) >> 1
		cmpRes := cmp(val, elems[m])
		if cmpRes <= 0 {
			e = m
		} else {
			b = m + 1
		}
	}
	if b > s.maxNonDeletedIdx {
		return -1
	}
	return s.nextNonDeletedAfter(b - 1)
}

// returns index of the first element that is less than val and is not deleted.
// If all elements are greater or equal to val, returns -1.
func (s *segment[T]) findLess(cmp CmpFunc[T], val T) int {
	elems := s.elements
	b, e := s.minNonDeletedIdx-1, s.maxNonDeletedIdx
	for b < e {
		m := (b+e)>>1 + 1
		cmpRes := cmp(val, elems[m])
		if cmpRes > 0 {
			b = m
		} else {
			e = m - 1
		}
	}
	return s.prevNonDeletedBefore(e + 1)
}

func (s *segment[T]) minNonDeletedIndex() (index int) {
	for i := s.minNonDeletedIdx; i < len(s.deleted); i++ {
		if !s.deleted[i] {
			s.minNonDeletedIdx = i
			return i
		}
	}
	return -1
}

func (s *segment[T]) maxNonDeletedIndex() (index int) {
	for i := s.maxNonDeletedIdx; i >= 0; i-- {
		if !s.deleted[i] {
			s.maxNonDeletedIdx = i
			return i
		}
	}
	return -1
}

func (s *segment[T]) nextNonDeletedAfter(index int) int {
	l := len(s.deleted)
	for i := index + 1; i < l; i++ {
		if !s.deleted[i] {
			return i
		}
	}
	return l
}

func (s *segment[T]) prevNonDeletedBefore(index int) int {
	for i := index - 1; i >= 0; i-- {
		if !s.deleted[i] {
			return i
		}
	}
	return -1
}

func (s *segment[T]) deepCopy() segment[T] {
	newSeg := segment[T]{
		elements:         make([]T, len(s.elements)),
		deleted:          make([]bool, len(s.deleted)),
		deletedNum:       s.deletedNum,
		minNonDeletedIdx: s.minNonDeletedIdx,
		maxNonDeletedIdx: s.maxNonDeletedIdx,
	}
	copy(newSeg.elements, s.elements)
	copy(newSeg.deleted, s.deleted)
	return newSeg
}

func calculateWhiteSegmentsQuantity(capacity int) int {
	if capacity < 0 {
		panic("negative capacity")
	}
	if capacity == 0 {
		return 0
	}
	return int(math.Log2(float64(capacity)) + 1) // Maybe: rewrite without using math (bit operations)?
}

func log2(x uint64) int {
	return bits.TrailingZeros64(x)
}
