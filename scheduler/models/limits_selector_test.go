package models

import (
	"testing"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SelectLimits(t *testing.T) {
	for _, testCase := range []struct {
		limits         []Limit
		res            AvailableResources
		limitsExpected []uint16
	}{
		{
			limits: []Limit{},
			res: AvailableResources{
				Cpu: 2,
				Gpu: 2,
			},
			limitsExpected: []uint16{},
		},
		{
			limits: []Limit{
				{
					Limit: scheduler_proto.Limit{
						Cpu: 2,
						Gpu: 2,
					},
				},
			},
			res: AvailableResources{
				Cpu: 2,
				Gpu: 2,
			},
			limitsExpected: []uint16{0},
		},
		{
			limits: []Limit{
				{
					Limit: scheduler_proto.Limit{
						Cpu: 2,
						Gpu: 2,
					},
				},
				{
					Limit: scheduler_proto.Limit{
						Cpu: 1,
						Gpu: 1,
					},
				},
				{
					Limit: scheduler_proto.Limit{
						Cpu: 1,
						Gpu: 1,
					},
				},
			},
			res: AvailableResources{
				Cpu: 2,
				Gpu: 2,
			},
			limitsExpected: []uint16{1, 2},
		},
		{
			limits: []Limit{
				{
					Limit: scheduler_proto.Limit{
						Cpu: 2,
						Gpu: 1,
					},
				},
				{
					Limit: scheduler_proto.Limit{
						Cpu: 1,
						Gpu: 2,
					},
				},
				{
					Limit: scheduler_proto.Limit{
						Cpu: 1,
						Gpu: 1,
					},
				},
				{
					Limit: scheduler_proto.Limit{
						Cpu: 1,
						Gpu: 0,
					},
				},
				{
					Limit: scheduler_proto.Limit{
						Cpu: 0,
						Gpu: 1,
					},
				},
				{
					Limit: scheduler_proto.Limit{
						Cpu: 1,
						Gpu: 1,
					},
				},
			},
			res: AvailableResources{
				Cpu: 4,
				Gpu: 4,
			},
			limitsExpected: []uint16{2, 3, 4, 5},
		},
	} {
		actualLimits := SelectLimits(testCase.limits, testCase.res)
		if len(testCase.limitsExpected) == 0 {
			assert.Nil(t, actualLimits)
		} else {
			require.Equal(t, len(testCase.limitsExpected), int(actualLimits.BitsSet))
			assert.Equal(t, testCase.limitsExpected, actualLimits.SetBits())

		}
	}
}
