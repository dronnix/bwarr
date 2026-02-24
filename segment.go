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
	if lowSeg.deletedNum == 0 && highSeg.deletedNum == 0 {
		mergeSegmentsClean(lowSeg, highSeg, cmp, highSegReadIdx)
	} else {
		mergeSegmentsDirty(lowSeg, highSeg, cmp, highSegReadIdx)
	}

	highSeg.minNonDeletedIdx = 0
	highSeg.maxNonDeletedIdx = len(highSeg.elements) - 1
}

// mergeSegmentsClean is the fast path for merging segments with no deleted elements.
func mergeSegmentsClean[T any](lowSeg, highSeg *segment[T], cmp CmpFunc[T], highSegReadIdx int) {
	lowSegEnd := len(lowSeg.elements)
	highSegWriteIdx := highSegReadIdx - lowSegEnd
	highSegEnd := highSegReadIdx + lowSegEnd

	// Sub-slice so the compiler can prove loop indices are in bounds (BCE).
	highElems := highSeg.elements[:highSegEnd]
	lowElems := lowSeg.elements[:lowSegEnd]

	lowSegReadIdx := 0

	for highSegReadIdx < len(highElems) && lowSegReadIdx < len(lowElems) {
		if cmp(highElems[highSegReadIdx], lowElems[lowSegReadIdx]) <= 0 {
			highElems[highSegWriteIdx] = highElems[highSegReadIdx]
			highSegReadIdx++
		} else {
			highElems[highSegWriteIdx] = lowElems[lowSegReadIdx]
			lowSegReadIdx++
		}
		highSegWriteIdx++
	}

	copy(highSeg.elements[highSegWriteIdx:], highSeg.elements[highSegReadIdx:highSegEnd])
	copy(highSeg.elements[highSegWriteIdx:], lowSeg.elements[lowSegReadIdx:lowSegEnd])
}

// mergeSegmentsDirty is the slow path for merging segments that have deleted elements.
func mergeSegmentsDirty[T any](lowSeg, highSeg *segment[T], cmp CmpFunc[T], highSegReadIdx int) {
	lowSegEnd := len(lowSeg.elements)
	highSegWriteIdx := highSegReadIdx - lowSegEnd
	highSegEnd := highSegReadIdx + lowSegEnd

	// Sub-slice so the compiler can prove loop indices are in bounds (BCE).
	highElems := highSeg.elements[:highSegEnd]
	highDel := highSeg.deleted[:highSegEnd]
	lowElems := lowSeg.elements[:lowSegEnd]
	lowDel := lowSeg.deleted[:lowSegEnd]

	lowSegReadIdx := 0

	for highSegReadIdx < len(highElems) && lowSegReadIdx < len(lowElems) {
		cmpResult := cmp(highElems[highSegReadIdx], lowElems[lowSegReadIdx])
		if (cmpResult < 0) || (cmpResult == 0 && !highDel[highSegReadIdx]) {
			highElems[highSegWriteIdx] = highElems[highSegReadIdx]
			highDel[highSegWriteIdx] = highDel[highSegReadIdx]
			highSegReadIdx++
		} else {
			highElems[highSegWriteIdx] = lowElems[lowSegReadIdx]
			highDel[highSegWriteIdx] = lowDel[lowSegReadIdx]
			lowSegReadIdx++
		}
		highSegWriteIdx++
	}

	copy(highSeg.elements[highSegWriteIdx:], highSeg.elements[highSegReadIdx:highSegEnd])
	copy(highSeg.deleted[highSegWriteIdx:], highSeg.deleted[highSegReadIdx:highSegEnd])
	copy(highSeg.elements[highSegWriteIdx:], lowSeg.elements[lowSegReadIdx:lowSegEnd])
	copy(highSeg.deleted[highSegWriteIdx:], lowSeg.deleted[lowSegReadIdx:lowSegEnd])

	highSeg.deletedNum += lowSeg.deletedNum
}

// Merge lowSeg and highSeg into highSeg using highSeg free space at the beginning.
// Preserve FIFO order for deleting. Maintain min/max non-deleted indexes.
func mergeSegmentsForDel[T any](lowSeg, highSeg *segment[T], cmp CmpFunc[T], highSegReadIdx int) {
	lowSegEnd := len(lowSeg.elements)
	highSegWriteIdx := highSegReadIdx - lowSegEnd
	highSegEnd := highSegReadIdx + lowSegEnd

	// Sub-slice so the compiler can prove loop indices are in bounds (BCE).
	highElems := highSeg.elements[:highSegEnd]
	highDel := highSeg.deleted[:highSegEnd]
	lowElems := lowSeg.elements[:lowSegEnd]
	lowDel := lowSeg.deleted[:lowSegEnd]

	lowSegReadIdx := 0

	for highSegReadIdx < len(highElems) && lowSegReadIdx < len(lowElems) {
		var del bool
		cmpResult := cmp(highElems[highSegReadIdx], lowElems[lowSegReadIdx])
		if (cmpResult > 0) || (cmpResult == 0 && !lowDel[lowSegReadIdx]) {
			highElems[highSegWriteIdx] = lowElems[lowSegReadIdx]
			del = lowDel[lowSegReadIdx]
			highDel[highSegWriteIdx] = del
			lowSegReadIdx++
		} else {
			highElems[highSegWriteIdx] = highElems[highSegReadIdx]
			del = highDel[highSegReadIdx]
			highDel[highSegWriteIdx] = del
			highSegReadIdx++
		}
		if !del {
			highSeg.maxNonDeletedIdx = max(highSeg.maxNonDeletedIdx, highSegWriteIdx)
			highSeg.minNonDeletedIdx = min(highSeg.minNonDeletedIdx, highSegWriteIdx)
		}
		highSegWriteIdx++
	}

	for highSegReadIdx < len(highElems) {
		highElems[highSegWriteIdx] = highElems[highSegReadIdx]
		del := highDel[highSegReadIdx]
		highDel[highSegWriteIdx] = del
		if !del {
			highSeg.maxNonDeletedIdx = max(highSeg.maxNonDeletedIdx, highSegWriteIdx)
			highSeg.minNonDeletedIdx = min(highSeg.minNonDeletedIdx, highSegWriteIdx)
		}
		highSegWriteIdx++
		highSegReadIdx++
	}
	for lowSegReadIdx < len(lowElems) {
		highElems[highSegWriteIdx] = lowElems[lowSegReadIdx]
		del := lowDel[lowSegReadIdx]
		highDel[highSegWriteIdx] = del
		if !del {
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

// moveNonDeletedValuesToSegmentEnd moves all non-deleted values to the end of the segment, preserving their order.
// It is used when a half of the elements in the segment deleted, as preparation for merging with lower segment.
func moveNonDeletedValuesToSegmentEnd[T any](seg segment[T]) {
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
	// Sub-slice for BCE: the compiler tracks len(elems) through e's mutations.
	elems := s.elements[:s.maxNonDeletedIdx+1]
	del := s.deleted[:s.maxNonDeletedIdx+1]
	b := s.minNonDeletedIdx
	e := len(elems)
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
	if del[idx] {
		return -1
	}
	if cmp(elems[idx], val) != 0 {
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
	elems := s.elements[:s.maxNonDeletedIdx+1]
	b := s.minNonDeletedIdx
	e := len(elems)
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
	elems := s.elements[:s.maxNonDeletedIdx+1]
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

func log2(x int) int {
	return bits.TrailingZeros64(uint64(x)) //nolint: gosec // x is always non-negative, so it is safe to convert it to uint64.
}
