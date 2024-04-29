package bwarr

import "math"

type BWArr[T any] struct {
	blackSegments []segment[T]
	whiteSegments []segment[T]
	total         int // Total number of elements in the array, including deleted ones.
	cmp           CmpFunc[T]
}

type segment[T any] struct {
	elements         []T    // Stores user's data.
	deleted          []bool // Stores whether i-th element is deleted.
	deletedNum       int    // Number of deleted elements in the segment.
	minNonDeletedIdx int    // Index of the first non-deleted element in the segment.
}

type CmpFunc[T any] func(a, b T) int

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
	panic("implement me")
}

func (bwa *BWArr[T]) DeleteMin() (deleted T, found bool) {
	if bwa.total == 0 {
		return deleted, false
	}
	seg, ind := bwa.min()
	return bwa.del(seg, ind), true
}

func (bwa *BWArr[T]) Len() int {
	panic("implement me")
}

func (bwa *BWArr[T]) Max() (maxElem T, found bool) {
	panic("implement me")
}

func (bwa *BWArr[T]) Min() (minElem T, found bool) {
	if bwa.total == 0 {
		return minElem, false
	}
	seg, ind := bwa.min()
	return bwa.whiteSegments[seg].elements[ind], true
}

func (bwa *BWArr[T]) Clear(dropSegments bool) {
	panic("implement me")
}

func (bwa *BWArr[T]) Clone() *BWArr[T] {
	panic("implement me")
}

func (bwa *BWArr[T]) del(segNum, index int) (deleted T) {
	seg := &bwa.whiteSegments[segNum]
	deleted = seg.elements[index]
	seg.deleted[index] = true
	seg.deletedNum++
	if index == seg.minNonDeletedIdx {
		seg.minNonDeletedIdx++
	}

	segmentCapacity := 1 << segNum
	if seg.deletedNum < segmentCapacity/2 {
		return deleted
	}
	if segNum == 0 {
		bwa.total--
		seg.minNonDeletedIdx, seg.deletedNum = 0, 0
		seg.deleted[0] = false
		return deleted
	}
	if (1<<(segNum-1))&bwa.total == 0 {
		demote(*seg, &bwa.whiteSegments[segNum-1])
		bwa.total -= 1 << segNum
		bwa.total += 1 << (segNum - 1)
	} else {
		demote(*seg, &bwa.blackSegments[segNum-1])
		mergeSegments(bwa.blackSegments[segNum-1], bwa.whiteSegments[segNum-1], bwa.cmp, seg)
		bwa.total -= 1 << (segNum - 1)
	}
	return deleted
}

// min assumes that there is at least one segment with elements!
func (bwa *BWArr[T]) min() (segNum, index int) {
	// First set result to the first segment with elements.
	for i := 0; i < len(bwa.whiteSegments); i++ {
		if bwa.total&(1<<i) != 0 {
			segNum = i
			break
		}
	}
	index = bwa.minNonDeletedIndex(segNum)
	// Then find the segment with the smallest element:
	for seg := segNum + 1; seg < len(bwa.whiteSegments); seg++ {
		if bwa.total&(1<<seg) == 0 {
			continue
		}
		// Less or equal is used to provide stable behavior (return the oldest one).
		ind := bwa.minNonDeletedIndex(seg)
		if bwa.cmp(bwa.whiteSegments[seg].elements[ind], bwa.whiteSegments[segNum].elements[index]) <= 0 {
			segNum, index = seg, ind
		}
	}
	return segNum, index
}

func (bwa *BWArr[T]) minNonDeletedIndex(segNum int) (index int) {
	seg := &bwa.whiteSegments[segNum]
	for i := seg.minNonDeletedIdx; i < len(seg.elements); i++ {
		if !seg.deleted[i] {
			seg.minNonDeletedIdx = i
			return i
		}
	}
	return -1
}

func (bwa *BWArr[T]) search(element T) (segNum, index int) {
	for segNum = len(bwa.whiteSegments) - 1; segNum >= 0; segNum-- {
		if bwa.total&(1<<segNum) == 0 {
			continue
		}
		if index = findRightmostNotDeleted(bwa.whiteSegments[segNum], bwa.cmp, element); index >= 0 {
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
		})
	bwa.blackSegments = append(bwa.blackSegments,
		segment[T]{
			elements:         make([]T, l),
			deleted:          make([]bool, l),
			deletedNum:       0,
			minNonDeletedIdx: 0,
		})
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
	result.minNonDeletedIdx = 0
}

func findRightmostNotDeleted[T any](seg segment[T], cmp CmpFunc[T], val T) int {
	b, e := uint64(seg.minNonDeletedIdx), uint64(len(seg.elements))
	elems := seg.elements
	del := seg.deleted
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
	idx := int(b)

	if idx == 0 {
		return -1
	}
	idx--
	if seg.deleted[idx] {
		return -1
	}
	if cmp(seg.elements[idx], val) != 0 {
		return -1
	}
	return idx
}

func demote[T any](from segment[T], to *segment[T]) {
	for r, w := 0, 0; r < len(from.elements); r++ {
		if from.deleted[r] {
			continue
		}
		to.elements[w] = from.elements[r]
		to.deleted[w] = false
		w++
	}
	to.deletedNum = 0 // Since demote is called only when we have exact len(to.elements) undeleted elements in from.
	to.minNonDeletedIdx = 0
}

func calculateWhiteSegmentsQuantity(capacity int) int {
	switch {
	case capacity == 0:
		return 2 // to avoid every time checking if capacity is 0
	case capacity < 0:
		panic("negative capacity")
	default:
		return int(math.Log2(float64(capacity)) + 1) // TODO: rewrite without using math?
	}
}

func createSegments[T any](num int) []segment[T] {
	segments := make([]segment[T], num)
	for i := 0; i < num; i++ {
		segments[i] = segment[T]{
			elements:         make([]T, 1<<i),
			deleted:          make([]bool, 1<<i),
			deletedNum:       0,
			minNonDeletedIdx: 0,
		}
	}
	return segments
}
