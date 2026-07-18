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
	// Cached layer-0 word of seg.deleted covering index, and the mask of index's bit
	// within it - lets next/prev test bits in registers and touch the bitset only when
	// crossing a 64-element boundary. Maintained by nextSlow/prevSlow.
	word uint64
	mask uint64
}

// newIterator collects a segmentIterator for every active segment using the bounds
// resolver, which returns the first and last index of the segment's sub-range in
// iteration order (for descending iterators first >= last), or ok=false when the
// segment has nothing to iterate. Segment iterators are sorted by their first element.
func newIterator[T any](bwa *BWArr[T], desc bool, bounds func(s *segment[T]) (first, last int, ok bool)) iterator[T] {
	iter := iterator[T]{
		segIters: make([]*segmentIterator[T], 0, len(bwa.whiteSegments)),
		cmp:      bwa.cmp,
	}

	si := make([]segmentIterator[T], len(bwa.whiteSegments))
	for i := range bwa.whiteSegments {
		if !bwa.active(i) {
			continue
		}
		first, last, ok := bounds(&bwa.whiteSegments[i])
		if !ok {
			continue
		}
		si[i] = segmentIterator[T]{
			index: first, seg: bwa.whiteSegments[i], end: last,
			word: bwa.whiteSegments[i].deleted.layers[0][first>>wordShift],
			mask: 1 << (first & wordMask),
		}
		iter.segIters = append(iter.segIters, &si[i])
	}

	slices.SortFunc(iter.segIters, func(s1, s2 *segmentIterator[T]) int {
		if desc {
			s1, s2 = s2, s1
		}
		return iter.cmp(s1.seg.elements[s1.index], s2.seg.elements[s2.index])
	})

	return iter
}

func createAscIteratorBegin[T any](bwa *BWArr[T]) iterator[T] {
	return newIterator(bwa, false, func(s *segment[T]) (int, int, bool) {
		return s.minNonDeletedIndex(), s.maxNonDeletedIndex(), true
	})
}

func createAscIteratorGTOE[T any](bwa *BWArr[T], elem T) iterator[T] {
	return newIterator(bwa, false, func(s *segment[T]) (int, int, bool) {
		first := s.findGTOE(bwa.cmp, elem)
		return first, s.maxNonDeletedIndex(), first >= 0
	})
}

func createAscIteratorLess[T any](bwa *BWArr[T], elem T) iterator[T] {
	return newIterator(bwa, false, func(s *segment[T]) (int, int, bool) {
		last := s.findLess(bwa.cmp, elem)
		return s.minNonDeletedIndex(), last, last >= 0
	})
}

func createAscIteratorFromTo[T any](bwa *BWArr[T], from, to T) iterator[T] {
	return newIterator(bwa, false, func(s *segment[T]) (int, int, bool) {
		first := s.findGTOE(bwa.cmp, from)
		last := s.findLess(bwa.cmp, to)
		// first > last means [from, to) falls in a gap between the segment's elements - nothing to iterate.
		return first, last, first >= 0 && first <= last
	})
}

func createDescIteratorEnd[T any](bwa *BWArr[T]) iterator[T] {
	return newIterator(bwa, true, func(s *segment[T]) (int, int, bool) {
		return s.maxNonDeletedIndex(), s.minNonDeletedIndex(), true
	})
}

func createDescIteratorGTOE[T any](bwa *BWArr[T], elem T) iterator[T] {
	return newIterator(bwa, true, func(s *segment[T]) (int, int, bool) {
		last := s.findGTOE(bwa.cmp, elem)
		return s.maxNonDeletedIndex(), last, last >= 0
	})
}

func createDescIteratorLess[T any](bwa *BWArr[T], elem T) iterator[T] {
	return newIterator(bwa, true, func(s *segment[T]) (int, int, bool) {
		first := s.findLess(bwa.cmp, elem)
		return first, s.minNonDeletedIndex(), first >= 0
	})
}

func createDescIteratorFromTo[T any](bwa *BWArr[T], from, to T) iterator[T] {
	return newIterator(bwa, true, func(s *segment[T]) (int, int, bool) {
		first := s.findLess(bwa.cmp, to)
		last := s.findGTOE(bwa.cmp, from)
		// first < last means [from, to) falls in a gap between the segment's elements - nothing to iterate.
		return first, last, last >= 0 && first >= last
	})
}

