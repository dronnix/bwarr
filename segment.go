package bwarr

import (
	"math/bits"
)

type segment[T any] struct {
	elements []T // Stores user's data.
	deleted  []bool
	fenwick  []int // Fenwick sums. fenwick[i] = sum(deleted[i - lsb(i) + 1, i])
	minIndex int
	maxIndex int
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
		elements: make([]T, l),
		deleted:  make([]bool, l),
		fenwick:  make([]int, l+1),
		minIndex: 0,
		maxIndex: l - 1,
	}
}

// mergeSegmentsDirty merges two ordered segments.
// FIFO invariants for equal elements: the newer element goes first, deleted elements
// go after non-deleted ones. lowSegIsNewer tells which segment holds the newer elements:
// the high one on insert cascades, the low one on delete consolidation.
func mergeSegmentsDirty[T any](lowSeg, highSeg *segment[T], cmp CmpFunc[T], highSegReadIdx int, lowSegIsNewer bool) {
	lowSegEnd := len(lowSeg.elements)
	highSegWriteIdx := highSegReadIdx - lowSegEnd
	highSegEnd := highSegReadIdx + lowSegEnd

	// Sub-slice so the compiler can prove loop indices are in bounds (BCE).
	highElems := highSeg.elements[:highSegEnd]
	lowElems := lowSeg.elements[:lowSegEnd]

	// Write pointer is always behind the read pointer (gap = remaining lowSeg elements),
	// so writes never overwrite unread positions — safe to read/write in-place.
	lowSegReadIdx := 0

	for highSegReadIdx < len(highElems) && lowSegReadIdx < len(lowElems) {
		var takeLow bool
		cmpResult := cmp(highElems[highSegReadIdx], lowElems[lowSegReadIdx])
		if cmpResult != 0 {
			takeLow = cmpResult > 0
		} else if lowSegIsNewer { // Equal elements: take the newer one, unless it is deleted.
			takeLow = !lowSeg.deleted[lowSegReadIdx]
		} else {
			takeLow = highSeg.deleted[highSegReadIdx]
		}
		if takeLow {
			highElems[highSegWriteIdx] = lowElems[lowSegReadIdx]
			highSeg.deleted[highSegWriteIdx] = lowSeg.deleted[lowSegReadIdx]
			lowSegReadIdx++
		} else {
			highElems[highSegWriteIdx] = highElems[highSegReadIdx]
			highSeg.deleted[highSegWriteIdx] = highSeg.deleted[highSegReadIdx]
			highSegReadIdx++
		}
		highSegWriteIdx++
	}

	// At most one of the tails is non-empty. TODO: Copy tail bits word-wise (needs a shifted bit-range copy).
	copyTailWithBits(highSeg, highSegWriteIdx, highSeg, highSegReadIdx, highSegEnd)
	copyTailWithBits(highSeg, highSegWriteIdx, lowSeg, lowSegReadIdx, lowSegEnd)
}

// copyTailWithBits copies src elements and their deleted bits from [readIdx, end) into dst starting at writeIdx.
func copyTailWithBits[T any](dst *segment[T], writeIdx int, src *segment[T], readIdx, end int) {
	for ; readIdx < end; readIdx++ {
		dst.elements[writeIdx] = src.elements[readIdx]
		dst.deleted[writeIdx] = src.deleted[readIdx]
		writeIdx++
	}
}

// from segment is twice bigger than to segment, but has exactly half of
// not-deleted elements. So in-place copy is correct and won't overflow
func demoteSegment[T any](from, to *segment[T]) {
	for r, w := 0, 0; r < len(from.elements); r++ {
		if !from.deleted[r] {
			to.elements[w] = from.elements[r]
			w++
		}
	}
	to.resetDeleted()
}

