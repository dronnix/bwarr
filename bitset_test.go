package bwarr

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const size = 4 * 1024 * 1024
const reminderSize = size - 1

func TestNewLayeredBitSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		size           int
		wantLayersNum  int
		wantLayerSizes []int
	}{
		{
			name:           "size 1",
			size:           1,
			wantLayersNum:  1,
			wantLayerSizes: []int{1},
		},
		{
			name:           "size 64",
			size:           64,
			wantLayersNum:  1,
			wantLayerSizes: []int{1},
		},
		{
			name:           "size 65",
			size:           65,
			wantLayersNum:  2,
			wantLayerSizes: []int{2, 1},
		},
		{
			name:           "size 128",
			size:           128,
			wantLayersNum:  2,
			wantLayerSizes: []int{2, 1},
		},
		{
			name:           "size 4096",
			size:           4096,
			wantLayersNum:  2,
			wantLayerSizes: []int{64, 1},
		},
		{
			name:           "size 4097",
			size:           4097,
			wantLayersNum:  3,
			wantLayerSizes: []int{65, 2, 1},
		},
		{
			name:           "size 262144", // 64^3
			size:           262144,
			wantLayersNum:  3,
			wantLayerSizes: []int{4096, 64, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bs := NewLayeredBitSet(tt.size)

			require.NotNil(t, bs)
			assert.Len(t, bs.layers, tt.wantLayersNum)

			layerSizes := make([]int, len(bs.layers))
			for i, layer := range bs.layers {
				layerSizes[i] = len(layer)
			}
			assert.Equal(t, tt.wantLayerSizes, layerSizes)
		})
	}
}

func TestLayeredBitSet_Set(t *testing.T) {
	t.Parallel()

	// 64^3 = 262144, gives 3 layers: [4096][64][1]
	bs := NewLayeredBitSet(262144)

	// Single bit — no propagation.
	bs.Set(3)
	assert.Equal(t, uint64(1<<3), bs.layers[0][0])
	assert.Equal(t, uint64(0), bs.layers[1][0], "layer 1 should not propagate for partial element")

	// Idempotent.
	bs.Set(3)
	assert.Equal(t, uint64(1<<3), bs.layers[0][0])

	// Fill all 64 bits in element 0 of layer 0 — should propagate to layer 1.
	for i := range bitsNum {
		bs.Set(i)
	}
	assert.Equal(t, ^uint64(0), bs.layers[0][0])
	assert.Equal(t, uint64(1), bs.layers[1][0], "layer 1 bit 0 set after element 0 full")

	// Fill remaining elements 1..63 of layer 0 — should propagate through layer 1 to layer 2.
	for i := 64; i < 4096; i++ {
		bs.Set(i)
	}
	assert.Equal(t, ^uint64(0), bs.layers[1][0], "layer 1 element 0 all-set")
	assert.Equal(t, uint64(1), bs.layers[2][0], "layer 2 bit 0 set")
}