// walkAsc feeds fn with the iterator's elements in ascending order until fn returns false.
func walkAsc[T any](iter iterator[T], fn IteratorFunc[T]) {
	for val, ok := iter.next(); ok; val, ok = iter.next() {
		if !fn(*val) {
			break
		}
	}
}

// walkDesc feeds fn with the iterator's elements in descending order until fn returns false.
func walkDesc[T any](iter iterator[T], fn IteratorFunc[T]) {
	for val, ok := iter.prev(); ok; val, ok = iter.prev() {
		if !fn(*val) {
			break
		}
	}
}

// next and prev are mirror-image duplicates kept separate on purpose: they are per-element hot paths.
func (iter *iterator[T]) next() (*T, bool) { //nolint:dupl
	if len(iter.segIters) == 0 {
		return nil, false
	}

	seg := iter.segIters[0]
	res := &seg.seg.elements[seg.index]

	// Advance to the segment's next live element. The fast path is inlined by hand:
	// the compiler declines to inline it as a method, and the call costs ~15% on sweeps.
	nextIdx := seg.index + 1
	m := seg.mask << 1
	advanced := m != 0 && seg.word&m == 0 && nextIdx <= seg.end
	if advanced {
		seg.index = nextIdx
		seg.mask = m
	} else {
		advanced = seg.nextSlow(nextIdx)
	}
	if !advanced { // Reached the end of the segment, remove it from the list.
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

	// Advance to the segment's previous live element. The fast path is inlined by hand:
	// the compiler declines to inline it as a method, and the call costs ~15% on sweeps.
	prevIdx := seg.index - 1
	m := seg.mask >> 1
	advanced := m != 0 && seg.word&m == 0 && prevIdx >= seg.end
	if advanced {
		seg.index = prevIdx
		seg.mask = m
	} else {
		advanced = seg.prevSlow(prevIdx)
	}
	if !advanced { // Reached the beginning of the segment, remove it from the list.
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

// nextSlow is the cold half of the segment advance; the hot half (register mask test)
// is inlined by hand into iterator.next. It handles bounds, word-boundary reload, and
// deleted skipping.
// Moves index to the first live element at or after `from`.
func (t *segmentIterator[T]) nextSlow(from int) bool {
	if from > t.end {
		return false
	}
	if from&wordMask == 0 { // Crossed a word boundary - reload the cache.
		t.word = t.seg.deleted.layers[0][from>>wordShift]
		if t.word&1 == 0 {
			t.index = from
			t.mask = 1
			return true
		}
	}
	// Short deleted runs resolve within the cached word.
	idx := from - from&wordMask // First index of the cached word.
	if bit := findNextUnsetBit(t.word, from&wordMask); bit < wordBits {
		idx += bit
	} else {
		// Long runs: the layered walk skips whole all-set words.
		idx = t.seg.deleted.FindNextUnsetBit(idx | wordMask)
		if idx < 0 {
			return false
		}
		t.word = t.seg.deleted.layers[0][idx>>wordShift]
	}
	if idx > t.end {
		return false
	}
	t.index = idx
	t.mask = 1 << (idx & wordMask)
	return true
}

// prevSlow is the cold half of the segment advance; the hot half (register mask test)
// is inlined by hand into iterator.prev. It handles bounds, word-boundary reload, and
// deleted skipping.
// Moves index to the first live element at or before `from`.
func (t *segmentIterator[T]) prevSlow(from int) bool {
	if from < t.end {
		return false
	}
	if from&wordMask == wordMask { // Crossed a word boundary - reload the cache.
		t.word = t.seg.deleted.layers[0][from>>wordShift]
		if t.word>>wordMask == 0 {
			t.index = from
			t.mask = 1 << wordMask
			return true
		}
	}
	// Short deleted runs resolve within the cached word.
	idx := from - from&wordMask // First index of the cached word.
	if bit := findPrevUnsetBit(t.word, from&wordMask); bit >= 0 {
		idx += bit
	} else {
		// Long runs: the layered walk skips whole all-set words.
		idx = t.seg.deleted.FindPrevUnsetBit(idx)
		if idx < 0 {
			return false
		}
		t.word = t.seg.deleted.layers[0][idx>>wordShift]
	}
	if idx < t.end {
		return false
	}
	t.index = idx
	t.mask = 1 << (idx & wordMask)
	return true
}

func (iter *iterator[T]) cmpSegIters(i, j int) int {
	s1, s2 := iter.segIters[i], iter.segIters[j]
	return iter.cmp(s1.seg.elements[s1.index], s2.seg.elements[s2.index])
}