// moveNonDeletedValuesToSegmentEnd moves all non-deleted values to the end of the segment, preserving their order.
// It is used when a half of the elements in the segment deleted, as preparation for merging with lower segment.
func moveNonDeletedValuesToSegmentEnd[T any](seg *segment[T]) {
	length := len(seg.elements)
	halfLen := length >> 1
	// Write pointer >= read pointer always, so reads see original bits.
	// Every position from length-1 down to halfLen is written exactly once (Unset).
	writePointer := length - 1
	readPointer := length - 1
	for writePointer >= halfLen {
		if !seg.deleted[readPointer] {
			seg.elements[writePointer] = seg.elements[readPointer]
			seg.deleted[writePointer] = false
			writePointer--
		}
		readPointer--
	}
}

// When upperBound is true returns index of the first element greater than to val o len if all elements are
//
//	smaller than val.
//
// for a range of the equal elements 'val' the first deleted element 'val' is considered to be strictly greater than val.
//
// When upperBound is false returns index of the first element greater than or equal to val or 'len' if all elements are
//
//	smaller than val.
func (s *segment[T]) findBound(cmp CmpFunc[T], val T, upperBound bool) int {
	b := 0
	e := len(s.elements)
	// invariant: elems[b] <= val < s.elements[e]
	for b < e {
		m := (b + e) >> 1
		cmpRes := cmp(val, s.elements[m])
		switch {
		case cmpRes < 0:
			e = m
		case cmpRes > 0:
			b = m + 1
		default: // elements are equal - follow invariant: deleted elements are to the right (higher index) of non-deleted ones.
			if !s.deleted[m] && upperBound {
				b = m + 1
			} else {
				e = m
			}
		}
	}

	return b
}

// returns index of the rightmost not-deleted element equal to val or -1 if such element does not exist.
// This returns oldest not-deleted element from the equal range.
func (s *segment[T]) findRightmostNotDeleted(cmp CmpFunc[T], val T) int {
	if cmp(val, s.elements[0]) < 0 {
		return -1
	}
	upperBound := s.findBound(cmp, val /*upperBound*/, true)

	if upperBound == 0 {
		return -1
	}
	upperBound--
	if cmp(s.elements[upperBound], val) != 0 {
		return -1
	}
	return upperBound
}

// returns minimum element index with respect to FIFO constraint: if we have
// several equal minimum elements, returns the rightmost (the oldest) one.
func (s *segment[T]) min(cmp CmpFunc[T]) int {
	if s.deleted[s.minIndex] ||
		(s.minIndex < len(s.elements)-1 && !s.deleted[s.minIndex+1] &&
			cmp(s.elements[s.minIndex], s.elements[s.minIndex+1]) == 0) {
		s.minIndex = s.minNonDeletedIndex(cmp)
	}
	return s.minIndex
}

// returns index of the leftmost element that is greater or equal to val and is not deleted.
// If all elements are less than val, returns -1.
func (s *segment[T]) findGTOE(cmp CmpFunc[T], val T) int {
	lowerBound := s.findBound(cmp, val /*upperBound*/, false)
	if lowerBound == len(s.elements) {
		return -1
	}
	if s.deleted[lowerBound] {
		lowerBound = s.nextNonDeletedAfter(lowerBound)
		if lowerBound == len(s.elements) {
			return -1
		}
	}
	return lowerBound
}

// returns index of the rightmost element that is less than val and is not deleted.
// If all elements are greater or equal to val, returns -1.
func (s *segment[T]) findLess(cmp CmpFunc[T], val T) int {
	lowerBound := s.findBound(cmp, val /*upperBound*/, false)
	lowerBound--
	if lowerBound >= 0 && s.deleted[lowerBound] {
		lowerBound = s.prevNonDeletedBefore(lowerBound)
	}
	return lowerBound
}

// If there are several equal mins, the index of the oldest inserted element is returned
func (s *segment[T]) minNonDeletedIndex(cmp CmpFunc[T]) int {
	firstNonDeleted := s.nextNonDeletedAfter(-1)
	return s.findRightmostNotDeleted(cmp, s.elements[firstNonDeleted])
}

