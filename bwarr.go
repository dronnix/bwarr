// Package bwarr implements a Black-White Array, a fast, ordered
// data structure with O(log N) memory allocations and O(log N) amortized complexity for
// insert, delete, and search operations. Can store equal elements and maintains stable ordering.
// See data structure details at: https://arxiv.org/abs/2004.09051
package bwarr

import "slices"

// BWArr is a Black-White Array, a fast, ordered data structure with O(log N) memory allocations
// and O(log N) amortized complexity for insert, delete, and search operations. Can store equal
// elements and maintains stable ordering.
// See data structure details at: https://arxiv.org/abs/2004.09051
type BWArr[T any] struct {
	whiteSegments []segment[T]
	total         uint64 // Total number of elements in the array, including deleted ones.
	cmp           CmpFunc[T]
}

// CmpFunc is a comparison function that defines the ordering of elements.
// It should return:
//   - a negative value if a < b
//   - zero if a == b
//   - a positive value if a > b
type CmpFunc[T any] func(a, b T) int

// IteratorFunc is a callback function used for iterating over elements.
// It receives each element during iteration and should return true to
// continue iteration or false to stop early.
type IteratorFunc[T any] func(item T) bool

// New creates a new empty BWArr with the given comparison function CmpFunc and
// capacity hint. The capacity parameter provides an estimate of the expected
// number of elements to optimize initial memory allocation. Use 0 if the
// capacity is unknown.
func New[T any](cmp CmpFunc[T], capacity int) *BWArr[T] {
	bwa := &BWArr[T]{cmp: cmp, total: 0}

	wSegNum := calculateWhiteSegmentsQuantity(capacity)
	if wSegNum > 0 {
		bwa.whiteSegments = createSegments[T](0, wSegNum)
	}
	return bwa
}

// NewFromSlice creates a new BWArr from an existing slice of elements and a comparison
// function CmpFunc.
// This constructor is more efficient than creating an empty BWArr and inserting elements one by one.
// The original slice is not modified.
func NewFromSlice[T any](cmp CmpFunc[T], slice []T) *BWArr[T] {
	l := len(slice)
	total := uint64(l)
	if l == 0 {
		return New[T](cmp, 0)
	}

	copyFrom := 0
	wSegNum := calculateWhiteSegmentsQuantity(l)
	segs := make([]segment[T], wSegNum)
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
		l -= mask
		rank++
	}
	return &BWArr[T]{
		whiteSegments: segs,
		total:         total,
		cmp:           cmp,
	}
}

// Insert adds an element to the BWArr maintaining sorted order.
// The operation has O(log N) amortized time complexity. Note that one in
// every N insert operations may take O(N) time for segment consolidation.
//
// Duplicate elements are allowed. If multiple equal elements exist, they
// maintain stable ordering based on insertion order.
func (bwa *BWArr[T]) Insert(element T) {
	newSegmentSize := (bwa.total + 1) & -(bwa.total + 1)
	newSegmentRank := log2(newSegmentSize)
	bwa.ensureSeg(newSegmentRank)
	resultSegment := &bwa.whiteSegments[newSegmentRank]
	resetSegment(resultSegment)
	resultSegment.elements[newSegmentSize-1] = element
	resultReadPtr := int(newSegmentSize - 1)
	for segmentNumber := range newSegmentRank {
		mergeSegments1(&bwa.whiteSegments[segmentNumber], resultSegment, bwa.cmp, resultReadPtr)
		resultReadPtr -= 1 << segmentNumber
	}
	bwa.total++
}

// ReplaceOrInsert inserts an element into the BWArr, or replaces an existing
// equal element if found. Returns the old element and true if an element was
// replaced, or the zero value of T and false if the element was inserted.
//
// When multiple equal elements exist, the first inserted element
// is replaced, maintaining stable ordering for the remaining duplicates.
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

// Has returns true if the element exists in the BWArr, false otherwise.
// The search operation has O(log N) time complexity.
func (bwa *BWArr[T]) Has(element T) bool {
	if _, index := bwa.search(element); index >= 0 {
		return true
	}
	return false
}

// Get returns the element equal to the given element and true if found,
// or the zero value of T and false if not found. The search operation has
// O(log N) time complexity.
//
// When multiple equal elements exist, the first inserted element
// is returned.
func (bwa *BWArr[T]) Get(element T) (res T, found bool) {
	if segNum, index := bwa.search(element); index >= 0 {
		return bwa.whiteSegments[segNum].elements[index], true
	}
	return
}

// Delete removes an element from the BWArr and returns it along with true
// if found, or the zero value of T and false if not found. The operation has
// O(log N) amortized time complexity.
//
// When multiple equal elements exist, the first inserted element (FIFO order)
// is deleted. Elements are marked as deleted using lazy deletion, and segments
// are consolidated when their occupancy falls below 50%.
func (bwa *BWArr[T]) Delete(element T) (deleted T, found bool) {
	segNum, index := bwa.search(element)
	if segNum < 0 {
		return deleted, false
	}
	return bwa.del(segNum, index), true
}

