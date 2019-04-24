package models

import (
	"github.com/mennanov/scalemate/scheduler/bitarray"
)

// AvailableResources is a representation of resources constraints used for Containers selection.
type AvailableResources struct {
	Cpu    uint32
	Memory uint32
	Gpu    uint32
	Disk   uint32
}

type selectLimitsCacheKey struct {
	index uint64
	res   AvailableResources
}

// selectLimitsCache is a memoization cache used in the recursive function below.
type selectLimitsCache map[selectLimitsCacheKey]*bitarray.BitArray

// SelectLimits finds the best combination of Containers that can be fitted into the given resources.
// This is a recursive dynamic programming algorithm with memoization.
// Time complexity is ~O(N^3), memory complexity is ~O(N^2): measured manually by running benchmarks.
func SelectLimits(limits []Limit, res AvailableResources) *bitarray.BitArray {
	return selectLimits(limits, res, 0, make(map[selectLimitsCacheKey]*bitarray.BitArray))
}

func selectLimits(limits []Limit, res AvailableResources, i uint64, cache selectLimitsCache) *bitarray.BitArray {
	n := uint64(len(limits))
	if n == 0 {
		return nil
	}
	if c, ok := cache[selectLimitsCacheKey{index: i, res: res}]; ok {
		return c
	}

	limit := limits[i]
	var limitsWithCurrent *bitarray.BitArray
	var limitsWithoutCurrent *bitarray.BitArray

	if limitFitsResources(limit, res) {
		limitsWithCurrent = bitarray.NewBitArray(n)
		limitsWithCurrent.SetBit(i)
		if i < n-1 {
			subLimits := selectLimits(limits, reducedResources(limit, res), i+1, cache)
			if subLimits != nil {
				limitsWithCurrent = limitsWithCurrent.Or(subLimits)
			}
		}
	}

	// If the length of the combination _with_ the current limit is the max possible - don't bother calculating the
	// combination without it.
	if i < n-1 && (limitsWithCurrent == nil || limitsWithCurrent.BitsSet < n-i) {
		limitsWithoutCurrent = selectLimits(limits, res, i+1, cache)
	}

	max := bitarray.MaxBitArray(limitsWithCurrent, limitsWithoutCurrent)
	cache[selectLimitsCacheKey{index: i, res: res}] = max
	return max
}

func limitFitsResources(limit Limit, res AvailableResources) bool {
	return limit.Cpu <= res.Cpu && limit.Memory <= res.Memory && limit.Gpu <= res.Gpu && limit.Disk <= res.Disk
}

func reducedResources(limit Limit, res AvailableResources) AvailableResources {
	r := AvailableResources{
		Cpu:    res.Cpu - limit.Cpu,
		Memory: res.Memory - limit.Memory,
		Disk:   res.Disk - limit.Disk,
		Gpu:    res.Gpu - limit.Gpu,
	}
	return r
}
