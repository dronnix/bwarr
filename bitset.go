package bwarr

import (
	"math/bits"
)

// LayeredBitSet is a special version of bitset, optimized for storing BWArr deleted elements.
// Layer 0 is the original bitset, where each bit represents whether the corresponding element is deleted.
// Layer I is a bitset where each bit represents whether the corresponding 64 bits in layer I-1 are all set.
// This way, we can quickly skip over large blocks of deleted elements.
type LayeredBitSet struct {
	layers     [][]uint64
	size       int // logical number of bits; Find methods constrain results to [0, size)
	firstUnset int
	lastUnset  int
}

const wordBits = 64
const wordShift = 6 // log2(wordBits)
const wordMask = wordBits - 1
const allSet = ^uint64(0)

// typicalMaxLayers is a capacity hint: 4 layers cover 64^4 = 16M bits without reallocation.
const typicalMaxLayers = 4

func NewLayeredBitSet(size int) *LayeredBitSet {
	if size <= 0 {
		panic("bwarr: LayeredBitSet size must be positive")
	}
	// Each layer summarizes the 64-bit words of the layer below; add layers until one word covers everything.
	layers := make([][]uint64, 0, typicalMaxLayers)
	for words := (size + wordMask) >> wordShift; ; words = (words + wordMask) >> wordShift {
		layers = append(layers, make([]uint64, words))
		if words == 1 {
			break
		}
	}
	return &LayeredBitSet{layers: layers, size: size, firstUnset: 0, lastUnset: size - 1}
}

func (s *LayeredBitSet) Set(idx int) {
	if s.Get(idx) {
		return
	}
	origIdx := idx
	for _, layer := range s.layers {
		elementIdx := idx >> wordShift
		bitIdx := idx & wordMask
		layer[elementIdx] |= 1 << bitIdx
		if layer[elementIdx] != allSet {
			break
		}
		idx = elementIdx
	}
	if s.firstUnset == origIdx {
		s.firstUnset = s.findFirstUnsetBit()
	}
	if s.lastUnset == origIdx {
		// Bits above lastUnset are all set by definition and origIdx was just set,
		// so the new last unset bit is strictly below origIdx.
		s.lastUnset = s.FindPrevUnsetBit(origIdx)
	}
}

// Unset clears the given bit.
func (s *LayeredBitSet) Unset(idx int) {
	if !s.Get(idx) {
		return
	}
	origIdx := idx
	for _, layer := range s.layers {
		elementIdx := idx >> wordShift
		wasAllSet := layer[elementIdx] == allSet
		layer[elementIdx] &^= 1 << (idx & wordMask)
		if !wasAllSet {
			break
		}
		idx = elementIdx
	}
	if origIdx < s.firstUnset || s.firstUnset < 0 {
		s.firstUnset = origIdx
	}
	if origIdx > s.lastUnset {
		s.lastUnset = origIdx
	}
}

func (s *LayeredBitSet) Get(idx int) bool {
	element := s.layers[0][idx>>wordShift]
	if element == 0 {
		return false
	}
	return (element & (1 << (idx & wordMask))) != 0
}

func (s *LayeredBitSet) DeepCopy() *LayeredBitSet {
	layersCopy := make([][]uint64, len(s.layers))
	for i, layer := range s.layers {
		layerCopy := make([]uint64, len(layer))
		copy(layerCopy, layer)
		layersCopy[i] = layerCopy
	}
	return &LayeredBitSet{layers: layersCopy, size: s.size, firstUnset: s.firstUnset, lastUnset: s.lastUnset}
}

func (s *LayeredBitSet) Reset() {
	for _, layer := range s.layers {
		clear(layer)
	}
	s.firstUnset = 0
	s.lastUnset = s.size - 1
}

