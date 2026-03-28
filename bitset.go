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
	elementIdx := idx / bitsNum
	if s.layers[0][elementIdx] == 0 {
		return false
	}
	bitIdx := idx % bitsNum
	return (s.layers[0][elementIdx] & (1 << bitIdx)) != 0
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

// FindRightmostUnsetBit returns the index of the leftmost (closest with higher index) unset bit starting from idx,
// or -1 if all bits are set.
func (s *LayeredBitSet) FindRightmostUnsetBit(idx int) int {
	// The algorithm is optimized to work faster with small series of unset bits, which is the common case for BWArr deleted elements.
	// So, it is bottom-up-bottom: we start from the lowest layer and go up until we find a layer with an unset bit,
	//then we go down to find the exact index of that bit.
	elementIdx := idx / bitsNum
	bitIdx := idx % bitsNum
	l := 0
	for l < len(s.layers) {
		element := s.layers[l][elementIdx]
		// set smaller bits to 1, so we can find the rightmost unset bit with trailing zeros count:
		element |= (1 << bitIdx) - 1
		bitIdx = bits.LeadingZeros64(element)
		if element == allSet { // Go upper layer
			bitIdx = elementIdx % bitsNum
			elementIdx = elementIdx / bitsNum
			l++
			continue
		}
		// Found element, let's find the rightmost unset bit in it:

	}
	panic("not implemented")
}

// findPrevUnsetBit returns position of the closest unset bit with lower index than pos in the given element,
// or negative if all bits with lower index are set.
func findPrevUnsetBit(element uint64, pos int) int {
	skipBits := bitsNum - pos // skip bits with higher index than pos, including pos itself
	element = element << skipBits
	// Invert bits to be able to use LeadingZeros to skip ones
	// -1 because 0 leading zeros means next bit to pos, according to the skipBits:
	return pos - bits.LeadingZeros64(^element) - 1
}
