package bwarr

import "math"

type BWArr[T any] struct {
	blackSegments []segment[T]
	whiteSegments []segment[T]
	total         int // Total number of elements in the array, including deleted ones.
	cmp           CmpFunc[T]
}

type segment[T any] struct {
	elements   []T    // Stores user's data.
	deleted    []bool // Stores whether i-th element is deleted.
	deletedNum int    // Number of deleted elements in the segment.
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

func (bwa *BWArr[T]) Append(element T) {
	if bwa.total&1 == 0 { // whiteSegments[0] is free
		bwa.whiteSegments[0].elements[0] = element
		bwa.whiteSegments[0].deleted[0] = false
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

const maxSegmentNumber = 62

func (bwa *BWArr[T]) extend() {
	l := len(bwa.whiteSegments[len(bwa.whiteSegments)-1].elements)
	bwa.whiteSegments = append(bwa.whiteSegments, segment[T]{elements: make([]T, l*2), deleted: make([]bool, l*2),
		deletedNum: 0})
	bwa.blackSegments = append(bwa.blackSegments, segment[T]{elements: make([]T, l), deleted: make([]bool, l),
		deletedNum: 0})
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
			elements:   make([]T, 1<<i),
			deleted:    make([]bool, 1<<i),
			deletedNum: 0,
		}
	}
	return segments
}
