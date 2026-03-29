package bwarr

import (
	"math"
	"math/bits"
)

// LayeredBitSet is a special version of bitset, optimized for storing BWArr deleted elements.
// Layer 0 is the original bitset, where each bit represents whether the corresponding element is deleted.
// Layer I is a bitset where each bit represents whether the corresponding 64 bits in layer I-1 are all set.
// This way, we can quickly skip over large blocks of deleted elements.
type LayeredBitSet struct {
	layers [][]uint64
}

const bitsNum = 64
const intDiv64 = 6 // log2(bitsNum)
const reminder64 = bitsNum - 1
const allSet = ^uint64(0)

func NewLayeredBitSet(size int) *LayeredBitSet {
	layersNum := int(math.Ceil(math.Log(float64(size)) / math.Log(bitsNum)))
	layersNum = max(layersNum, 1)
	layers := make([][]uint64, layersNum)

	bitsPerElement := bitsNum
	for i := range layersNum {
		layerSize := int(math.Ceil(float64(size) / float64(bitsPerElement)))
		layers[i] = make([]uint64, layerSize)
		bitsPerElement *= bitsNum
	}

	return &LayeredBitSet{layers: layers}
}

func (s *LayeredBitSet) Set(idx int) {
	for _, layer := range s.layers {
		elementIdx := idx / bitsNum
		bitIdx := idx % bitsNum
		layer[elementIdx] |= 1 << bitIdx
		if layer[elementIdx] != allSet {
			break
		}
		idx = elementIdx
	}
}

func (s *LayeredBitSet) Get(idx int) bool {
	element := s.layers[0][idx>>intDiv64]
	if element == 0 {
		return false
	}
	return (element & (1 << (idx & reminder64))) != 0
}

func (s *LayeredBitSet) DeepCopy() *LayeredBitSet {
	layersCopy := make([][]uint64, len(s.layers))
	for i, layer := range s.layers {
		layerCopy := make([]uint64, len(layer))
		copy(layerCopy, layer)
		layersCopy[i] = layerCopy
	}
	return &LayeredBitSet{layers: layersCopy}
}

func (s *LayeredBitSet) Reset() {
	for _, layer := range s.layers {
		for i := range layer {
			layer[i] = 0
		}
	}
}

// FindPrevUnsetBit returns the index of the closest unset bit with lower index  or -1 if all bits are set.
func (s *LayeredBitSet) FindPrevUnsetBit(idx int) int {
	// The algorithm is optimized to work faster with small series of unset bits, which is the common case for BWArr deleted elements.
	// So, it is bottom-up-bottom: we start from the lowest layer and go up until we find a layer with an unset bit,
	// then we go down to find the exact index of that bit.
	l, bitIdx := 0, 0
	for ; l < len(s.layers); l++ {
		bitIdx = idx & reminder64
		idx = idx >> intDiv64 // nolint:gocritic
		bitIdx = findPrevUnsetBit(s.layers[l][idx], bitIdx)
		if bitIdx >= 0 {
			break
		}
	}
	if bitIdx < 0 {
		return -1
	}

	for ; l > 0; l-- {
		idx = idx<<intDiv64 + bitIdx
		bitIdx = findPrevUnsetBit(s.layers[l-1][idx], bitsNum)
	}

	return idx<<intDiv64 + bitIdx
}

// findPrevUnsetBit returns position of the closest unset bit with lower index than pos in the given element,
// or negative if all bits with lower index are set.
func findPrevUnsetBit(element uint64, pos int) int {
	skipBits := bitsNum - pos     // skip bits with higher index than pos, including pos itself
	element = element << skipBits // nolint:gocritic
	// Invert bits to be able to use LeadingZeros to skip ones
	// -1 because 0 leading zeros means next bit to pos, according to the skipBits:
	return pos - bits.LeadingZeros64(^element) - 1
}