// DeleteMax removes and returns the maximum element in the BWArr and true,
// or the zero value of T and false if the BWArr is empty. The operation has
// O(log N) amortized time complexity.
//
// This method is useful for implementing priority queues. When multiple equal
// maximum elements exist, the first inserted element (FIFO order) is removed.
func (bwa *BWArr[T]) DeleteMax() (deleted T, found bool) {
	if bwa.total == 0 {
		return deleted, false
	}
	seg, ind := bwa.max()
	return bwa.del(seg, ind), true
}

// DeleteMin removes and returns the minimum element in the BWArr and true,
// or the zero value of T and false if the BWArr is empty. The operation has
// O(log N) amortized time complexity.
//
// This method is useful for implementing priority queues. When multiple equal
// minimum elements exist, the first inserted element (FIFO order) is removed.
func (bwa *BWArr[T]) DeleteMin() (deleted T, found bool) {
	if bwa.total == 0 {
		return deleted, false
	}
	seg, ind := bwa.min()
	return bwa.del(seg, ind), true
}

// Len returns the number of elements currently stored in the BWArr,
// excluding deleted elements. The operation has O(log N) time complexity
// as it counts non-deleted elements across all segments.
func (bwa *BWArr[T]) Len() int {
	deleted := 0
	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) != 0 {
			deleted += bwa.whiteSegments[i].deletedNum
		}
	}
	return int(bwa.total) - deleted
}

// Max returns the maximum element in the BWArr and true, or the zero value
// of T and false if the BWArr is empty. The operation has O(log N) time
// complexity in the worst case.
//
// When multiple equal maximum elements exist, the first inserted element is returned.
func (bwa *BWArr[T]) Max() (maxElem T, found bool) {
	if bwa.total == 0 {
		return maxElem, false
	}

	seg, ind := bwa.max()
	return bwa.whiteSegments[seg].elements[ind], true
}

// Min returns the minimum element in the BWArr and true, or the zero value
// of T and false if the BWArr is empty. The operation has O(log N) time
// complexity in the worst case.
//
// When multiple equal minimum elements exist, the first inserted element is returned.
func (bwa *BWArr[T]) Min() (minElem T, found bool) {
	if bwa.total == 0 {
		return minElem, false
	}
	seg, ind := bwa.min()
	return bwa.whiteSegments[seg].elements[ind], true
}

// Clear removes all elements from the BWArr. If dropSegments is true,
// all internal memory is released; if false, internal segments are retained
// for reuse, which is more efficient if the BWArr will be repopulated.
func (bwa *BWArr[T]) Clear(dropSegments bool) {
	bwa.total = 0
	if dropSegments {
		bwa.whiteSegments = bwa.whiteSegments[:0]
	}
}

// Clone creates a deep copy of the BWArr. The new BWArr is completely
// independent and modifications to it will not affect the original.
// The operation has O(N) time and space complexity.
func (bwa *BWArr[T]) Clone() *BWArr[T] {
	newBWA := &BWArr[T]{
		whiteSegments: make([]segment[T], len(bwa.whiteSegments)),
		total:         bwa.total,
		cmp:           bwa.cmp,
	}

	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) != 0 {
			newBWA.whiteSegments[i] = bwa.whiteSegments[i].deepCopy()
		}
	}
	return newBWA
}

// Ascend calls the iterator function for each element in the BWArr in
// ascending order. Iteration stops early if the iterator returns false.
// The operation visits all elements in O(N*Log(N)) time.
func (bwa *BWArr[T]) Ascend(iterator IteratorFunc[T]) {
	iter := createAscIteratorBegin(bwa)
	for val, ok := iter.next(); ok; val, ok = iter.next() {
		if !iterator(*val) {
			break
		}
	}
}

// AscendGreaterOrEqual calls the iterator function for each element in the
// BWArr that is greater than or equal to the given element, in ascending order.
// Iteration stops early if the iterator returns false. The operation has O(N*Log(N)
// time complexity in the worst case.
func (bwa *BWArr[T]) AscendGreaterOrEqual(elem T, iterator IteratorFunc[T]) {
	iter := createAscIteratorGTOE(bwa, elem)
	for val, ok := iter.next(); ok; val, ok = iter.next() {
		if !iterator(*val) {
			break
		}
	}
}

// AscendLessThan calls the iterator function for each element in the BWArr
// that is less than the given element, in ascending order. Iteration stops
// early if the iterator returns false. The operation has O(N*Log(N)) time complexity
// in the worst case.
func (bwa *BWArr[T]) AscendLessThan(elem T, iterator IteratorFunc[T]) {
	iter := createAscIteratorLess(bwa, elem)
	for val, ok := iter.next(); ok; val, ok = iter.next() {
		if !iterator(*val) {
			break
		}
	}
}

