package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_SelectJobs(t *testing.T) {
	for _, testCase := range []struct {
		jobs         []Job
		res          AvailableResources
		jobsExpected uint64
	}{
		{
			jobs: []Job{
				{
					CpuLimit: 2,
				},
				{
					CpuLimit: 1.5,
				},
				{
					CpuLimit: 0.5,
				},
				{
					CpuLimit: 0.5,
				},
			},
			res: AvailableResources{
				CpuAvailable: 2.5,
			},
			jobsExpected: 3,
		},
		{
			jobs: []Job{
				{
					CpuLimit: 2,
				},
				{
					CpuLimit: 1.5,
				},
				{
					CpuLimit: 0.5,
				},
				{
					CpuLimit: 0.25,
				},
				{
					CpuLimit: 1.0,
				},
				{
					CpuLimit: 0.25,
				},
				{
					CpuLimit: 0.5,
				},
				{
					CpuLimit: 0.1,
				},
			},
			res: AvailableResources{
				CpuAvailable: 1.5,
			},
			jobsExpected: 4,
		},
		{
			jobs: []Job{
				{
					CpuLimit: 2,
				},
				{
					CpuLimit: 1.5,
				},
				{
					CpuLimit: 0.5,
				},
				{
					CpuLimit: 0.25,
				},
				{
					CpuLimit: 1.0,
				},
				{
					CpuLimit: 0.25,
				},
				{
					CpuLimit: 0.5,
				},
				{
					CpuLimit: 0.1,
				},
				{
					CpuLimit: 0.2,
				},
				{
					CpuLimit: 0.2,
				},
			},
			res: AvailableResources{
				CpuAvailable: 1.5,
			},
			jobsExpected: 6,
		},
		{
			jobs: []Job{
				{
					CpuLimit: 2,
				},
				{
					CpuLimit: 1.5,
				},
				{
					CpuLimit: 2.5,
				},
			},
			res: AvailableResources{
				CpuAvailable: 1,
			},
			jobsExpected: 0,
		},
		{
			jobs: []Job{
				{
					CpuLimit: 2,
				},
				{
					CpuLimit: 1.5,
				},
				{
					CpuLimit: 2.5,
				},
			},
			res: AvailableResources{
				CpuAvailable: 1.5,
			},
			jobsExpected: 1,
		},
		{
			jobs:         []Job{},
			res:          AvailableResources{},
			jobsExpected: 0,
		},
	} {
		actualJobs := SelectJobs(testCase.jobs, testCase.res)
		if testCase.jobsExpected == 0 {
			assert.Nil(t, actualJobs)
		} else {
			assert.Equal(t, testCase.jobsExpected, actualJobs.BitsSet)
		}
	}
}

func Test_SelectJobsPerformance(t *testing.T) {
	// Stress test: input size is 10 times bigger than the expected.
	n := JobsScheduledForNodeQueryLimit * 10
	jobs := make([]Job, 0, n)
	var c float32
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			c = 0.5
		} else {
			c = 1
		}
		jobs = append(jobs, Job{CpuLimit: c})
	}
	res := AvailableResources{CpuAvailable: float32(n / 2)}
	start := time.Now()
	actualJobs := SelectJobs(jobs, res)
	assert.True(t, time.Since(start) < time.Second*3)
	assert.Equal(t, uint64(float32(n)*0.75), actualJobs.BitsSet)
}
