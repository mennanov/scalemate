package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SelectJobs(t *testing.T) {
	for _, testCase := range []struct {
		jobs         []Container
		res          AvailableResources
		jobsExpected uint64
	}{
		{
			jobs: []Container{
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
				Cpu: 2.5,
			},
			jobsExpected: 3,
		},
		{
			jobs: []Container{
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
				Cpu: 1.5,
			},
			jobsExpected: 4,
		},
		{
			jobs: []Container{
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
				Cpu: 1.5,
			},
			jobsExpected: 6,
		},
		{
			jobs: []Container{
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
				Cpu: 1,
			},
			jobsExpected: 0,
		},
		{
			jobs: []Container{
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
				Cpu: 1.5,
			},
			jobsExpected: 1,
		},
		{
			jobs:         []Container{},
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
