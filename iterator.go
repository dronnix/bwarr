package bwarr

import (
	"slices"
)

type iterator[T any] struct {
	segIters []*segmentIterator[T] // Pointers used to be able to pass pointers to segmentIterators to SortFunc.
	cmp      CmpFunc[T]
}

type segmentIterator[T any] struct {
	seg   segment[T]
	index int
	end   int
}

func createAscIteratorBegin[T any](bwa *BWArr[T]) iterator[T] { //nolint:dupl
	iter := iterator[T]{
		segIters: make([]*segmentIterator[T], 0, len(bwa.whiteSegments)),
		cmp:      bwa.cmp,
	}

	si := make([]segmentIterator[T], len(bwa.whiteSegments))
	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) == 0 {
			continue
		}
		idx := bwa.whiteSegments[i].minNonDeletedIndex()
		end := bwa.whiteSegments[i].maxNonDeletedIndex()
		si[i] = segmentIterator[T]{index: idx, seg: bwa.whiteSegments[i], end: end}
		iter.segIters = append(iter.segIters, &si[i])
	}

	slices.SortFunc(iter.segIters, func(s1, s2 *segmentIterator[T]) int {
		return iter.cmp(s1.seg.elements[s1.index], s2.seg.elements[s2.index])
	})

	return iter
}

func createAscIteratorGTOE[T any](bwa *BWArr[T], elem T) iterator[T] { //nolint:dupl
	iter := iterator[T]{
		segIters: make([]*segmentIterator[T], 0, len(bwa.whiteSegments)),
		cmp:      bwa.cmp,
	}

	si := make([]segmentIterator[T], len(bwa.whiteSegments))
	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) == 0 {
			continue
		}
		idx := bwa.whiteSegments[i].findGTOE(bwa.cmp, elem)
		if idx < 0 {
			continue
		}
		end := bwa.whiteSegments[i].maxNonDeletedIndex()
		si[i] = segmentIterator[T]{index: idx, seg: bwa.whiteSegments[i], end: end}
		iter.segIters = append(iter.segIters, &si[i])
	}

	slices.SortFunc(iter.segIters, func(s1, s2 *segmentIterator[T]) int {
		return iter.cmp(s1.seg.elements[s1.index], s2.seg.elements[s2.index])
	})

	return iter
}

func createAscIteratorLess[T any](bwa *BWArr[T], elem T) iterator[T] { //nolint:dupl
	iter := iterator[T]{
		segIters: make([]*segmentIterator[T], 0, len(bwa.whiteSegments)),
		cmp:      bwa.cmp,
	}

	si := make([]segmentIterator[T], len(bwa.whiteSegments))
	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) == 0 {
			continue
		}
		end := bwa.whiteSegments[i].findLess(bwa.cmp, elem)
		if end < 0 {
			continue
		}
		idx := bwa.whiteSegments[i].minNonDeletedIndex()
		si[i] = segmentIterator[T]{index: idx, seg: bwa.whiteSegments[i], end: end}
		iter.segIters = append(iter.segIters, &si[i])
	}

	slices.SortFunc(iter.segIters, func(s1, s2 *segmentIterator[T]) int {
		return iter.cmp(s1.seg.elements[s1.index], s2.seg.elements[s2.index])
	})

	return iter
}

func createAscIteratorFromTo[T any](bwa *BWArr[T], from, to T) iterator[T] { //nolint:dupl
	iter := iterator[T]{
		segIters: make([]*segmentIterator[T], 0, len(bwa.whiteSegments)),
		cmp:      bwa.cmp,
	}

	si := make([]segmentIterator[T], len(bwa.whiteSegments))
	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) == 0 {
			continue
		}
		end := bwa.whiteSegments[i].findLess(bwa.cmp, to)
		if end < 0 {
			continue
		}
		begin := bwa.whiteSegments[i].findGTOE(bwa.cmp, from)
		if begin < 0 {
			continue
		}

		si[i] = segmentIterator[T]{index: begin, seg: bwa.whiteSegments[i], end: end}
		iter.segIters = append(iter.segIters, &si[i])
	}

	slices.SortFunc(iter.segIters, func(s1, s2 *segmentIterator[T]) int {
		return iter.cmp(s1.seg.elements[s1.index], s2.seg.elements[s2.index])
	})

	return iter
}

func createDescIteratorEnd[T any](bwa *BWArr[T]) iterator[T] { //nolint:dupl
	iter := iterator[T]{
		segIters: make([]*segmentIterator[T], 0, len(bwa.whiteSegments)),
		cmp:      bwa.cmp,
	}

	si := make([]segmentIterator[T], len(bwa.whiteSegments))
	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) == 0 {
			continue
		}
		idx := bwa.whiteSegments[i].maxNonDeletedIndex()
		end := bwa.whiteSegments[i].minNonDeletedIndex()
		si[i] = segmentIterator[T]{index: idx, seg: bwa.whiteSegments[i], end: end}
		iter.segIters = append(iter.segIters, &si[i])
	}

	slices.SortFunc(iter.segIters, func(s1, s2 *segmentIterator[T]) int {
		return iter.cmp(s2.seg.elements[s2.index], s1.seg.elements[s1.index])
	})

	return iter
}

