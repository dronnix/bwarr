package bwarr

import "slices"

type BWArr[T any] struct {
	whiteSegments []segment[T]
	highBlackSeg  segment[T]
	lowBlackSeg   segment[T]
	total         int // Total number of elements in the array, including deleted ones.
	cmp           CmpFunc[T]
}

type CmpFunc[T any] func(a, b T) int

type IteratorFunc[T any] func(item T) bool

func New[T any](cmp CmpFunc[T], capacity int) *BWArr[T] {
	bwa := &BWArr[T]{cmp: cmp, total: 0}

	wSegNum := calculateWhiteSegmentsQuantity(capacity)
	if wSegNum > 0 {
		bwa.whiteSegments = createSegments[T](0, wSegNum)
	}
	if wSegNum > 1 {
		bwa.highBlackSeg = makeSegment[T](wSegNum - 2) //nolint:mnd
	}
	if wSegNum > 2 { //nolint:mnd
		bwa.lowBlackSeg = makeSegment[T](wSegNum - 3) //nolint:mnd
	}

	return bwa
}

func NewFromSlice[T any](cmp CmpFunc[T], slice []T) *BWArr[T] {
	l := len(slice)
	if l == 0 {
		return New[T](cmp, 0)
	}

	copyFrom := 0
	wSegNum := calculateWhiteSegmentsQuantity(l)
	segs := make([]segment[T], wSegNum)
	total := 0
	rank := 0
	for l > 0 {
		mask := 1 << rank
		if mask&l == 0 {
			rank++
			continue
		}
		seg := makeSegment[T](rank)
		copyTo := copyFrom + mask
		copy(seg.elements, slice[copyFrom:copyTo])
		slices.SortFunc(seg.elements, cmp)
		copyFrom += mask

		segs[rank] = seg
		total |= mask
		l -= mask
		rank++
	}
	return &BWArr[T]{
		whiteSegments: segs,
		highBlackSeg:  makeSegment[T](0),
		lowBlackSeg:   makeSegment[T](0),
		total:         total,
		cmp:           cmp,
	}
}

func (bwa *BWArr[T]) Insert(element T) {
	if bwa.total&1 == 0 { // whiteSegments[0] is free
		bwa.ensureSeg(0)
		bwa.whiteSegments[0].elements[0] = element
		bwa.total++
		return
	}

	lowBlack := bwa.lowBlack(0)
	lowBlack.elements[0] = element
	for segNum := 1; segNum <= maxSegmentNumber; segNum++ {
		if bwa.total&(1<<segNum) == 0 {
			bwa.ensureSeg(segNum)
			mergeSegments(bwa.whiteSegments[segNum-1], *lowBlack, bwa.cmp, &bwa.whiteSegments[segNum])
			bwa.total++
			return
		}
		highBlack := bwa.highBlack(segNum)

		bwa.ensureSeg(segNum)
		mergeSegments(bwa.whiteSegments[segNum-1], *lowBlack, bwa.cmp, highBlack)
		swapSegments(lowBlack, highBlack)
	}
}

func (bwa *BWArr[T]) ReplaceOrInsert(element T) (old T, found bool) {
	seg, ind := bwa.search(element)
	if ind < 0 {
		bwa.Insert(element)
		return old, false
	}
	old = bwa.whiteSegments[seg].elements[ind]
	bwa.whiteSegments[seg].elements[ind] = element
	return old, true
}

func (bwa *BWArr[T]) Has(element T) bool {
	if _, index := bwa.search(element); index >= 0 {
		return true
	}
	return false
}

func (bwa *BWArr[T]) Get(element T) (res T, found bool) {
	if segNum, index := bwa.search(element); index >= 0 {
		return bwa.whiteSegments[segNum].elements[index], true
	}
	return
}

func (bwa *BWArr[T]) Delete(element T) (deleted T, found bool) {
	segNum, index := bwa.search(element)
	if segNum < 0 {
		return deleted, false
	}
	return bwa.del(segNum, index), true
}

func (bwa *BWArr[T]) DeleteMax() (deleted T, found bool) {
	if bwa.total == 0 {
		return deleted, false
	}
	seg, ind := bwa.max()
	return bwa.del(seg, ind), true
}

func (bwa *BWArr[T]) DeleteMin() (deleted T, found bool) {
	if bwa.total == 0 {
		return deleted, false
	}
	seg, ind := bwa.min()
	return bwa.del(seg, ind), true
}

