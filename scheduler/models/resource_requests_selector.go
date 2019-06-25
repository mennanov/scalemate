package models

import (
	"github.com/mennanov/scalemate/shared/utils/bitarray"
)

// AvailableResources is a representation of resources constraints used for Containers selection.
type AvailableResources struct {
	Cpu    uint32
	Memory uint32
	Gpu    uint32
	Disk   uint32
}

type selectRequestsCacheKey struct {
	index uint16
	res   AvailableResources
}

// selectRequestsCache is a memoization cache used in the recursive function below.
type selectRequestsCache map[selectRequestsCacheKey]*bitarray.BitArray

// SelectResourceRequests finds the best combination of Containers that can be fitted into the given resources.
// This is a recursive dynamic programming algorithm with memoization.
// Time complexity is ~O(N^3), memory complexity is ~O(N^2): measured manually by running benchmarks.
func SelectResourceRequests(requests []ResourceRequest, res AvailableResources) *bitarray.BitArray {
	return selectRequests(requests, res, 0, make(map[selectRequestsCacheKey]*bitarray.BitArray))
}

func selectRequests(requests []ResourceRequest, res AvailableResources, i uint16, cache selectRequestsCache) *bitarray.BitArray {
	n := uint16(len(requests))
	if n == 0 {
		return nil
	}
	if c, ok := cache[selectRequestsCacheKey{index: i, res: res}]; ok {
		return c
	}

	request := requests[i]
	var requestsWithCurrent *bitarray.BitArray
	var requestsWithoutCurrent *bitarray.BitArray

	if requestFitsResources(request, res) {
		requestsWithCurrent = bitarray.NewBitArray(n)
		requestsWithCurrent.SetBit(i)
		if i < n-1 {
			subRequests := selectRequests(requests, reducedResources(request, res), i+1, cache)
			if subRequests != nil {
				requestsWithCurrent = requestsWithCurrent.Or(subRequests)
			}
		}
	}

	// If the length of the combination _with_ the current request is the max possible - don't bother calculating the
	// combination without it.
	if i < n-1 && (requestsWithCurrent == nil || requestsWithCurrent.BitsSet < n-i) {
		requestsWithoutCurrent = selectRequests(requests, res, i+1, cache)
	}

	max := bitarray.Max(requestsWithCurrent, requestsWithoutCurrent)
	cache[selectRequestsCacheKey{index: i, res: res}] = max
	return max
}

func requestFitsResources(request ResourceRequest, res AvailableResources) bool {
	return request.Cpu <= res.Cpu && request.Memory <= res.Memory && request.Gpu <= res.Gpu && request.Disk <= res.Disk
}

func reducedResources(request ResourceRequest, res AvailableResources) AvailableResources {
	r := AvailableResources{
		Cpu:    res.Cpu - request.Cpu,
		Memory: res.Memory - request.Memory,
		Disk:   res.Disk - request.Disk,
		Gpu:    res.Gpu - request.Gpu,
	}
	return r
}
