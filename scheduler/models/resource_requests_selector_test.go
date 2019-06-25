package models

import (
	"testing"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SelectResourceRequests(t *testing.T) {
	for _, testCase := range []struct {
		requests         []ResourceRequest
		res              AvailableResources
		requestsExpected []uint16
	}{
		{
			requests: []ResourceRequest{},
			res: AvailableResources{
				Cpu: 2,
				Gpu: 2,
			},
			requestsExpected: []uint16{},
		},
		{
			requests: []ResourceRequest{
				{
					ResourceRequest: scheduler_proto.ResourceRequest{
						Cpu: 2,
						Gpu: 2,
					},
				},
			},
			res: AvailableResources{
				Cpu: 2,
				Gpu: 2,
			},
			requestsExpected: []uint16{0},
		},
		{
			requests: []ResourceRequest{
				{
					ResourceRequest: scheduler_proto.ResourceRequest{
						Cpu: 2,
						Gpu: 2,
					},
				},
				{
					ResourceRequest: scheduler_proto.ResourceRequest{
						Cpu: 1,
						Gpu: 1,
					},
				},
				{
					ResourceRequest: scheduler_proto.ResourceRequest{
						Cpu: 1,
						Gpu: 1,
					},
				},
			},
			res: AvailableResources{
				Cpu: 2,
				Gpu: 2,
			},
			requestsExpected: []uint16{1, 2},
		},
		{
			requests: []ResourceRequest{
				{
					ResourceRequest: scheduler_proto.ResourceRequest{
						Cpu: 2,
						Gpu: 1,
					},
				},
				{
					ResourceRequest: scheduler_proto.ResourceRequest{
						Cpu: 1,
						Gpu: 2,
					},
				},
				{
					ResourceRequest: scheduler_proto.ResourceRequest{
						Cpu: 1,
						Gpu: 1,
					},
				},
				{
					ResourceRequest: scheduler_proto.ResourceRequest{
						Cpu: 1,
						Gpu: 0,
					},
				},
				{
					ResourceRequest: scheduler_proto.ResourceRequest{
						Cpu: 0,
						Gpu: 1,
					},
				},
				{
					ResourceRequest: scheduler_proto.ResourceRequest{
						Cpu: 1,
						Gpu: 1,
					},
				},
			},
			res: AvailableResources{
				Cpu: 4,
				Gpu: 4,
			},
			requestsExpected: []uint16{2, 3, 4, 5},
		},
	} {
		actualResourceRequests := SelectResourceRequests(testCase.requests, testCase.res)
		if len(testCase.requestsExpected) == 0 {
			assert.Nil(t, actualResourceRequests)
		} else {
			require.Equal(t, len(testCase.requestsExpected), int(actualResourceRequests.BitsSet))
			assert.Equal(t, testCase.requestsExpected, actualResourceRequests.SetBits())
		}
	}
}
