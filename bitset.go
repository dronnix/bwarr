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
	if s.Get(idx) {
		return
	}
	for _, layer := range s.layers {
		elementIdx := idx >> intDiv64
		bitIdx := idx & reminder64
		layer[elementIdx] |= 1 << bitIdx
		if layer[elementIdx] != allSet {
			break
		}
		idx = elementIdx
	}
}

// SetIfTrue Sets a bit if it's true. Useful for newly created or reset bitset, so we can skip Unset operation.
func (s *LayeredBitSet) SetIfTrue(idx int, value bool) {
	if value {
		s.Set(idx)
	}
}

// Unset given bit. Prefer to use Reset + SetIfTrue instead of Unset if possible
func (s *LayeredBitSet) Unset(idx int) {
	if !s.Get(idx) {
		return
	}
	for _, layer := range s.layers {
		elementIdx := idx >> intDiv64
		wasAllSet := layer[elementIdx] == allSet
		layer[elementIdx] &^= 1 << (idx & reminder64)
		if !wasAllSet {
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
		clear(layer)
	}
}

// ResetFrom resets bits with index greater or equal to idx
func (s *LayeredBitSet) ResetFrom(idx int) {
	for _, layer := range s.layers {
		elementIdx := idx >> intDiv64
		layer[elementIdx] &= (uint64(1) << (idx & reminder64)) - 1
		clear(layer[elementIdx+1:])
		idx >>= intDiv64
	}
}

// CopyFrom copies bits with index >= idx from `from` into s, preserving bits below idx.
func (s *LayeredBitSet) CopyFrom(from *LayeredBitSet, idx int) {
	var prevBoundaryAllSet bool
	for i, layer := range s.layers {
		elementIdx := idx >> intDiv64
		lowMask := (uint64(1) << (idx & reminder64)) - 1
		layer[elementIdx] = (layer[elementIdx] & lowMask) | (from.layers[i][elementIdx] &^ lowMask)
		copy(layer[elementIdx+1:], from.layers[i][elementIdx+1:])
		// The merge mixed s and from in the boundary element, so the summary bit
		// copied from `from` for this element may be wrong. Fix it.
		if i > 0 {
			bit := uint64(1) << (idx & reminder64)
			if prevBoundaryAllSet {
				layer[elementIdx] |= bit
			} else {
				layer[elementIdx] &^= bit
			}
		}
		prevBoundaryAllSet = layer[elementIdx] == allSet
		idx >>= intDiv64
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

func (s *LayeredBitSet) FindNextUnsetBit(idx int) int {
	panic("not implemented")
}

func (s *LayeredBitSet) FindFirstUnsetBit() int {
	panic("not implemented")
}

func (s *LayeredBitSet) FindLastUnsetBit() int {
	panic("not implemented")
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