func TestLayeredBitSet_Unset(t *testing.T) {
	t.Parallel()

	t.Run("unset already unset bit is no-op", func(t *testing.T) {
		t.Parallel()
		bs := NewLayeredBitSet(256)
		bs.Unset(42)
		assert.False(t, bs.Get(42))
	})

	t.Run("unset single bit no propagation", func(t *testing.T) {
		t.Parallel()
		bs := NewLayeredBitSet(256)
		bs.Set(3)
		bs.Set(10)
		bs.Unset(3)
		assert.False(t, bs.Get(3))
		assert.True(t, bs.Get(10), "neighboring bit should be unaffected")
	})

	t.Run("unset propagates to layer 1", func(t *testing.T) {
		t.Parallel()
		// 3 layers: [4096][64][1]
		bs := NewLayeredBitSet(262144)
		// Fill element 0 of layer 0 fully — propagates to layer 1.
		for i := range bitsNum {
			bs.Set(i)
		}
		require.Equal(t, allSet, bs.layers[0][0])
		require.Equal(t, uint64(1), bs.layers[1][0])

		// Unset one bit — must clear the summary bit in layer 1.
		bs.Unset(3)
		assert.False(t, bs.Get(3))
		assert.NotEqual(t, allSet, bs.layers[0][0], "layer 0 element 0 should no longer be all-set")
		assert.Equal(t, uint64(0), bs.layers[1][0], "layer 1 bit 0 should be cleared")
	})

	t.Run("unset propagates through all layers", func(t *testing.T) {
		t.Parallel()
		// 3 layers: [4096][64][1]
		bs := NewLayeredBitSet(262144)
		// Fill all bits 0..4095 — full propagation to layer 2.
		for i := range 4096 {
			bs.Set(i)
		}
		require.Equal(t, allSet, bs.layers[1][0])
		require.Equal(t, uint64(1), bs.layers[2][0])

		bs.Unset(0)
		assert.False(t, bs.Get(0))
		assert.Equal(t, uint64(0), bs.layers[1][0]&1, "layer 1 bit 0 should be cleared")
		assert.Equal(t, uint64(0), bs.layers[2][0], "layer 2 bit 0 should be cleared")
	})

	t.Run("set after unset restores bit", func(t *testing.T) {
		t.Parallel()
		bs := NewLayeredBitSet(256)
		bs.Set(50)
		bs.Unset(50)
		assert.False(t, bs.Get(50))
		bs.Set(50)
		assert.True(t, bs.Get(50))
	})

	t.Run("unset is idempotent", func(t *testing.T) {
		t.Parallel()
		bs := NewLayeredBitSet(256)
		bs.Set(7)
		bs.Unset(7)
		bs.Unset(7)
		assert.False(t, bs.Get(7))
	})

	t.Run("unset only affects target bit in element", func(t *testing.T) {
		t.Parallel()
		bs := NewLayeredBitSet(256)
		// Set all 64 bits then unset one.
		for i := range bitsNum {
			bs.Set(i)
		}
		bs.Unset(31)
		for i := range bitsNum {
			if i == 31 {
				assert.False(t, bs.Get(i), "bit 31 should be unset")
			} else {
				assert.True(t, bs.Get(i), "bit %d should remain set", i)
			}
		}
	})
}

func TestLayeredBitSet_SetNum(t *testing.T) {
	t.Parallel()

	t.Run("zero on new bitset", func(t *testing.T) {
		t.Parallel()
		bs := NewLayeredBitSet(256)
		assert.Equal(t, 0, bs.SetNum())
	})

	t.Run("increments on Set", func(t *testing.T) {
		t.Parallel()
		bs := NewLayeredBitSet(256)
		bs.Set(0)
		bs.Set(100)
		bs.Set(200)
		assert.Equal(t, 3, bs.SetNum())
	})

	t.Run("idempotent Set does not double count", func(t *testing.T) {
		t.Parallel()
		bs := NewLayeredBitSet(256)
		bs.Set(42)
		bs.Set(42)
		bs.Set(42)
		assert.Equal(t, 1, bs.SetNum())
	})

	t.Run("decrements on Unset", func(t *testing.T) {
		t.Parallel()
		bs := NewLayeredBitSet(256)
		bs.Set(10)
		bs.Set(20)
		bs.Unset(10)
		assert.Equal(t, 1, bs.SetNum())
	})

	t.Run("idempotent Unset does not double decrement", func(t *testing.T) {
		t.Parallel()
		bs := NewLayeredBitSet(256)
		bs.Set(7)
		bs.Unset(7)
		bs.Unset(7)
		assert.Equal(t, 0, bs.SetNum())
	})

	t.Run("Unset on unset bit is no-op", func(t *testing.T) {
		t.Parallel()
		bs := NewLayeredBitSet(256)
		bs.Unset(42)
		assert.Equal(t, 0, bs.SetNum())
	})

	t.Run("Reset zeroes counter", func(t *testing.T) {
		t.Parallel()
		bs := NewLayeredBitSet(256)
		for i := range 100 {
			bs.Set(i)
		}
		require.Equal(t, 100, bs.SetNum())
		bs.Reset()
		assert.Equal(t, 0, bs.SetNum())
	})

	t.Run("DeepCopy preserves counter", func(t *testing.T) {
		t.Parallel()
		bs := NewLayeredBitSet(256)
		bs.Set(5)
		bs.Set(100)
		cp := bs.DeepCopy()
		assert.Equal(t, 2, cp.SetNum())

		// Mutations are independent.
		cp.Set(0)
		assert.Equal(t, 3, cp.SetNum())
		assert.Equal(t, 2, bs.SetNum())
	})

	t.Run("Set and Unset roundtrip", func(t *testing.T) {
		t.Parallel()
		bs := NewLayeredBitSet(4096)
		for i := range 4096 {
			bs.Set(i)
		}
		assert.Equal(t, 4096, bs.SetNum())
		for i := range 4096 {
			bs.Unset(i)
		}
		assert.Equal(t, 0, bs.SetNum())
	})
}

