package models

import (
	"github.com/mennanov/scalemate/scheduler/bitarray"
)

// AvailableResources is a representation of resources constraints used for Jobs selection.
type AvailableResources struct {
	CpuAvailable    float32
	MemoryAvailable uint32
	GpuAvailable    uint32
	DiskAvailable   uint32
}

type selectJobsCacheKey struct {
	index uint64
	res   AvailableResources
}

// selectJobsCache is a memoization cache used in the recursive function below.
type selectJobsCache map[selectJobsCacheKey]*bitarray.BitArray

// SelectJobs finds the best combination of Jobs that can be fitted into the given resources.
// This is a recursive dynamic programming algorithm with memoization.
// Time complexity is ~O(N^3), memory complexity is ~O(N^2): measured manually by running benchmarks.
func SelectJobs(jobs []Job, res AvailableResources) *bitarray.BitArray {
	return selectJobs(jobs, res, 0, make(map[selectJobsCacheKey]*bitarray.BitArray))
}

func selectJobs(jobs []Job, res AvailableResources, i uint64, cache selectJobsCache) *bitarray.BitArray {
	n := uint64(len(jobs))
	if n == 0 {
		return nil
	}
	if c, ok := cache[selectJobsCacheKey{index: i, res: res}]; ok {
		return c
	}

	job := jobs[i]
	var jobsWithCurrent *bitarray.BitArray
	var jobsWithoutCurrent *bitarray.BitArray

	if jobFitsInResources(job, res) {
		jobsWithCurrent = bitarray.NewBitArray(n)
		jobsWithCurrent.SetBit(i)
		if i < n-1 {
			subJobs := selectJobs(jobs, updatedResources(job, res), i+1, cache)
			if subJobs != nil {
				jobsWithCurrent = jobsWithCurrent.Or(subJobs)
			}
		}
	}

	// If the length of the combination _with_ the current Job is the max possible - don't bother calculating the
	// combination without it.
	if i < n-1 && (jobsWithCurrent == nil || jobsWithCurrent.BitsSet < n-i) {
		jobsWithoutCurrent = selectJobs(jobs, res, i+1, cache)
	}

	max := bitarray.MaxBitArray(jobsWithCurrent, jobsWithoutCurrent)
	cache[selectJobsCacheKey{index: i, res: res}] = max
	return max
}

func jobFitsInResources(job Job, res AvailableResources) bool {
	if job.CpuLimit > res.CpuAvailable {
		return false
	}
	if job.MemoryLimit > res.MemoryAvailable {
		return false
	}
	if job.GpuLimit > res.GpuAvailable {
		return false
	}
	if job.DiskLimit > res.DiskAvailable {
		return false
	}
	return true
}

func updatedResources(job Job, res AvailableResources) AvailableResources {
	r := AvailableResources{
		CpuAvailable:    res.CpuAvailable - job.CpuLimit,
		MemoryAvailable: res.MemoryAvailable - job.MemoryLimit,
		DiskAvailable:   res.DiskAvailable - job.DiskLimit,
	}
	if job.GpuLimit > 0 {
		r.GpuAvailable = res.GpuAvailable - job.GpuLimit
	}
	return r
}