func (bwa *BWArr[T]) Len() int {
	deleted := 0
	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) != 0 {
			deleted += bwa.whiteSegments[i].deletedNum
		}
	}
	return bwa.total - deleted
}

func (bwa *BWArr[T]) Max() (maxElem T, found bool) {
	if bwa.total == 0 {
		return maxElem, false
	}

	seg, ind := bwa.max()
	return bwa.whiteSegments[seg].elements[ind], true
}

func (bwa *BWArr[T]) Min() (minElem T, found bool) {
	if bwa.total == 0 {
		return minElem, false
	}
	seg, ind := bwa.min()
	return bwa.whiteSegments[seg].elements[ind], true
}

func (bwa *BWArr[T]) Clear(dropSegments bool) {
	bwa.total = 0
	if dropSegments {
		// TODO: drop all segments after introducing a smart getter.
		bwa.whiteSegments = bwa.whiteSegments[:2]
	}
}

func (bwa *BWArr[T]) Clone() *BWArr[T] {
	// TODO: Call compaction after it will be implemented.
	newBWA := &BWArr[T]{
		whiteSegments: make([]segment[T], len(bwa.whiteSegments)),
		highBlackSeg:  makeSegment[T](0),
		lowBlackSeg:   makeSegment[T](0),
		total:         bwa.total,
		cmp:           bwa.cmp,
	}

	for i := range bwa.whiteSegments {
		newBWA.whiteSegments[i] = bwa.whiteSegments[i].deepCopy()
	}
	return newBWA
}

func (bwa *BWArr[T]) Ascend(iterator IteratorFunc[T]) {
	iter := createAscIteratorBegin(bwa)
	for val, ok := iter.next(); ok; val, ok = iter.next() {
		if !iterator(*val) {
			break
		}
	}
}

func (bwa *BWArr[T]) AscendGreaterOrEqual(elem T, iterator IteratorFunc[T]) {
	iter := createAscIteratorGTOE(bwa, elem)
	for val, ok := iter.next(); ok; val, ok = iter.next() {
		if !iterator(*val) {
			break
		}
	}
}

func (bwa *BWArr[T]) AscendLessThan(elem T, iterator IteratorFunc[T]) {
	iter := createAscIteratorLess(bwa, elem)
	for val, ok := iter.next(); ok; val, ok = iter.next() {
		if !iterator(*val) {
			break
		}
	}
}

func (bwa *BWArr[T]) AscendRange(greaterOrEqual, lessThan T, iterator IteratorFunc[T]) {
	iter := createAscIteratorFromTo(bwa, greaterOrEqual, lessThan)
	for val, ok := iter.next(); ok; val, ok = iter.next() {
		if !iterator(*val) {
			break
		}
	}
}

func (bwa *BWArr[T]) Descend(iterator IteratorFunc[T]) {
	iter := createDescIteratorEnd(bwa)
	for val, ok := iter.prev(); ok; val, ok = iter.prev() {
		if !iterator(*val) {
			break
		}
	}
}

func (bwa *BWArr[T]) DescendGreaterOrEqual(elem T, iterator IteratorFunc[T]) {
	iter := createDescIteratorGTOE(bwa, elem)
	for val, ok := iter.prev(); ok; val, ok = iter.prev() {
		if !iterator(*val) {
			break
		}
	}
}

func (bwa *BWArr[T]) DescendLessThan(elem T, iterator IteratorFunc[T]) {
	iter := createDescIteratorLess(bwa, elem)
	for val, ok := iter.prev(); ok; val, ok = iter.prev() {
		if !iterator(*val) {
			break
		}
	}
}

func (bwa *BWArr[T]) DescendRange(greaterOrEqual, lessThan T, iterator IteratorFunc[T]) {
	iter := createDescIteratorFromTo(bwa, greaterOrEqual, lessThan)
	for val, ok := iter.prev(); ok; val, ok = iter.prev() {
		if !iterator(*val) {
			break
		}
	}
}

func (bwa *BWArr[T]) Compact() {
	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) == 0 {
			bwa.whiteSegments[i] = segment[T]{} //nolint:exhaustruct
		}
	}
}