// AscendRange calls the iterator function for each element in the BWArr
// that is greater than or equal to greaterOrEqual and less than lessThan,
// in ascending order. Iteration stops early if the iterator returns false.
// The operation has O(N*Log(N)) time complexity in the worst case.
func (bwa *BWArr[T]) AscendRange(greaterOrEqual, lessThan T, iterator IteratorFunc[T]) {
	iter := createAscIteratorFromTo(bwa, greaterOrEqual, lessThan)
	for val, ok := iter.next(); ok; val, ok = iter.next() {
		if !iterator(*val) {
			break
		}
	}
}

// Descend calls the iterator function for each element in the BWArr in
// descending order. Iteration stops early if the iterator returns false.
// The operation visits all elements in O(N*Log(N)) time.
func (bwa *BWArr[T]) Descend(iterator IteratorFunc[T]) {
	iter := createDescIteratorEnd(bwa)
	for val, ok := iter.prev(); ok; val, ok = iter.prev() {
		if !iterator(*val) {
			break
		}
	}
}

// DescendGreaterOrEqual calls the iterator function for each element in the
// BWArr that is greater than or equal to the given element, in descending order.
// Iteration stops early if the iterator returns false. The operation has O(N*Log(N))
// time complexity in the worst case.
func (bwa *BWArr[T]) DescendGreaterOrEqual(elem T, iterator IteratorFunc[T]) {
	iter := createDescIteratorGTOE(bwa, elem)
	for val, ok := iter.prev(); ok; val, ok = iter.prev() {
		if !iterator(*val) {
			break
		}
	}
}

// DescendLessThan calls the iterator function for each element in the BWArr
// that is less than the given element, in descending order. Iteration stops
// early if the iterator returns false. The operation has O(N*Log(N)) time complexity
// in the worst case.
func (bwa *BWArr[T]) DescendLessThan(elem T, iterator IteratorFunc[T]) {
	iter := createDescIteratorLess(bwa, elem)
	for val, ok := iter.prev(); ok; val, ok = iter.prev() {
		if !iterator(*val) {
			break
		}
	}
}

// DescendRange calls the iterator function for each element in the BWArr
// that is greater than or equal to greaterOrEqual and less than lessThan,
// in descending order. Iteration stops early if the iterator returns false.
// The operation has O(N*Log(N)) time complexity in the worst case.
func (bwa *BWArr[T]) DescendRange(greaterOrEqual, lessThan T, iterator IteratorFunc[T]) {
	iter := createDescIteratorFromTo(bwa, greaterOrEqual, lessThan)
	for val, ok := iter.prev(); ok; val, ok = iter.prev() {
		if !iterator(*val) {
			break
		}
	}
}

// UnorderedWalk calls the iterator function for each element in the BWArr
// in an arbitrary order (not necessarily sorted). This method is faster than
// ordered iteration and should be used when element ordering is not required.
// Iteration stops early if the iterator returns false. The operation visits
// all elements in O(N) time.
func (bwa *BWArr[T]) UnorderedWalk(iterator IteratorFunc[T]) {
	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) == 0 {
			continue
		}
		seg := &bwa.whiteSegments[i]
		for j := range seg.elements {
			if seg.deleted[j] {
				continue
			}
			if !iterator(seg.elements[j]) {
				return
			}
		}
	}
}

// Compact releases memory used by inactive segments and lazy-deleted elements.
// This can improve memory usage and iteration performance when many deletions
// have occurred. The operation is typically not needed as the BWArr manages
// memory automatically, but can be useful after large numbers of deletions.
func (bwa *BWArr[T]) Compact() {
	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) == 0 { // Segment is not used
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
	halfSegmentCapacity := segmentCapacity >> 1
	if seg.deletedNum < halfSegmentCapacity {
		return deleted
	}
	if segNum == 0 {
		bwa.total--
		seg.deletedNum, seg.minNonDeletedIdx, seg.maxNonDeletedIdx = 0, 0, len(seg.elements)-1
		seg.deleted[0] = false
		return deleted
	}
	if uint64(halfSegmentCapacity)&bwa.total == 0 {
		bwa.ensureSeg(segNum - 1)
		demoteSegment(*seg, &bwa.whiteSegments[segNum-1])
	} else {
		demoteSegment1(*seg)
		mergeSegments1(&bwa.whiteSegments[segNum-1], seg, bwa.cmp, halfSegmentCapacity)
		seg.deletedNum = bwa.whiteSegments[segNum-1].deletedNum
	}
	bwa.total -= uint64(halfSegmentCapacity)
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
