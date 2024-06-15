package bwarr

import "sort"

type iterator[T any] struct {
	segIters   []segmentIterator[T]
	curValPtrs []*T
}

type segmentIterator[T any] struct {
	seg   segment[T]
	index int
}

func createIterator[T any](bwa *BWArr[T]) iterator[T] {
	iter := iterator[T]{
		segIters:   make([]segmentIterator[T], 0, len(bwa.whiteSegments)),
		curValPtrs: make([]*T, 0, len(bwa.whiteSegments)),
	}

	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) == 0 {
			continue
		}
		idx := bwa.whiteSegments[i].minNonDeletedIndex()

		iter.segIters = append(iter.segIters, segmentIterator[T]{index: idx, seg: bwa.whiteSegments[i]})
		iter.curValPtrs = append(iter.curValPtrs, &bwa.whiteSegments[i].elements[idx])
	}

	sort.Slice(iter.curValPtrs, func(i, j int) bool {
		return bwa.cmp(*iter.curValPtrs[i], *iter.curValPtrs[j]) < 0
	})

	return iter
}

func (iter *iterator[T]) next() (*T, bool) {
	panic("not implemented")
}
