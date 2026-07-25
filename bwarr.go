// Package bwarr implements a Black-White Array, a fast, ordered
// data structure with O(log N) memory allocations and O(log N) amortized complexity for
// insert, delete, and search operations. Can store equal elements and maintains stable ordering.
// See data structure details at: https://arxiv.org/abs/2004.09051
package bwarr

import (
	"math/bits"
)

const defaultMaxSegmentRank = 2

// BWArr is a Black-White Array, a fast, ordered data structure with O(log N) memory allocations
// and O(log N) amortized complexity for insert, delete, and search operations. Can store equal
// elements and maintains stable ordering.
// See data structure details at: https://arxiv.org/abs/2004.09051
type BWArr[T any] struct {
	// Data invariants for equal elements to maintain stable (FIFO) ordering and O(Log(N)) search complexity:
	// 1. If equal elements are in the same segment, older is righter (greater index).
	// 2. If equal elements are in different segments, older is placed in the higher-rank segment.
	// 3. If segment contains equal deleted and non-deleted elements, deleted are placed after non-deleted (greater index).

	whiteSegments        []segment[T]
	total                int // Total number of elements in the array, including deleted ones.
	deletedTotal         int // Number of lazily-deleted elements among the active segments (subset of total).
	cmp                  CmpFunc[T]
	maxSegmentRankToKeep int // Always keep segments with rank <= maxSegmentRankToKeep
	// If maxSegmentRankToKeep is 10 the structure will never shrink below 2047 elements.
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
	return NewWithOptions[T](cmp, capacity, Options{ElementsKeepAllocated: 1 << defaultMaxSegmentRank})
}

type Options struct {
	// Number of elements to keep allocated in segments after deletion to prevent allocations on smaller sizes.
	// Rounded down to a power of two: for example, if set to 10, segments of up to 8 elements stay allocated.
	ElementsKeepAllocated uint64
}

// NewWithOptions creates a new empty BWArr with the given comparison function CmpFunc, capacity hint, and Options.
// See Options struct for details on available options.
func NewWithOptions[T any](cmp CmpFunc[T], capacity int, options Options) *BWArr[T] {
	maxSegmentRankToKeep := bits.Len64(options.ElementsKeepAllocated) - 1 //nolint: gosec
	bwa := &BWArr[T]{cmp: cmp, total: 0, maxSegmentRankToKeep: maxSegmentRankToKeep}

	wSegNum := calculateWhiteSegmentsQuantity(capacity)
	if wSegNum > 0 {
		bwa.whiteSegments = createSegments[T](0, wSegNum)
	}
	return bwa
}

