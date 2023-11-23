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
