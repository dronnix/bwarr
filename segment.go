package bwarr

import (
	"math"
	"math/bits"
)

type segment[T any] struct {
	elements   []T            // Stores user's data.
	deleted    *LayeredBitSet // Stores whether i-th element is deleted.
	deletedNum int            // Number of deleted elements in the segment.
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
		elements:   make([]T, l),
		deleted:    NewLayeredBitSet(l),
		deletedNum: 0,
	}
}

// Merge lowSeg and highSeg into highSeg using highSeg free space at the beginning.
func mergeSegments[T any](lowSeg, highSeg *segment[T], cmp CmpFunc[T], highSegReadIdx int) {
	if lowSeg.deletedNum == 0 && highSeg.deletedNum == 0 {
		mergeSegmentsClean(lowSeg, highSeg, cmp, highSegReadIdx)
	} else {
		mergeSegmentsDirty(lowSeg, highSeg, cmp, highSegReadIdx)
	}
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
	lowElems := lowSeg.elements[:lowSegEnd]

	// Write pointer is always behind the read pointer (gap = remaining lowSeg elements),
	// so writes never overwrite unread positions — safe to read/write in-place.
	lowSegReadIdx := 0

	for highSegReadIdx < len(highElems) && lowSegReadIdx < len(lowElems) {
		cmpResult := cmp(highElems[highSegReadIdx], lowElems[lowSegReadIdx])
		if (cmpResult < 0) || (cmpResult == 0 && !highSeg.deleted.Get(highSegReadIdx)) { // TODO: Call get only once
			highElems[highSegWriteIdx] = highElems[highSegReadIdx]
			setOrUnset(highSeg.deleted, highSegWriteIdx, highSeg.deleted.Get(highSegReadIdx)) // TODO: Use ResetFrom before copying, and SetIfTrue here.
			highSegReadIdx++
		} else {
			highElems[highSegWriteIdx] = lowElems[lowSegReadIdx]
			setOrUnset(highSeg.deleted, highSegWriteIdx, lowSeg.deleted.Get(lowSegReadIdx)) // TODO: Call get only once
			lowSegReadIdx++
		}
		highSegWriteIdx++
	}

	// TODO: Use copy and CopyFrom here.
	for highSegReadIdx < highSegEnd {
		highElems[highSegWriteIdx] = highElems[highSegReadIdx]
		setOrUnset(highSeg.deleted, highSegWriteIdx, highSeg.deleted.Get(highSegReadIdx))
		highSegWriteIdx++
		highSegReadIdx++
	}
	for lowSegReadIdx < lowSegEnd {
		highElems[highSegWriteIdx] = lowElems[lowSegReadIdx]
		setOrUnset(highSeg.deleted, highSegWriteIdx, lowSeg.deleted.Get(lowSegReadIdx))
		highSegWriteIdx++
		lowSegReadIdx++
	}

	highSeg.deletedNum += lowSeg.deletedNum
}

// Merge lowSeg and highSeg into highSeg using highSeg free space at the beginning.
// Preserve FIFO order for deleting.
func mergeSegmentsForDel[T any](lowSeg, highSeg *segment[T], cmp CmpFunc[T], highSegReadIdx int) {
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
		cmpResult := cmp(highElems[highSegReadIdx], lowElems[lowSegReadIdx])
		if (cmpResult > 0) || (cmpResult == 0 && !lowSeg.deleted.Get(lowSegReadIdx)) { // TODO: Call get only once;
			highElems[highSegWriteIdx] = lowElems[lowSegReadIdx]
			setOrUnset(highSeg.deleted, highSegWriteIdx, lowSeg.deleted.Get(lowSegReadIdx))
			lowSegReadIdx++
		} else {
			highElems[highSegWriteIdx] = highElems[highSegReadIdx]
			setOrUnset(highSeg.deleted, highSegWriteIdx, highSeg.deleted.Get(highSegReadIdx))
			highSegReadIdx++
		}
		highSegWriteIdx++
	}

	for highSegReadIdx < len(highElems) {
		highElems[highSegWriteIdx] = highElems[highSegReadIdx]
		setOrUnset(highSeg.deleted, highSegWriteIdx, highSeg.deleted.Get(highSegReadIdx))
		highSegWriteIdx++
		highSegReadIdx++
	}
	for lowSegReadIdx < len(lowElems) {
		highElems[highSegWriteIdx] = lowElems[lowSegReadIdx]
		setOrUnset(highSeg.deleted, highSegWriteIdx, lowSeg.deleted.Get(lowSegReadIdx))
		highSegWriteIdx++
		lowSegReadIdx++
	}

	highSeg.deletedNum += lowSeg.deletedNum
}

