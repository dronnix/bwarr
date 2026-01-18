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

func resetSegment[T any](s *segment[T]) {
	length := len(s.deleted)
	for i := range length {
		s.deleted[i] = false
	}
	s.deletedNum = 0
	s.minNonDeletedIdx = 0
	s.maxNonDeletedIdx = length - 1
}

func mergeSegments[T any](seg1, seg2 segment[T], cmp CmpFunc[T], result *segment[T]) {
	i, j, k := 0, 0, 0
	for i < len(seg1.elements) && j < len(seg2.elements) {
		if cmp(seg1.elements[i], seg2.elements[j]) < 0 {
			result.elements[k] = seg1.elements[i]
			result.deleted[k] = seg1.deleted[i]
			i++
		} else {
			result.elements[k] = seg2.elements[j]
			result.deleted[k] = seg2.deleted[j]
			j++
		}
		k++
	}

	copy(result.elements[k:], seg1.elements[i:])
	copy(result.deleted[k:], seg1.deleted[i:])
	copy(result.elements[k:], seg2.elements[j:])
	copy(result.deleted[k:], seg2.deleted[j:])

	result.deletedNum = seg1.deletedNum + seg2.deletedNum
	result.minNonDeletedIdx, result.maxNonDeletedIdx = 0, len(result.elements)-1
}

func mergeSegments1[T any](oldSegment, newSegment *segment[T], cmp CmpFunc[T], readPointer int) {
	j := 0
	currentSegmentLength := len(oldSegment.elements)
	writePointer := readPointer - currentSegmentLength
	newSegmentEnd := readPointer + currentSegmentLength
	for readPointer < newSegmentEnd && j < currentSegmentLength {
		cmpResult := cmp(newSegment.elements[readPointer], oldSegment.elements[j])
		if (cmpResult < 0) || (cmpResult == 0 && !newSegment.deleted[readPointer]) {
			newSegment.elements[writePointer] = newSegment.elements[readPointer]
			newSegment.deleted[writePointer] = newSegment.deleted[readPointer]
			readPointer++
		} else {
			newSegment.elements[writePointer] = oldSegment.elements[j]
			newSegment.deleted[writePointer] = oldSegment.deleted[j]
			j++
		}
		writePointer++
	}

	for readPointer < newSegmentEnd {
		newSegment.elements[writePointer] = newSegment.elements[readPointer]
		newSegment.deleted[writePointer] = newSegment.deleted[readPointer]
		writePointer++
		readPointer++
	}
	for j < currentSegmentLength {
		newSegment.elements[writePointer] = oldSegment.elements[j]
		newSegment.deleted[writePointer] = oldSegment.deleted[j]
		writePointer++
		j++
	}

	newSegment.deletedNum += oldSegment.deletedNum
	newSegment.minNonDeletedIdx, newSegment.maxNonDeletedIdx = 0, len(newSegment.elements)-1
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

func demoteSegment1[T any](from segment[T]) {
	length := len(from.elements)
	writePointer := length - 1
	readPointer := length - 1
	for writePointer >= (length >> 1) {
		if !from.deleted[readPointer] {
			from.elements[writePointer] = from.elements[readPointer]
			from.deleted[writePointer] = false
			writePointer--
		}
		readPointer--
	}
	from.deletedNum = length >> 1
	from.minNonDeletedIdx, from.maxNonDeletedIdx = length>>1, length-1
}

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
		default:
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