func (bwa *BWArr[T]) del(segNum, index int) (deleted T) {
	seg := &bwa.whiteSegments[segNum]
	deleted = seg.elements[index]
	seg.deleted[index] = true
	seg.deletedNum++

	if index == seg.minNonDeletedIdx {
		seg.minNonDeletedIdx++
	}
	if index == seg.maxNonDeletedIdx {
		seg.maxNonDeletedIdx--
	}

	segmentCapacity := 1 << segNum
	if seg.deletedNum < segmentCapacity/2 {
		return deleted
	}
	if segNum == 0 {
		bwa.total--
		seg.deletedNum, seg.minNonDeletedIdx, seg.maxNonDeletedIdx = 0, 0, len(seg.elements)-1
		seg.deleted[0] = false
		return deleted
	}
	if (1<<(segNum-1))&bwa.total == 0 {
		demoteSegment(*seg, &bwa.whiteSegments[segNum-1])
		bwa.total -= 1 << segNum
		bwa.total += 1 << (segNum - 1)
	} else {
		blackSeg := bwa.highBlack(segNum - 1)
		demoteSegment(*seg, blackSeg)
		mergeSegments(*blackSeg, bwa.whiteSegments[segNum-1], bwa.cmp, seg)
		bwa.total -= 1 << (segNum - 1)
	}
	return deleted
}

// min assumes that there is at least one segment with elements!
func (bwa *BWArr[T]) min() (segNum, index int) { //nolint:dupl
	// First set result to the first segment with elements.
	for i := range len(bwa.whiteSegments) {
		if bwa.total&(1<<i) != 0 {
			segNum = i
			break
		}
	}
	index = bwa.whiteSegments[segNum].minNonDeletedIndex()
	// Then find the segment with the smallest element:
	for seg := segNum + 1; seg < len(bwa.whiteSegments); seg++ {
		if bwa.total&(1<<seg) == 0 {
			continue
		}
		// Less or equal is used to provide stable behavior (return the oldest one).
		ind := bwa.whiteSegments[seg].minNonDeletedIndex()
		if bwa.cmp(bwa.whiteSegments[seg].elements[ind], bwa.whiteSegments[segNum].elements[index]) <= 0 {
			segNum, index = seg, ind
		}
	}
	return segNum, index
}

// max assumes that there is at least one segment with elements!
func (bwa *BWArr[T]) max() (segNum, index int) { //nolint:dupl
	// First set result to the first segment with elements.
	for i := range len(bwa.whiteSegments) {
		if bwa.total&(1<<i) != 0 {
			segNum = i
			break
		}
	}
	index = bwa.whiteSegments[segNum].maxNonDeletedIndex()
	// Then find the segment with the smallest element:
	for seg := segNum + 1; seg < len(bwa.whiteSegments); seg++ {
		if bwa.total&(1<<seg) == 0 {
			continue
		}
		// Less or equal is used to provide stable behavior (return the oldest one).
		ind := bwa.whiteSegments[seg].maxNonDeletedIndex()
		if bwa.cmp(bwa.whiteSegments[seg].elements[ind], bwa.whiteSegments[segNum].elements[index]) >= 0 {
			segNum, index = seg, ind
		}
	}
	return segNum, index
}

func (bwa *BWArr[T]) search(element T) (segNum, index int) {
	for segNum = len(bwa.whiteSegments) - 1; segNum >= 0; segNum-- {
		if bwa.total&(1<<segNum) == 0 {
			continue
		}
		if index = bwa.whiteSegments[segNum].findRightmostNotDeleted(bwa.cmp, element); index >= 0 {
			return segNum, index
		}
	}
	return -1, -1
}

const maxSegmentNumber = 62

func (bwa *BWArr[T]) ensureSeg(rank int) {
	l := len(bwa.whiteSegments)
	if rank >= l {
		whites := make([]segment[T], rank-l+1)
		bwa.whiteSegments = append(bwa.whiteSegments, whites...)
	}
	if len(bwa.whiteSegments[rank].elements) == 0 {
		bwa.whiteSegments[rank] = makeSegment[T](rank)
	}
}

func (bwa *BWArr[T]) highBlack(rank int) *segment[T] {
	bwa.highBlackSeg = *reallocateSegment(&bwa.highBlackSeg, rank)
	return &bwa.highBlackSeg
}

func (bwa *BWArr[T]) lowBlack(rank int) *segment[T] {
	bwa.lowBlackSeg = *reallocateSegment(&bwa.lowBlackSeg, rank)
	return &bwa.lowBlackSeg
}
