package bwarr

import (
	"slices"
)

type iterator[T any] struct {
	segIters []*segmentIterator[T]
	cmp      CmpFunc[T]
}

type segmentIterator[T any] struct {
	seg   segment[T]
	index int
}

func (t *segmentIterator[T]) curVal() (val *T, last bool) {
	return &t.seg.elements[t.index], t.index == len(t.seg.elements)-1
}

func createIterator[T any](bwa *BWArr[T]) iterator[T] {
	iter := iterator[T]{
		segIters: make([]*segmentIterator[T], 0, len(bwa.whiteSegments)),
		cmp:      bwa.cmp,
	}

	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) == 0 {
			continue
		}
		idx := bwa.whiteSegments[i].minNonDeletedIndex()
		iter.segIters = append(iter.segIters, &segmentIterator[T]{index: idx, seg: bwa.whiteSegments[i]})
	}

	slices.SortFunc(iter.segIters, func(s1, s2 *segmentIterator[T]) int {
		return iter.cmp(s1.seg.elements[s1.index], s2.seg.elements[s2.index])
	})

	return iter
}

func (iter *iterator[T]) cmpSegIters(i, j int) int {
	s1, s2 := iter.segIters[i], iter.segIters[j]
	return iter.cmp(s1.seg.elements[s1.index], s2.seg.elements[s2.index])
}

func (iter *iterator[T]) next() (*T, bool) {
	if len(iter.segIters) == 0 {
		return nil, false
	}

	res, last := iter.segIters[0].curVal()
	if last {
		iter.segIters = iter.segIters[1:]
		return res, true
	}
	iter.segIters[0].index++

	if len(iter.segIters) == 1 {
		return res, true
	}

	if iter.cmpSegIters(0, 1) <= 0 {
		return res, true
	}

	insPos := 1
	for i := 2; i < len(iter.segIters); i++ {
		if iter.cmpSegIters(0, i) > 0 {
			insPos = i
			break
		}
	}

	v := iter.segIters[0]
	copy(iter.segIters, iter.segIters[1:insPos+1])
	iter.segIters[insPos] = v

	return res, true
}