func TestLayeredBitSet_Get(t *testing.T) {
	t.Parallel()

	bs := NewLayeredBitSet(256)

	// Unset bits return false.
	assert.False(t, bs.Get(0))
	assert.False(t, bs.Get(63))
	assert.False(t, bs.Get(100))

	// Set a few bits and verify Get.
	bs.Set(0)
	bs.Set(63)
	bs.Set(100)

	assert.True(t, bs.Get(0))
	assert.True(t, bs.Get(63))
	assert.True(t, bs.Get(100))

	// Neighbors remain unset.
	assert.False(t, bs.Get(1))
	assert.False(t, bs.Get(62))
	assert.False(t, bs.Get(64)) // different element than bit 63
	assert.False(t, bs.Get(99))
	assert.False(t, bs.Get(101))
}

func TestLayeredBitSet_DeepCopy(t *testing.T) {
	t.Parallel()

	bs := NewLayeredBitSet(256)
	bs.Set(5)
	bs.Set(100)

	cp := bs.DeepCopy()

	// Copy has the same bits set.
	assert.True(t, cp.Get(5))
	assert.True(t, cp.Get(100))
	assert.False(t, cp.Get(0))

	// Mutating copy does not affect original.
	cp.Set(0)
	assert.True(t, cp.Get(0))
	assert.False(t, bs.Get(0))

	// Mutating original does not affect copy.
	bs.Set(200)
	assert.True(t, bs.Get(200))
	assert.False(t, cp.Get(200))
}

func TestLayeredBitSet_Reset(t *testing.T) {
	t.Parallel()

	bs := NewLayeredBitSet(256)
	bs.Set(0)
	bs.Set(63)
	bs.Set(100)

	bs.Reset()

	// All bits cleared.
	assert.False(t, bs.Get(0))
	assert.False(t, bs.Get(63))
	assert.False(t, bs.Get(100))

	// All layers zeroed.
	for i, layer := range bs.layers {
		for j, val := range layer {
			assert.Zerof(t, val, "layer[%d][%d] should be zero after Reset", i, j)
		}
	}

	// Structure preserved — can Set again after Reset.
	bs.Set(42)
	assert.True(t, bs.Get(42))
}

func Test_findPrevUnsetBit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		element uint64
		pos     int
		want    int
	}{
		// Basic cases — strictly less than pos.
		{
			name:    "all zeros, pos 0 — nothing below 0",
			element: 0,
			pos:     0,
			want:    -1,
		},
		{
			name:    "all zeros, pos 63",
			element: 0,
			pos:     63,
			want:    62,
		},
		// Bit at pos is unset but we look strictly below.
		{
			name:    "bit 3 unset but pos 3 excluded, bits 0-2 set",
			element: 0b0111,
			pos:     3,
			want:    -1,
		},
		// Bit at pos is set — find previous unset.
		{
			name:    "only bit 3 set, pos 3",
			element: 0b1000,
			pos:     3,
			want:    2,
		},
		{
			name:    "bits 2 and 3 set, pos 3",
			element: 0b1100,
			pos:     3,
			want:    1,
		},
		{
			name:    "bits 0-3 all set, pos 3",
			element: 0b1111,
			pos:     3,
			want:    -1,
		},
		// All bits set — no unset bit anywhere.
		{
			name:    "all set, pos 63",
			element: ^uint64(0),
			pos:     63,
			want:    -1,
		},
		{
			name:    "all set, pos 0",
			element: ^uint64(0),
			pos:     0,
			want:    -1,
		},
		// Only bit 0 unset in lower nibble.
		{
			name:    "bit 0 unset rest set, pos 3",
			element: 0b1110,
			pos:     3,
			want:    0,
		},
		// Bits above pos are irrelevant.
		{
			name:    "all set except bit 5, pos 5 excluded — bits 0-4 set",
			element: ^uint64(0) &^ (1 << 5),
			pos:     5,
			want:    -1,
		},
		{
			name:    "high bits unset but only low bits matter",
			element: 0b11, // bits 0,1 set
			pos:     1,
			want:    -1,
		},
		// High positions.
		{
			name:    "bit 62 unset, pos 63",
			element: ^uint64(0) &^ (1 << 62),
			pos:     63,
			want:    62,
		},
		{
			name:    "only bit 63 unset, pos 63 excluded — bits 0-62 set",
			element: ^uint64(0) &^ (1 << 63),
			pos:     63,
			want:    -1,
		},
		// Sparse pattern.
		{
			name:    "alternating 0b...10101010, pos 7",
			element: 0xAA, // 10101010
			pos:     7,
			want:    6,
		},
		{
			name:    "alternating 0b...10101010, pos 6 excluded — prev unset is 4",
			element: 0xAA,
			pos:     6,
			want:    4,
		},
		{
			name:    "alternating 0b...10101010, pos 5 — bit 5 set, prev unset is 4",
			element: 0xAA,
			pos:     5,
			want:    4,
		},
		{
			name:    "full element search",
			element: allSet - 2,
			pos:     bitsNum,
			want:    1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tt.want, findPrevUnsetBit(tt.element, tt.pos), "findPrevUnsetBit(%064b, %d)", tt.element, tt.pos)
		})
	}
}

