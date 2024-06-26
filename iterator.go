package bwarr

type iterator[T any] struct {
	segIters   []segmentIterator[T]
	curValPtrs []*T
	cmp        CmpFunc[T]
}

type segmentIterator[T any] struct {
	seg   segment[T]
	index int
}

func createIterator[T any](bwa *BWArr[T]) iterator[T] {
	iter := iterator[T]{
		segIters:   make([]segmentIterator[T], 0, len(bwa.whiteSegments)),
		curValPtrs: make([]*T, 0, len(bwa.whiteSegments)),
		cmp:        bwa.cmp,
	}

	for i := range bwa.whiteSegments {
		if bwa.total&(1<<i) == 0 {
			continue
		}
		idx := bwa.whiteSegments[i].minNonDeletedIndex()

		iter.segIters = append(iter.segIters, segmentIterator[T]{index: idx, seg: bwa.whiteSegments[i]})
		iter.curValPtrs = append(iter.curValPtrs, &bwa.whiteSegments[i].elements[idx])
	}

	// TODO: uncomment this after writing benchmarks to compare find min ws sorted.
	//sort.Slice(iter.curValPtrs, func(i, j int) bool {
	//	return bwa.cmp(*iter.curValPtrs[i], *iter.curValPtrs[j]) < 0
	//})

	return iter
}

func (iter *iterator[T]) next() (*T, bool) {
	if len(iter.segIters) == 0 {
		return nil, false
	}

	minIdx := len(iter.segIters) - 1
	for i := len(iter.segIters) - 2; i >= 0; i-- {
		if iter.cmp(*iter.curValPtrs[minIdx], *iter.curValPtrs[i]) > 0 {
			minIdx = i
		}
	}
	res := iter.curValPtrs[minIdx]

	if iter.segIters[minIdx].index < len(iter.segIters[minIdx].seg.elements)-1 {
		iter.segIters[minIdx].index++
		idx := iter.segIters[minIdx].index
		iter.curValPtrs[minIdx] = &iter.segIters[minIdx].seg.elements[idx]
		return res, true
	}

	// Segment with the minimum value has been fully iterated:
	copy(iter.segIters[minIdx:], iter.segIters[minIdx+1:])
	iter.segIters = iter.segIters[:len(iter.segIters)-1]
	copy(iter.curValPtrs[minIdx:], iter.curValPtrs[minIdx+1:])
	iter.curValPtrs = iter.curValPtrs[:len(iter.curValPtrs)-1]
	return res, true
}