func (s *segment[T]) maxNonDeletedIndex() int {
	if s.deleted[s.maxIndex] {
		s.maxIndex = s.prevNonDeletedBefore(len(s.elements))
	}
	return s.maxIndex
}

func (s *segment[T]) nextNonDeletedAfter(index int) int {
	deletedToIndex := 0
	if index >= 0 {
		deletedToIndex = s.numDeleted(index)
	}
	b := index + 1
	e := len(s.deleted)
	// allDeleted(b), someNonDeleted(e)
	for b < e {
		m := b + ((e - b) >> 1)
		// [index + 1, m + 1) at least one '0'
		if s.numDeleted(m)-deletedToIndex < m-index {
			e = m
		} else {
			b = m + 1
		}
	}
	return b
}

func (s *segment[T]) prevNonDeletedBefore(index int) int {
	e := index - 1
	if e < 0 {
		return -1
	}
	b := 0
	endDeleted := s.numDeleted(e)

	for b < e {
		m := b + ((e - b) >> 1)
		mDeleted := s.numDeleted(m)
		// all elements from [m+1, e] are deleted
		if endDeleted-mDeleted == e-m {
			endDeleted = mDeleted
			e = m
		} else {
			b = m + 1
		}
	}
	if s.deleted[b] {
		return -1
	}
	return b
}

func (s *segment[T]) deepCopy() segment[T] {
	newSeg := segment[T]{
		elements: make([]T, len(s.elements)),
		deleted:  make([]bool, len(s.elements)),
		fenwick:  make([]int, len(s.elements)+1),
		minIndex: s.minIndex,
		maxIndex: s.maxIndex,
	}
	copy(newSeg.elements, s.elements)
	copy(newSeg.deleted, s.deleted)
	copy(newSeg.fenwick, s.fenwick)
	return newSeg
}

func (s *segment[T]) delete(index int, cmp CmpFunc[T]) {
	s.deleted[index] = true
	if index == s.minIndex {
		if index > 0 && cmp(s.elements[index-1], s.elements[index]) < 0 {
			s.minIndex += 1
		} else if index > 0 {
			s.minIndex -= 1
		}
	}
	if index == s.maxIndex {
		s.maxIndex -= 1
	}
	index++
	for index < len(s.fenwick) {
		s.fenwick[index]++
		index += index & (-index)
	}
	s.fenwick[0]++
}

// Num deleted elements till index (included). numDeleted(0) returns 1 if first element is deleted and 0 otherwise
func (s *segment[T]) numDeleted(index int) int {
	sum := 0
	index++
	for index > 0 {
		sum += s.fenwick[index]
		index ^= index & (-index)
	}
	return sum
}

// returns the total number of deleted elements in segment
func (s *segment[T]) initFenwick(cmp CmpFunc[T]) *segment[T] {
	clear(s.fenwick)
	//prefixSums := make([]int, len(s.elements)+1)
	s.minIndex = 0
	s.maxIndex = len(s.elements) - 1
	for i := range s.deleted {
		if s.deleted[i] {
			s.delete(i, cmp)
		}
	}
	return s
}

func (s *segment[T]) deletedNum() int {
	return s.fenwick[0]
}

func (s *segment[T]) resetDeleted() {
	clear(s.fenwick)
	clear(s.deleted)
	s.minIndex = 0
	s.maxIndex = len(s.elements) - 1
}

func calculateWhiteSegmentsQuantity(capacity int) int {
	if capacity < 0 {
		panic("negative capacity")
	}
	return bits.Len(uint(capacity)) // Rank of the highest segment needed for `capacity` elements, plus one.
}

func rightmostTrueBitPosition(x int) int {
	return bits.TrailingZeros64(uint64(x)) //nolint: gosec // x is always non-negative, so it is safe to convert it to uint64.
}