func TestLayeredBitSet_FindPrevUnsetBit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size int
		set  []int // bits to set before searching
		idx  int
		want int
	}{
		{
			name: "no bits set, idx 0 — nothing below 0",
			size: 64,
			set:  nil,
			idx:  0,
			want: -1,
		},
		{
			name: "no bits set, idx 10",
			size: 64,
			set:  nil,
			idx:  10,
			want: 9,
		},
		{
			name: "bit 5 set, idx 10",
			size: 64,
			set:  []int{5},
			idx:  10,
			want: 9,
		},
		{
			name: "bits 0-8 set, idx 10 — bit 9 unset",
			size: 64,
			set:  []int{0, 1, 2, 3, 4, 5, 6, 7, 8},
			idx:  10,
			want: 9,
		},
		{
			name: "bits 0-9 all set, idx 10 — none below",
			size: 64,
			set:  []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			idx:  10,
			want: -1,
		},
		{
			name: "all 64 bits set, idx 63",
			size: 64,
			set:  seq(0, 64),
			idx:  63,
			want: -1,
		},
		// Cross-element: idx in element 1, result in element 0.
		{
			name: "no bits set, idx 100 — crosses element boundary",
			size: 256,
			set:  nil,
			idx:  100,
			want: 99,
		},
		{
			name: "bits 64-99 set, idx 100 — falls back to element 0",
			size: 256,
			set:  seq(64, 100),
			idx:  100,
			want: 63,
		},
		{
			name: "bits 0-99 all set, idx 100",
			size: 256,
			set:  seq(0, 100),
			idx:  100,
			want: -1,
		},
		// Cross-layer: needs to go up to layer 1 and back down.
		{
			name: "first 128 bits set, idx 128 — skips via layer 1",
			size: 4096,
			set:  seq(0, 128),
			idx:  128,
			want: -1,
		},
		{
			name: "bits 1-127 set, bit 0 unset, idx 128",
			size: 4096,
			set:  seq(1, 128),
			idx:  128,
			want: 0,
		},
		{
			name: "first element full, second partial, idx 100",
			size: 256,
			set:  seq(0, 64),
			idx:  100,
			want: 99,
		},
		// Large multi-layer traversal.
		{
			name: "bits 1-4095 set, bit 0 unset, idx 4095",
			size: 4096,
			set:  seq(1, 4095),
			idx:  4095,
			want: 0,
		},
		// Non-power-of-2 sizes.
		{
			name: "size 100, no bits set, idx 99",
			size: 100,
			set:  nil,
			idx:  99,
			want: 98,
		},
		{
			name: "size 100, bits 0-98 set, idx 99",
			size: 100,
			set:  seq(0, 99),
			idx:  99,
			want: -1,
		},
		{
			name: "size 100, bits 65-98 set, idx 99 — falls back to element 0",
			size: 100,
			set:  seq(65, 99),
			idx:  99,
			want: 64,
		},
		{
			name: "size 200, bits 1-199 set, bit 0 unset, idx 199",
			size: 200,
			set:  seq(1, 199),
			idx:  199,
			want: 0,
		},
		{
			name: "size 5000, bits 1-4999 set, bit 0 unset, idx 4999",
			size: 5000,
			set:  seq(1, 4999),
			idx:  4999,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bs := NewLayeredBitSet(tt.size)
			for _, i := range tt.set {
				bs.Set(i)
			}

			var got int
			require.NotPanics(t, func() {
				got = bs.FindPrevUnsetBit(tt.idx)
			})
			assert.Equal(t, tt.want, got)
		})
	}
}