// Insert adds an element to the BWArr maintaining sorted order.
// The operation has O(log N) amortized time complexity. Note that one in
// every N insert operations may take O(N) time for segment consolidation.
//
// Duplicate elements are allowed. If multiple equal elements exist, they
// maintain stable ordering based on insertion order.
func (bwa *BWArr[T]) Insert(element T) {
	// bwa.total + 1 - the new total number of elements after insertion, including the new element.
	// & -(bwa.total + 1)  bit trick to get  the lowest set bit - segment that will become active after insertion.
	destSegSize := (bwa.total + 1) & -(bwa.total + 1)
	destSegRank := rightmostTrueBitPosition(destSegSize)
	bwa.ensureSeg(destSegRank)
	destSeg := &bwa.whiteSegments[destSegRank]

	// Segments deactivate without cleanup, so a reused segment may carry stale deleted state - drop it.
	if destSeg.deletedNum != 0 {
		destSeg.resetDeleted()
	}

	// Put the new element at the end of the destination segment
	destSeg.elements[destSegSize-1] = element

	destReadPtr := destSegSize - 1
	for segmentNumber := range destSegRank {
		mergeSegments(&bwa.whiteSegments[segmentNumber], destSeg, bwa.cmp, destReadPtr)
		destReadPtr -= 1 << segmentNumber
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
// excluding deleted elements. The operation has O(1) time complexity.
func (bwa *BWArr[T]) Len() int {
	return bwa.total - bwa.deletedTotal
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
	bwa.deletedTotal = 0
	if dropSegments {
		bwa.whiteSegments = nil
	}
}

// Clone creates a deep copy of the BWArr. The new BWArr is completely
// independent and modifications to it will not affect the original.
// The operation has O(N) time and space complexity.
func (bwa *BWArr[T]) Clone() *BWArr[T] {
	newBWA := &BWArr[T]{
		whiteSegments: make([]segment[T], len(bwa.whiteSegments)),
		total:         bwa.total,
		deletedTotal:  bwa.deletedTotal,
		cmp:           bwa.cmp,
	}

	for i := range bwa.activeSegments {
		newBWA.whiteSegments[i] = bwa.whiteSegments[i].deepCopy()
	}
	return newBWA
}

// Ascend calls the iterator function for each element in the BWArr in
// ascending order. Iteration stops early if the iterator returns false.
// The operation visits all elements in O(N*Log(N)) time.
func (bwa *BWArr[T]) Ascend(iterator IteratorFunc[T]) {
	walkAsc(createAscIteratorBegin(bwa), iterator)
}

// AscendGreaterOrEqual calls the iterator function for each element in the
// BWArr that is greater than or equal to the given element, in ascending order.
// Iteration stops early if the iterator returns false. The operation has O(N*Log(N)
// time complexity in the worst case.
func (bwa *BWArr[T]) AscendGreaterOrEqual(elem T, iterator IteratorFunc[T]) {
	walkAsc(createAscIteratorGTOE(bwa, elem), iterator)
}

// AscendLessThan calls the iterator function for each element in the BWArr
// that is less than the given element, in ascending order. Iteration stops
// early if the iterator returns false. The operation has O(N*Log(N)) time complexity
// in the worst case.
func (bwa *BWArr[T]) AscendLessThan(elem T, iterator IteratorFunc[T]) {
	walkAsc(createAscIteratorLess(bwa, elem), iterator)
}

// AscendRange calls the iterator function for each element in the BWArr
// that is greater than or equal to greaterOrEqual and less than lessThan,
// in ascending order. Iteration stops early if the iterator returns false.
// The operation has O(N*Log(N)) time complexity in the worst case.
func (bwa *BWArr[T]) AscendRange(greaterOrEqual, lessThan T, iterator IteratorFunc[T]) {
	walkAsc(createAscIteratorFromTo(bwa, greaterOrEqual, lessThan), iterator)
}

// Descend calls the iterator function for each element in the BWArr in
// descending order. Iteration stops early if the iterator returns false.
// The operation visits all elements in O(N*Log(N)) time.
func (bwa *BWArr[T]) Descend(iterator IteratorFunc[T]) {
	walkDesc(createDescIteratorEnd(bwa), iterator)
}

// DescendGreaterOrEqual calls the iterator function for each element in the
// BWArr that is greater than or equal to the given element, in descending order.
// Iteration stops early if the iterator returns false. The operation has O(N*Log(N))
// time complexity in the worst case.
func (bwa *BWArr[T]) DescendGreaterOrEqual(elem T, iterator IteratorFunc[T]) {
	walkDesc(createDescIteratorGTOE(bwa, elem), iterator)
}

// DescendLessThan calls the iterator function for each element in the BWArr
// that is less than the given element, in descending order. Iteration stops
// early if the iterator returns false. The operation has O(N*Log(N)) time complexity
// in the worst case.
func (bwa *BWArr[T]) DescendLessThan(elem T, iterator IteratorFunc[T]) {
	walkDesc(createDescIteratorLess(bwa, elem), iterator)
}

// DescendRange calls the iterator function for each element in the BWArr
// that is greater than or equal to greaterOrEqual and less than lessThan,
// in descending order. Iteration stops early if the iterator returns false.
// The operation has O(N*Log(N)) time complexity in the worst case.
func (bwa *BWArr[T]) DescendRange(greaterOrEqual, lessThan T, iterator IteratorFunc[T]) {
	walkDesc(createDescIteratorFromTo(bwa, greaterOrEqual, lessThan), iterator)
}

// UnorderedWalk calls the iterator function for each element in the BWArr
// in an arbitrary order (not necessarily sorted). This method is faster than
// ordered iteration and should be used when element ordering is not required.
// Iteration stops early if the iterator returns false. The operation visits
// all elements in O(N) time.
func (bwa *BWArr[T]) UnorderedWalk(iterator IteratorFunc[T]) {
	for i := range bwa.activeSegments {
		if !bwa.walkSegment(i, iterator) {
			return
		}
	}
}

// walkSegment feeds the live elements of one segment to iterator in index order, and reports whether
// the iterator wants more. The per-element loop lives here, not inside the range-over-func body of
// UnorderedWalk: a body closure around it costs ~20% on a full walk.
func (bwa *BWArr[T]) walkSegment(rank int, iterator IteratorFunc[T]) bool {
	seg := &bwa.whiteSegments[rank]
	elems := seg.elements
	for w, word := range seg.deleted.layers[0] {
		base := w << wordShift
		// Zero bits are live elements; bits beyond len(elems) in the last word are phantom zeros.
		for live := ^word; live != 0; live &= live - 1 {
			j := base + bits.TrailingZeros64(live)
			if j >= len(elems) {
				break
			}
			if !iterator(elems[j]) {
				return false
			}
		}
	}
	return true
}

// Compact releases memory used by inactive segments and lazy-deleted elements.
// This can improve memory usage and iteration performance when many deletions
// have occurred. The operation is typically not needed as the BWArr manages
// memory automatically, but can be useful after large numbers of deletions.
func (bwa *BWArr[T]) Compact() {
	for i := range bwa.whiteSegments {
		if !bwa.active(i) {
			bwa.whiteSegments[i] = segment[T]{} //nolint:exhaustruct
		}
	}
}

func (bwa *BWArr[T]) del(segNum, index int) (deleted T) {
	seg := &bwa.whiteSegments[segNum]
	deleted = seg.elements[index]
	seg.deleted.Set(index)
	seg.deletedNum++
	bwa.deletedTotal++

	segmentCapacity := 1 << segNum
	halfSegmentCapacity := segmentCapacity >> 1
	if seg.deletedNum < halfSegmentCapacity {
		return deleted
	}
	if segNum == 0 {
		bwa.total--
		bwa.deletedTotal--
		seg.deletedNum = 0
		seg.deleted.Reset()
		return deleted
	}
	if !bwa.active(segNum - 1) { // Lower neighbor is free - demote into it; otherwise merge with it.
		bwa.ensureSeg(segNum - 1)
		demoteSegment(seg, &bwa.whiteSegments[segNum-1])
		if bwa.maxRank() == segNum && segNum > bwa.maxSegmentRankToKeep {
			bwa.whiteSegments[segNum] = segment[T]{} //nolint:exhaustruct
		}
	} else {
		moveNonDeletedValuesToSegmentEnd(seg)
		// The lower-rank segment holds the newer elements (FIFO invariant), so lowSegIsNewer=true.
		mergeSegmentsDirty(&bwa.whiteSegments[segNum-1], seg, bwa.cmp, halfSegmentCapacity, true)
		seg.deletedNum = bwa.whiteSegments[segNum-1].deletedNum
	}
	// Both consolidation paths remove exactly half a segment's worth of deleted elements from the accounting.
	bwa.total -= halfSegmentCapacity
	bwa.deletedTotal -= halfSegmentCapacity
	return deleted
}

// min assumes that there is at least one segment with elements!
func (bwa *BWArr[T]) min() (segNum, index int) { //nolint:dupl
	segNum, index = -1, -1
	// Find the segment with the smallest element:
	for seg := range bwa.activeSegments {
		// Less or equal is used to provide stable behavior (return the oldest one):
		// the ranks come in increasing order and the higher the rank, the older the elements.
		ind := bwa.whiteSegments[seg].min(bwa.cmp)
		if segNum < 0 || bwa.cmp(bwa.whiteSegments[seg].elements[ind], bwa.whiteSegments[segNum].elements[index]) <= 0 {
			segNum, index = seg, ind
		}
	}
	return segNum, index
}

// max assumes that there is at least one segment with elements!
func (bwa *BWArr[T]) max() (segNum, index int) { //nolint:dupl
	segNum, index = -1, -1
	// Find the segment with the largest element:
	for seg := range bwa.activeSegments {
		// Greater or equal is used to provide stable behavior (return the oldest one):
		// the ranks come in increasing order and the higher the rank, the older the elements.
		ind := bwa.whiteSegments[seg].maxNonDeletedIndex()
		if segNum < 0 || bwa.cmp(bwa.whiteSegments[seg].elements[ind], bwa.whiteSegments[segNum].elements[index]) >= 0 {
			segNum, index = seg, ind
		}
	}
	return segNum, index
}

func (bwa *BWArr[T]) search(element T) (segNum, index int) {
	// The oldest match wins (FIFO), and the higher the rank, the older the elements.
	for seg := range bwa.activeSegmentsDesc {
		if index = bwa.whiteSegments[seg].findRightmostNotDeleted(bwa.cmp, element); index >= 0 {
			return seg, index
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

// active reports whether the segment of the given rank currently holds data:
// rank r is active iff bit r of total is set (the paper's active(i) predicate).
func (bwa *BWArr[T]) active(rank int) bool {
	return bwa.total&(1<<rank) != 0
}

// activeSegments yields the ranks of the segments that currently hold data, from the lowest rank
// (the newest elements) to the highest (the oldest). Rank r is active iff bit r of total is set, so
// the ranks to visit are the set bits of the element count: the walk takes one step per segment that
// holds data, not one per allocated segment.
//
//	for rank := range bwa.activeSegments { ... }
func (bwa *BWArr[T]) activeSegments(yield func(rank int) bool) {
	for m := uint64(bwa.total); m != 0; m &= m - 1 { //nolint: gosec // total is always non-negative.
		if !yield(bits.TrailingZeros64(m)) {
			return
		}
	}
}

// activeSegmentsDesc is activeSegments in the opposite direction: highest rank, holding the oldest
// elements, first.
func (bwa *BWArr[T]) activeSegmentsDesc(yield func(rank int) bool) {
	for m := uint64(bwa.total); m != 0; { //nolint: gosec // total is always non-negative.
		rank := bits.Len64(m) - 1
		m &= 1<<rank - 1 // Visited: only the lower ranks are left.
		if !yield(rank) {
			return
		}
	}
}

func (bwa *BWArr[T]) maxRank() int {
	return bits.Len64(uint64(bwa.total)) - 1 //nolint: gosec // x is always non-negative, so it is safe to convert it to uint64.
}
