package bitarray_test

import (
	"testing"

	"github.com/mennanov/scalemate/scheduler/bitarray"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBitArray_Or(t *testing.T) {
	ba1 := bitarray.NewBitArray(128)
	ba1.SetBit(0)
	ba1.SetBit(4)
	ba1.SetBit(64)
	ba1.SetBit(68)
	require.Equal(t, uint64(4), ba1.BitsSet)
	ba2 := bitarray.NewBitArray(128)
	ba2.SetBit(2)
	ba2.SetBit(3)
	ba2.SetBit(5)
	ba2.SetBit(127)
	require.Equal(t, uint64(4), ba2.BitsSet)
	b := ba1.Or(ba2)
	assert.Equal(t, uint64(8), b.BitsSet)
	assert.Equal(t, []uint64{0, 2, 3, 4, 5, 64, 68, 127}, b.SetBits())
}

func TestBitArray_SetBits(t *testing.T) {
	b := bitarray.NewBitArray(128)
	b.SetBit(0)
	b.SetBit(32)
	b.SetBit(80)
	b.SetBit(120)
	assert.Equal(t, []uint64{0, 32, 80, 120}, b.SetBits())
}

func TestBitArray_SetAndGetBit(t *testing.T) {
	b := bitarray.NewBitArray(128)
	for i := uint64(0); i < 128; i++ {
		if i%2 == 0 {
			b.SetBit(i)
		}
	}
	for i := uint64(0); i < 128; i++ {
		if i%2 == 0 {
			assert.True(t, b.GetBit(i))
		} else {
			assert.False(t, b.GetBit(i))
		}
	}
}

func TestMaxBitArray(t *testing.T) {
	t.Run("left array greater", func(t *testing.T) {
		b1 := bitarray.NewBitArray(128)
		b1.SetBit(0)
		b1.SetBit(45)
		b1.SetBit(120)
		b2 := bitarray.NewBitArray(128)
		b2.SetBit(63)
		b2.SetBit(120)
		assert.Equal(t, b1, bitarray.MaxBitArray(b1, b2))
	})

	t.Run("right array greater", func(t *testing.T) {
		b1 := bitarray.NewBitArray(128)
		b1.SetBit(0)
		b2 := bitarray.NewBitArray(128)
		b2.SetBit(63)
		b2.SetBit(120)
		assert.Equal(t, b2, bitarray.MaxBitArray(b1, b2))
	})

	t.Run("arrays same size", func(t *testing.T) {
		b1 := bitarray.NewBitArray(128)
		b1.SetBit(0)
		b2 := bitarray.NewBitArray(128)
		b2.SetBit(120)
		assert.Equal(t, b2, bitarray.MaxBitArray(b1, b2))
	})

	t.Run("left array is nil", func(t *testing.T) {
		b := bitarray.NewBitArray(128)
		b.SetBit(120)
		assert.Equal(t, b, bitarray.MaxBitArray(nil, b))
	})

	t.Run("right array is nil", func(t *testing.T) {
		b := bitarray.NewBitArray(128)
		b.SetBit(120)
		assert.Equal(t, b, bitarray.MaxBitArray(b, nil))
	})

	t.Run("both arrays nil", func(t *testing.T) {
		assert.Nil(t, bitarray.MaxBitArray(nil, nil))
	})

}