// seq returns a slice of ints [from, to).
func seq(from, to int) []int {
	s := make([]int, 0, to-from)
	for i := from; i < to; i++ {
		s = append(s, i)
	}
	return s
}

var benchResult int //nolint:gochecknoglobals // prevent compiler optimization

func Benchmark_findPrevUnsetBit(b *testing.B) {
	sparse := uint64(0x8000000000000001)         // few bits set
	dense := ^uint64(0) &^ (1 << 31) &^ (1 << 7) // most bits set, few gaps
	allSet := ^uint64(0)

	b.Run("sparse", func(b *testing.B) {
		for range b.N {
			benchResult = findPrevUnsetBit(sparse, 63)
		}
	})
	b.Run("dense", func(b *testing.B) {
		for range b.N {
			benchResult = findPrevUnsetBit(dense, 63)
		}
	})
	b.Run("all_set", func(b *testing.B) {
		for range b.N {
			benchResult = findPrevUnsetBit(allSet, 63)
		}
	})
}

func Benchmark_FindPrevUnsetBit(b *testing.B) {
	b.Run("best_case_prev_bit_unset", func(b *testing.B) {
		// Bit right before idx is unset — found in layer 0, no ascend.
		bs := NewLayeredBitSet(size)
		bs.Set(size - 1)

		b.ResetTimer()
		for range b.N {
			benchResult = bs.FindPrevUnsetBit(size - 1)
		}
	})

	b.Run("same_element", func(b *testing.B) {
		// All bits in the element set except bit 0 — found in layer 0, same element.
		bs := NewLayeredBitSet(size)
		for i := 1; i < 64; i++ {
			bs.Set(i)
		}

		b.ResetTimer()
		for range b.N {
			benchResult = bs.FindPrevUnsetBit(63)
		}
	})

	b.Run("prev_element", func(b *testing.B) {
		// Current element fully set — falls back to previous element in layer 0.
		bs := NewLayeredBitSet(size)
		for i := 64; i < 128; i++ {
			bs.Set(i)
		}

		b.ResetTimer()
		for range b.N {
			benchResult = bs.FindPrevUnsetBit(127)
		}
	})

	b.Run("cross_64_elements", func(b *testing.B) {
		// 64 elements (4096 bits) fully set — ascend to layer 1, descend back.
		bs := NewLayeredBitSet(size)
		for i := 1; i < 4096; i++ {
			bs.Set(i)
		}

		b.ResetTimer()
		for range b.N {
			benchResult = bs.FindPrevUnsetBit(4095)
		}
	})

	b.Run("worst_case_4M", func(b *testing.B) {
		// All bits set except bit 0 — must traverse all layers up and back down.
		bs := NewLayeredBitSet(size)
		for i := 1; i < size; i++ {
			bs.Set(i)
		}

		b.ResetTimer()
		for range b.N {
			benchResult = bs.FindPrevUnsetBit(size - 1)
		}
	})

	b.Run("not_found_4M", func(b *testing.B) {
		// All bits set — must go all the way up and return -1.
		bs := NewLayeredBitSet(size)
		for i := range size {
			bs.Set(i)
		}

		b.ResetTimer()
		for range b.N {
			benchResult = bs.FindPrevUnsetBit(size - 1)
		}
	})
}

func Benchmark_Set(b *testing.B) {
	b.Run("no_propagation", func(b *testing.B) {
		// One bit per element — no element ever becomes fully set, no propagation.
		bs := NewLayeredBitSet(size)

		b.ResetTimer()
		for i := range b.N {
			bs.Set((i * bitsNum) % size)
		}
	})

	b.Run("full_propagation", func(b *testing.B) {
		bs := NewLayeredBitSet(size)
		for i := range size {
			bs.Set(i)
		}

		b.ResetTimer()
		for i := range b.N {
			bs.Set(i & reminderSize)
		}
	})

	b.Run("random_access", func(b *testing.B) {
		bs := NewLayeredBitSet(size)
		indices := make([]int, size)
		for i := range indices {
			indices[i] = rand.Intn(size)
		}

		b.ResetTimer()
		for i := range b.N {
			bs.Set(indices[i&reminderSize])
		}
	})
}

