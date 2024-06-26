package bwarr

type BWArr[T any] struct {
	blackSegments []segment[T]
	whiteSegments []segment[T]
	total         int // Total number of elements in the array, including deleted ones.
	cmp           CmpFunc[T]
}

type CmpFunc[T any] func(a, b T) int

type IteratorFunc[T any] func(item T) bool

func New[T any](cmp CmpFunc[T], capacity int) *BWArr[T] {
	wSegNum := calculateWhiteSegmentsQuantity(capacity)
	bSegNum := wSegNum - 1
	if bSegNum < 0 {
		bSegNum = 0
	}

	return &BWArr[T]{
		blackSegments: createSegments[T](bSegNum),
		whiteSegments: createSegments[T](wSegNum),
		total:         0,
		cmp:           cmp,
	}
}

func (bwa *BWArr[T]) Insert(element T) {
	if bwa.total&1 == 0 { // whiteSegments[0] is free
		bwa.whiteSegments[0].elements[0] = element
		bwa.total++
		return
	}

	bwa.blackSegments[0].elements[0] = element
	for segNum := 1; segNum <= maxSegmentNumber; segNum++ {
		if bwa.total&(1<<segNum) == 0 {
			mergeSegments(bwa.whiteSegments[segNum-1], bwa.blackSegments[segNum-1], bwa.cmp, &bwa.whiteSegments[segNum])
			bwa.total++
			return
		}
		if len(bwa.blackSegments) == segNum {
			bwa.extend()
		}
		mergeSegments(bwa.whiteSegments[segNum-1], bwa.blackSegments[segNum-1], bwa.cmp, &bwa.blackSegments[segNum])
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
		bwa.blackSegments = bwa.blackSegments[:1]
	}
}

func (bwa *BWArr[T]) Clone() *BWArr[T] {
	// TODO: Call compaction after it will be implemented.
	newBWA := &BWArr[T]{
		blackSegments: make([]segment[T], len(bwa.blackSegments)),
		whiteSegments: make([]segment[T], len(bwa.whiteSegments)),
		total:         bwa.total,
		cmp:           bwa.cmp,
	}

	newBWA.blackSegments = createSegments[T](len(bwa.blackSegments))
	for i := range bwa.whiteSegments {
		newBWA.whiteSegments[i] = bwa.whiteSegments[i].deepCopy()
	}
	return newBWA
}

func (bwa *BWArr[T]) Ascend(iterator IteratorFunc[T]) {
	iter := createIterator(bwa)
	for val, ok := iter.next(); ok; val, ok = iter.next() {
		if !iterator(*val) {
			break
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
		demoteSegment(*seg, &bwa.blackSegments[segNum-1])
		mergeSegments(bwa.blackSegments[segNum-1], bwa.whiteSegments[segNum-1], bwa.cmp, seg)
		bwa.total -= 1 << (segNum - 1)
	}
	return deleted
}

// min assumes that there is at least one segment with elements!
func (bwa *BWArr[T]) min() (segNum, index int) { //nolint:dupl
	// First set result to the first segment with elements.
	for i := 0; i < len(bwa.whiteSegments); i++ {
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
	for i := 0; i < len(bwa.whiteSegments); i++ {
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

func (bwa *BWArr[T]) extend() {
	l := len(bwa.whiteSegments[len(bwa.whiteSegments)-1].elements)
	bwa.whiteSegments = append(bwa.whiteSegments,
		segment[T]{
			elements:         make([]T, l*2),
			deleted:          make([]bool, l*2),
			deletedNum:       0,
			minNonDeletedIdx: 0,
			maxNonDeletedIdx: l*2 - 1,
		})
	bwa.blackSegments = append(bwa.blackSegments,
		segment[T]{
			elements:         make([]T, l),
			deleted:          make([]bool, l),
			deletedNum:       0,
			minNonDeletedIdx: 0,
			maxNonDeletedIdx: l - 1,
		})
}
