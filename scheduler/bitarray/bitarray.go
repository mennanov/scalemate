// Package bitarray provides a simple Bit Array implementation tailored specifically for the Job scheduling purpose.
package bitarray

import (
	"math"
)

// BitArray represents an array of bits of a fixed length.
type BitArray struct {
	Blocks  []uint64
	BitsSet uint64
}

// NewBitArray creates a new BitArray of a fixed size.
func NewBitArray(size uint64) *BitArray {
	return &BitArray{
		Blocks: make([]uint64, int(math.Ceil(float64(size)/float64(64)))),
	}
}

// SetBit sets the i'th bit of the array to true.
func (b *BitArray) SetBit(i uint64) {
	blockIndex := i / 64
	bitIndex := uint8(i % 64)
	b.Blocks[blockIndex] = b.Blocks[blockIndex] | (1 << bitIndex)
	b.BitsSet++
}

// GetBit gets the bool value of the i'th bit of the array.
func (b *BitArray) GetBit(i uint64) bool {
	blockIndex := i / 64
	bitIndex := uint8(i % 64)
	return b.Blocks[blockIndex]&(1<<bitIndex) != 0
}

// Or performs a bitwise OR operation on two arrays of the same size. Returns a new BitArray.
func (b *BitArray) Or(b2 *BitArray) *BitArray {
	if len(b.Blocks) != len(b2.Blocks) {
		panic("BitArrays have different blocks length")
	}
	ba := &BitArray{
		Blocks: make([]uint64, len(b.Blocks)),
	}
	for i := range ba.Blocks {
		ba.Blocks[i] = b.Blocks[i] | b2.Blocks[i]
		ba.BitsSet += positiveBits(ba.Blocks[i])
	}
	return ba
}

// SetBits returns a slice of indexes of the bits that are set (true) in this array.
func (b *BitArray) SetBits() []uint64 {
	result := make([]uint64, 0)
	for i, block := range b.Blocks {
		for j := uint64(0); j < 64; j++ {
			if block&(1<<j) != 0 {
				result = append(result, uint64(i*64)+j)
			}
		}
	}
	return result
}

// positiveBits counts the number of positive bits in a given number.
func positiveBits(n uint64) uint64 {
	c := uint64(0)
	for n > 0 {
		n = n & (n - 1)
		c++
	}
	return c
}

// MaxBitArray returns the BitArray that has the most set bits count.
func MaxBitArray(b1, b2 *BitArray) *BitArray {
	if b1 == nil && b2 == nil {
		return nil
	}
	if b1 == nil && b2 != nil {
		return b2
	}
	if b1 != nil && b2 == nil {
		return b1
	}
	if b1.BitsSet > b2.BitsSet {
		return b1
	}
	return b2
}