func createDescIteratorGTOE[T any](bwa *BWArr[T], elem T) iterator[T] { //nolint:dupl
	iter := iterator[T]{
		segIters: make([]*segmentIterator[T], 0, len(bwa.whiteSegments)),
		cmp:      bwa.cmp,
	}

	si := make([]segmentIterator[T], len(bwa.whiteSegments))
	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) == 0 {
			continue
		}
		end := bwa.whiteSegments[i].findGTOE(bwa.cmp, elem)
		if end < 0 {
			continue
		}
		idx := bwa.whiteSegments[i].maxNonDeletedIndex()
		si[i] = segmentIterator[T]{index: idx, seg: bwa.whiteSegments[i], end: end}
		iter.segIters = append(iter.segIters, &si[i])
	}

	slices.SortFunc(iter.segIters, func(s1, s2 *segmentIterator[T]) int {
		return iter.cmp(s2.seg.elements[s2.index], s1.seg.elements[s1.index])
	})

	return iter
}

func createDescIteratorLess[T any](bwa *BWArr[T], elem T) iterator[T] {
	iter := iterator[T]{
		segIters: make([]*segmentIterator[T], 0, len(bwa.whiteSegments)),
		cmp:      bwa.cmp,
	}

	si := make([]segmentIterator[T], len(bwa.whiteSegments))
	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) == 0 {
			continue
		}
		idx := bwa.whiteSegments[i].findLess(bwa.cmp, elem)
		if idx < 0 {
			continue
		}
		end := bwa.whiteSegments[i].minNonDeletedIdx
		si[i] = segmentIterator[T]{index: idx, seg: bwa.whiteSegments[i], end: end}
		iter.segIters = append(iter.segIters, &si[i])
	}

	slices.SortFunc(iter.segIters, func(s1, s2 *segmentIterator[T]) int {
		return iter.cmp(s2.seg.elements[s2.index], s1.seg.elements[s1.index])
	})

	return iter
}

func createDescIteratorFromTo[T any](bwa *BWArr[T], from, to T) iterator[T] { //nolint:dupl
	iter := iterator[T]{
		segIters: make([]*segmentIterator[T], 0, len(bwa.whiteSegments)),
		cmp:      bwa.cmp,
	}

	si := make([]segmentIterator[T], len(bwa.whiteSegments))
	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) == 0 {
			continue
		}
		end := bwa.whiteSegments[i].findGTOE(bwa.cmp, from)
		if end < 0 {
			continue
		}
		begin := bwa.whiteSegments[i].findLess(bwa.cmp, to)
		if begin < 0 {
			continue
		}

		si[i] = segmentIterator[T]{index: begin, seg: bwa.whiteSegments[i], end: end}
		iter.segIters = append(iter.segIters, &si[i])
	}

	slices.SortFunc(iter.segIters, func(s1, s2 *segmentIterator[T]) int {
		return iter.cmp(s2.seg.elements[s2.index], s1.seg.elements[s1.index])
	})

	return iter
}

func (iter *iterator[T]) next() (*T, bool) { //nolint:dupl
	if len(iter.segIters) == 0 {
		return nil, false
	}

	seg := iter.segIters[0]
	res := &seg.seg.elements[seg.index]

	// Advance the iterator to the next non-deleted element.
	if !seg.next() { // Reached the end of the segment, remove it from the list.
		iter.segIters = iter.segIters[1:]
		return res, true
	}

	if len(iter.segIters) == 1 { // Only one segment left, no need to sort.
		return res, true
	}

	// Find the position to insert advanced iterator:
	insPos := len(iter.segIters) - 1
	for i := 1; i < len(iter.segIters); i++ {
		if iter.cmpSegIters(0, i) <= 0 {
			insPos = i - 1
			break
		}
	}
	if insPos == 0 { // Advanced iterator is already in the correct position.
		return res, true
	}

	// Insert the advanced iterator in the right position:
	v := iter.segIters[0]
	copy(iter.segIters, iter.segIters[1:insPos+1])
	iter.segIters[insPos] = v
	return res, true
}

func (iter *iterator[T]) prev() (*T, bool) { //nolint:dupl
	if len(iter.segIters) == 0 {
		return nil, false
	}

	seg := iter.segIters[0]
	res := &seg.seg.elements[seg.index]

	// Advance the iterator to the previous non-deleted element.
	if !seg.prev() { // Reached the beginning of the segment, remove it from the list.
		iter.segIters = iter.segIters[1:]
		return res, true
	}

	if len(iter.segIters) == 1 { // Only one segment left, no need to sort.
		return res, true
	}

	// Find the position to insert advanced iterator:
	insPos := len(iter.segIters) - 1
	for i := 1; i < len(iter.segIters); i++ {
		if iter.cmpSegIters(0, i) >= 0 {
			insPos = i - 1
			break
		}
	}
	if insPos == 0 { // Advanced iterator is already in the correct position.
		return res, true
	}

	// Insert the advanced iterator in the right position:
	v := iter.segIters[0]
	copy(iter.segIters, iter.segIters[1:insPos+1])
	iter.segIters[insPos] = v
	return res, true
}

func (t *segmentIterator[T]) next() bool {
	idx := t.seg.nextNonDeletedAfter(t.index)
	if idx > t.end {
		return false
	}
	t.index = idx
	return true
}

func (t *segmentIterator[T]) prev() bool {
	idx := t.seg.prevNonDeletedBefore(t.index)
	if idx < t.end {
		return false
	}
	t.index = idx
	return true
}

func (iter *iterator[T]) cmpSegIters(i, j int) int {
	s1, s2 := iter.segIters[i], iter.segIters[j]
	return iter.cmp(s1.seg.elements[s1.index], s2.seg.elements[s2.index])
}
