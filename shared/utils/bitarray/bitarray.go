// Package bitarray provides a simple Bit Array implementation tailored specifically for the Job scheduling purpose.
package bitarray

import (
	"math"
)

// BitArray represents an array of bits of a fixed length.
type BitArray struct {
	BitsSet uint16
	blocks  []uint64
}

// NewBitArray creates a new BitArray of a fixed size.
func NewBitArray(size uint16) *BitArray {
	return &BitArray{
		blocks: make([]uint64, int(math.Ceil(float64(size)/float64(64)))),
	}
}

// SetBit sets the i'th bit of the array to true.
func (b *BitArray) SetBit(i uint16) {
	blockIndex := i / 64
	bitIndex := uint8(i % 64)
	b.blocks[blockIndex] = b.blocks[blockIndex] | (1 << bitIndex)
	b.BitsSet++
}

// GetBit gets the bool value of the i'th bit of the array.
func (b *BitArray) GetBit(i uint16) bool {
	blockIndex := i / 64
	bitIndex := uint8(i % 64)
	return b.blocks[blockIndex]&(1<<bitIndex) != 0
}

// Or performs a bitwise OR operation on two arrays of the same size. Returns a new BitArray.
func (b *BitArray) Or(b2 *BitArray) *BitArray {
	if len(b.blocks) != len(b2.blocks) {
		panic("BitArrays have different blocks length")
	}
	ba := &BitArray{
		blocks: make([]uint64, len(b.blocks)),
	}
	for i := range ba.blocks {
		ba.blocks[i] = b.blocks[i] | b2.blocks[i]
		ba.BitsSet += positiveBits(ba.blocks[i])
	}
	return ba
}

// SetBits returns a slice of indexes of the bits that are set (true) in this array.
func (b *BitArray) SetBits() []uint16 {
	result := make([]uint16, 0)
	for i, block := range b.blocks {
		for j := uint8(0); j < 64; j++ {
			if block&(1<<j) != 0 {
				result = append(result, uint16(i*64)+uint16(j))
			}
		}
	}
	return result
}

// positiveBits counts the number of positive bits in a given number.
func positiveBits(n uint64) uint16 {
	c := uint16(0)
	for n > 0 {
		n = n & (n - 1)
		c++
	}
	return c
}

// Max returns the BitArray that has the most set bits count.
func Max(b1, b2 *BitArray) *BitArray {
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