func Benchmark_Unset(b *testing.B) {
	b.Run("already_unset", func(b *testing.B) {
		// Bit is not set — early return via Get.
		bs := NewLayeredBitSet(size)

		b.ResetTimer()
		for i := range b.N {
			bs.Unset(i & reminderSize)
		}
	})

	b.Run("no_propagation", func(b *testing.B) {
		// Element is partially set — unset exits after layer 0.
		bs := NewLayeredBitSet(size)
		for i := 0; i < size; i += 2 {
			bs.Set(i) // set even bits only, no element is fully set
		}
		indices := make([]int, size/2)
		for i := range indices {
			indices[i] = i * 2 // even indices (set bits)
		}

		b.ResetTimer()
		for i := range b.N {
			idx := indices[i%(size/2)]
			bs.Unset(idx)
			bs.layers[0][idx>>intDiv64] |= 1 << (idx & reminder64) // restore for next iteration
		}
	})

	b.Run("propagation", func(b *testing.B) {
		// Element fully set — unset must propagate to clear summary bits.
		bs := NewLayeredBitSet(size)
		for i := range size {
			bs.Set(i)
		}

		b.ResetTimer()
		for i := range b.N {
			idx := (i * bitsNum) & reminderSize // bit 0 of each element
			bs.Unset(idx)
			// Restore: re-set the bit and fix propagation.
			bs.Set(idx)
		}
	})

	b.Run("random_access", func(b *testing.B) {
		// Random unset/restore on a sparse bitset.
		bs := NewLayeredBitSet(size)
		indices := make([]int, size)
		for i := range indices {
			indices[i] = rand.Intn(size)
			bs.Set(indices[i])
		}

		b.ResetTimer()
		for i := range b.N {
			idx := indices[i&reminderSize]
			bs.Unset(idx)
			bs.Set(idx) // restore for next iteration
		}
	})
}

func Benchmark_Reset(b *testing.B) {
	b.Run("basic", func(b *testing.B) {
		bs := NewLayeredBitSet(size)

		b.ResetTimer()
		for range b.N {
			bs.Reset()
		}
	})
}

var benchBoolResult bool //nolint:gochecknoglobals // prevent compiler optimization

func Benchmark_Get(b *testing.B) {
	b.Run("hit_sparse", func(b *testing.B) {
		// Only a few bits set — tests the bit-check path.
		bs := NewLayeredBitSet(size)
		bs.Set(size / 2)

		b.ResetTimer()
		for range b.N {
			benchBoolResult = bs.Get(size / 2)
		}
	})

	b.Run("miss_zero_element", func(b *testing.B) {
		// Element is all zeros — tests the fast path (element == 0).
		bs := NewLayeredBitSet(size)

		b.ResetTimer()
		for range b.N {
			benchBoolResult = bs.Get(size / 2)
		}
	})

	b.Run("miss_nonzero_element", func(b *testing.B) {
		// Element has bits set but not the one we query.
		bs := NewLayeredBitSet(size)
		bs.Set(size/2 + 1)

		b.ResetTimer()
		for range b.N {
			benchBoolResult = bs.Get(size / 2)
		}
	})

	b.Run("random_access_hit", func(b *testing.B) {
		// Set every other bit, query random set bits — stresses cache.
		bs := NewLayeredBitSet(size)
		for i := 0; i < size; i += 2 {
			bs.Set(i)
		}
		// Pre-generate random indices to avoid RNG cost in the loop.
		indices := make([]int, 1024)
		for i := range indices {
			indices[i] = rand.Intn(size/2) * 2 // even indices only (set bits)
		}

		b.ResetTimer()
		for i := range b.N {
			benchBoolResult = bs.Get(indices[i&1023])
		}
	})

	b.Run("random_access_miss", func(b *testing.B) {
		// All bits unset, query random indices — stresses cache on zero elements.
		bs := NewLayeredBitSet(size)
		indices := make([]int, 1024)
		for i := range indices {
			indices[i] = rand.Intn(size)
		}

		b.ResetTimer()
		for i := range b.N {
			benchBoolResult = bs.Get(indices[i&1023])
		}
	})
}
