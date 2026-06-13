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
	layers     [][]uint64
	size       int // logical number of bits; Find methods constrain results to [0, size)
	firstUnset int
	lastUnset  int
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

	return &LayeredBitSet{layers: layers, size: size, firstUnset: 0, lastUnset: size - 1}
}

func (s *LayeredBitSet) Set(idx int) {
	if s.Get(idx) {
		return
	}
	origIdx := idx
	for _, layer := range s.layers {
		elementIdx := idx >> intDiv64
		bitIdx := idx & reminder64
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
		s.lastUnset = s.findLastUnsetBit()
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
	origIdx := idx
	for _, layer := range s.layers {
		elementIdx := idx >> intDiv64
		wasAllSet := layer[elementIdx] == allSet
		layer[elementIdx] &^= 1 << (idx & reminder64)
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
	return &LayeredBitSet{layers: layersCopy, size: s.size, firstUnset: s.firstUnset, lastUnset: s.lastUnset}
}

func (s *LayeredBitSet) Reset() {
	for _, layer := range s.layers {
		clear(layer)
	}
	s.firstUnset = 0
	s.lastUnset = s.size - 1
}

// ResetFrom resets bits with index greater or equal to idx
func (s *LayeredBitSet) ResetFrom(idx int) {
	origIdx := idx
	for _, layer := range s.layers {
		elementIdx := idx >> intDiv64
		layer[elementIdx] &= (uint64(1) << (idx & reminder64)) - 1
		clear(layer[elementIdx+1:])
		idx >>= intDiv64
	}
	if origIdx < s.size {
		if s.firstUnset < 0 || origIdx < s.firstUnset {
			s.firstUnset = origIdx
		}
		s.lastUnset = s.size - 1
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

// CopyTailFrom overwrites s's suffix [dstStart, s.size) with src's suffix [srcStart, src.size),
// preserving s's bits below dstStart. The two suffixes must have equal length
// (s.size-dstStart == src.size-srcStart). Unlike CopyFrom, the bits are relocated: src bit
// srcStart+k lands at s bit dstStart+k, so a nonzero shift dstStart-srcStart is supported.
// Summary layers and the firstUnset/lastUnset cache are updated.
//
// The fast path is a word-level layer-0 copy and runs when the two starts share the same
// intra-word offset and both sizes are whole words; this is the common segment-merge case
// (the relocation distance is a power of two >= 64). Otherwise it falls back to a per-bit copy.
func (s *LayeredBitSet) CopyTailFrom(dstStart int, src *LayeredBitSet, srcStart int) {
	wordAligned := dstStart&reminder64 == srcStart&reminder64 &&
		s.size&reminder64 == 0 && src.size&reminder64 == 0
	if !wordAligned {
		for d, srcIdx := dstStart, srcStart; d < s.size; d, srcIdx = d+1, srcIdx+1 {
			if src.Get(srcIdx) {
				s.Set(d)
			} else {
				s.Unset(d)
			}
		}
		return
	}

	sw, dw := srcStart>>intDiv64, dstStart>>intDiv64
	if so := srcStart & reminder64; so == 0 {
		copy(s.layers[0][dw:], src.layers[0][sw:])
	} else {
		highMask := ^((uint64(1) << so) - 1) // bits [so, 64)
		s.layers[0][dw] = (s.layers[0][dw] &^ highMask) | (src.layers[0][sw] & highMask)
		copy(s.layers[0][dw+1:], src.layers[0][sw+1:])
	}
	s.rebuildSummaryFrom(dstStart)
	s.firstUnset = s.findFirstUnsetBit()
	s.lastUnset = s.findLastUnsetBit()
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
		bitIdx = findLastUnsetBit(s.layers[l-1][idx])
	}

	return idx<<intDiv64 + bitIdx
}

// FindNextUnsetBit returns the index of the closest unset bit with higher index or -1 if all bits are set.
func (s *LayeredBitSet) FindNextUnsetBit(idx int) int {
	l, bitIdx := 0, 0
	for ; l < len(s.layers); l++ {
		bitIdx = idx & reminder64
		idx >>= intDiv64
		bitIdx = findNextUnsetBit(s.layers[l][idx], bitIdx)
		if bitIdx < bitsNum {
			break
		}
	}
	if bitIdx >= bitsNum {
		return -1
	}

	for ; l > 0; l-- {
		idx = idx<<intDiv64 + bitIdx
		if idx >= len(s.layers[l-1]) {
			return -1 // phantom bit in summary layer — beyond actual data
		}
		bitIdx = findFirstUnsetBit(s.layers[l-1][idx])
	}

	result := idx<<intDiv64 + bitIdx
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
		if bitIndex >= bitsNum {
			return -1
		}
		elemIdx = elemIdx<<intDiv64 + bitIndex
	}
	if elemIdx >= s.size {
		return -1
	}
	return elemIdx
}

func (s *LayeredBitSet) findLastUnsetBit() int {
	last := s.size - 1
	if !s.Get(last) {
		return last
	}
	return s.FindPrevUnsetBit(last)
}

// rebuildSummaryFrom recomputes summary-layer bits for every layer-0 word at or above the word
// containing idx, propagating the result upward through all layers. Layer 0 must already hold the
// final bits. Summary bits covering words below idx's word are left untouched.
func (s *LayeredBitSet) rebuildSummaryFrom(idx int) {
	for l := 1; l < len(s.layers); l++ {
		lower, cur := s.layers[l-1], s.layers[l]
		for w := idx >> intDiv64; w < len(lower); w++ {
			cw, bit := w>>intDiv64, w&reminder64
			if lower[w] == allSet {
				cur[cw] |= 1 << bit
			} else {
				cur[cw] &^= 1 << bit
			}
		}
		idx >>= intDiv64
	}
}

// findFirstUnsetBit returns position of the lowest unset bit in the given element,
// or bitsNum if all bits are set.
func findFirstUnsetBit(element uint64) int {
	return bits.TrailingZeros64(^element)
}

// findLastUnsetBit returns position of the highest unset bit in the given element,
// or -1 if all bits are set.
func findLastUnsetBit(element uint64) int {
	return bitsNum - 1 - bits.LeadingZeros64(^element)
}

// findNextUnsetBit returns position of the closest unset bit with higher index than pos in the given element,
// or bitsNum if all bits with higher index are set.
func findNextUnsetBit(element uint64, pos int) int {
	skipBits := pos + 1
	if skipBits >= bitsNum {
		return bitsNum
	}
	return skipBits + bits.TrailingZeros64(^(element >> skipBits))
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