// FindPrevUnsetBit returns the index of the closest unset bit with lower index  or -1 if all bits are set.
func (s *LayeredBitSet) FindPrevUnsetBit(idx int) int {
	// The algorithm is optimized to work faster with small series of unset bits, which is the common case for BWArr deleted elements.
	// So, it is bottom-up-bottom: we start from the lowest layer and go up until we find a layer with an unset bit,
	// then we go down to find the exact index of that bit.
	l, bitIdx := 0, 0
	for ; l < len(s.layers); l++ {
		bitIdx = idx & wordMask
		idx = idx >> wordShift // nolint:gocritic
		bitIdx = findPrevUnsetBit(s.layers[l][idx], bitIdx)
		if bitIdx >= 0 {
			break
		}
	}
	if bitIdx < 0 {
		return -1
	}

	for ; l > 0; l-- {
		idx = idx<<wordShift + bitIdx
		bitIdx = findLastUnsetBit(s.layers[l-1][idx])
	}

	return idx<<wordShift + bitIdx
}

// FindNextUnsetBit returns the index of the closest unset bit with higher index or -1 if all bits are set.
func (s *LayeredBitSet) FindNextUnsetBit(idx int) int {
	l, bitIdx := 0, 0
	for ; l < len(s.layers); l++ {
		bitIdx = idx & wordMask
		idx >>= wordShift
		bitIdx = findNextUnsetBit(s.layers[l][idx], bitIdx)
		if bitIdx < wordBits {
			break
		}
	}
	if bitIdx >= wordBits {
		return -1
	}

	for ; l > 0; l-- {
		idx = idx<<wordShift + bitIdx
		if idx >= len(s.layers[l-1]) {
			return -1 // phantom bit in summary layer — beyond actual data
		}
		bitIdx = findFirstUnsetBit(s.layers[l-1][idx])
	}

	result := idx<<wordShift + bitIdx
	if result >= s.size {
		return -1
	}
	return result
}

func (s *LayeredBitSet) FindFirstUnsetBit() int {
	return s.firstUnset
}

func (s *LayeredBitSet) FindLastUnsetBit() int {
	return s.lastUnset
}

func (s *LayeredBitSet) findFirstUnsetBit() int {
	// Use top-bottom approach, optimized for tail deletions:
	elemIdx := 0
	for l := len(s.layers) - 1; l >= 0; l-- {
		if elemIdx >= len(s.layers[l]) {
			return -1 // summary layer points beyond actual data
		}
		bitIndex := findFirstUnsetBit(s.layers[l][elemIdx])
		if bitIndex >= wordBits {
			return -1
		}
		elemIdx = elemIdx<<wordShift + bitIndex
	}
	if elemIdx >= s.size {
		return -1
	}
	return elemIdx
}

// findFirstUnsetBit returns position of the lowest unset bit in the given element,
// or wordBits if all bits are set.
func findFirstUnsetBit(element uint64) int {
	return bits.TrailingZeros64(^element)
}

// findLastUnsetBit returns position of the highest unset bit in the given element,
// or -1 if all bits are set.
func findLastUnsetBit(element uint64) int {
	return wordBits - 1 - bits.LeadingZeros64(^element)
}

// findNextUnsetBit returns position of the closest unset bit with higher index than pos in the given element,
// or wordBits if all bits with higher index are set.
func findNextUnsetBit(element uint64, pos int) int {
	skipBits := pos + 1
	if skipBits >= wordBits {
		return wordBits
	}
	return skipBits + bits.TrailingZeros64(^(element >> skipBits))
}

// findPrevUnsetBit returns position of the closest unset bit with lower index than pos in the given element,
// or negative if all bits with lower index are set.
func findPrevUnsetBit(element uint64, pos int) int {
	skipBits := wordBits - pos    // skip bits with higher index than pos, including pos itself
	element = element << skipBits // nolint:gocritic
	// Invert bits to be able to use LeadingZeros to skip ones
	// -1 because 0 leading zeros means next bit to pos, according to the skipBits:
	return pos - bits.LeadingZeros64(^element) - 1
}