func demoteSegment[T any](from segment[T], to *segment[T]) {
	for r, w := 0, 0; r < len(from.elements); r++ {
		if from.deleted.Get(r) {
			continue
		}
		to.elements[w] = from.elements[r]
		w++
	}
	to.deletedNum = 0 // Since demoteSegment is called only when we have exact len(to.elements) undeleted elements in from.
	to.deleted.Reset()
}

// moveNonDeletedValuesToSegmentEnd moves all non-deleted values to the end of the segment, preserving their order.
// It is used when a half of the elements in the segment deleted, as preparation for merging with lower segment.
func moveNonDeletedValuesToSegmentEnd[T any](seg segment[T]) {
	length := len(seg.elements)
	halfLen := length >> 1
	// Write pointer >= read pointer always, so reads see original bits.
	// Every position from length-1 down to halfLen is written exactly once (Unset).
	writePointer := length - 1
	readPointer := length - 1
	for writePointer >= halfLen {
		if !seg.deleted.Get(readPointer) {
			seg.elements[writePointer] = seg.elements[readPointer]
			seg.deleted.Unset(writePointer)
			writePointer--
		}
		readPointer--
	}
	seg.deletedNum = halfLen
}

func setOrUnset(bs *LayeredBitSet, idx int, value bool) {
	if value {
		bs.Set(idx)
	} else {
		bs.Unset(idx)
	}
}

// returns index of the rightmost element equal to val that is not deleted.
func (s *segment[T]) findRightmostNotDeleted(cmp CmpFunc[T], val T) int {
	maxNonDel := s.maxNonDeletedIndex()
	if maxNonDel < 0 {
		return -1
	}
	// Sub-slice for BCE: the compiler tracks len(elems) through e's mutations.
	elems := s.elements[:maxNonDel+1]
	b := s.minNonDeletedIndex()
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
			if s.deleted.Get(m) { // TODO: use FindPrevUnsetBit here
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
	if s.deleted.Get(idx) {
		return -1
	}
	if cmp(elems[idx], val) != 0 {
		return -1
	}
	return idx
}

// returns minimum element index with respect to FIFO constraint: if we have
// several equal minimum elements, returns the rightmost one.
func (s *segment[T]) min(cmp CmpFunc[T]) int { // TODO: review and rewrite
	minIdx := s.minNonDeletedIndex()
	maxIdx := s.maxNonDeletedIndex()
	for i := s.deleted.FindNextUnsetBit(minIdx); i >= 0 && i <= maxIdx; i = s.deleted.FindNextUnsetBit(i) {
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
	maxNonDel := s.maxNonDeletedIndex()
	if maxNonDel < 0 {
		return -1
	}
	elems := s.elements[:maxNonDel+1]
	b := s.minNonDeletedIndex()
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
	if b > maxNonDel {
		return -1
	}
	return s.nextNonDeletedAfter(b - 1)
}

// returns index of the first element that is less than val and is not deleted.
// If all elements are greater or equal to val, returns -1.
func (s *segment[T]) findLess(cmp CmpFunc[T], val T) int {
	maxNonDel := s.maxNonDeletedIndex()
	if maxNonDel < 0 {
		return -1
	}
	elems := s.elements[:maxNonDel+1]
	b, e := s.minNonDeletedIndex()-1, maxNonDel
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

func (s *segment[T]) minNonDeletedIndex() int {
	return s.deleted.FindFirstUnsetBit()
}

func (s *segment[T]) maxNonDeletedIndex() int {
	return s.deleted.FindLastUnsetBit()
}

func (s *segment[T]) nextNonDeletedAfter(index int) int { // TODO: replace with FindLastUnsetBit
	var idx int
	if index < 0 {
		idx = s.deleted.FindFirstUnsetBit()
	} else {
		idx = s.deleted.FindNextUnsetBit(index)
	}
	if idx < 0 {
		return len(s.elements)
	}
	return idx
}

func (s *segment[T]) prevNonDeletedBefore(index int) int { // TODO: replace with FindNextUnsetBit
	if index >= len(s.elements) {
		return s.deleted.FindLastUnsetBit()
	}
	return s.deleted.FindPrevUnsetBit(index)
}

func (s *segment[T]) deepCopy() segment[T] {
	newSeg := segment[T]{
		elements:   make([]T, len(s.elements)),
		deleted:    s.deleted.DeepCopy(),
		deletedNum: s.deletedNum,
	}
	copy(newSeg.elements, s.elements)
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

func rightmostTrueBitPosition(x int) int {
	return bits.TrailingZeros64(uint64(x)) //nolint: gosec // x is always non-negative, so it is safe to convert it to uint64.
}
