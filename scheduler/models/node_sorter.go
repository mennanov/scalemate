package models

import (
	"math"
	"sort"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
)

// NodeSorter is an interface that is used to sort Nodes.
type NodeSorter interface {
	sort.Interface
	// BucketLength returns the length of the bucket of sorted Nodes for the given difference percentage.
	// The metric used to calculate the difference is implementation specific.
	BucketLength(differencePercentage int) int
}

// NodesByPrice sorts the Nodes by price asc.
type NodesByPrice struct {
	nodes   []*NodeExt
	request *ResourceRequest
}

func (n *NodesByPrice) Len() int {
	return len(n.nodes)
}

func (n *NodesByPrice) nodePrice(node *NodeExt) uint64 {
	return uint64(n.request.Cpu)*node.CpuPrice + uint64(n.request.Gpu)*node.GpuPrice +
		uint64(n.request.Disk)*node.DiskPrice + uint64(n.request.Memory)*node.MemoryPrice
}

func (n *NodesByPrice) Less(i, j int) bool {
	return n.nodePrice(n.nodes[i]) < n.nodePrice(n.nodes[j])
}

func (n *NodesByPrice) Swap(i, j int) {
	n.nodes[i], n.nodes[j] = n.nodes[j], n.nodes[i]
}

// BucketLength returns the length of the bucket of Nodes sorted by price for the given difference percentage.
func (n *NodesByPrice) BucketLength(differencePercentage int) int {
	if len(n.nodes) == 0 {
		return 0
	}
	boundaryPrice := float64(n.nodePrice(n.nodes[0])) * (float64(100+differencePercentage) / 100)

	return sort.Search(len(n.nodes), func(i int) bool {
		return float64(n.nodePrice(n.nodes[i])) >= boundaryPrice
	})
}

// Compile time interface check.
var _ NodeSorter = new(NodesByPrice)

// NodesByWorkload sorts the Nodes by their relative availability.
// The least busy Nodes come first.
type NodesByWorkload struct {
	nodes []*NodeExt
}

func (n *NodesByWorkload) Len() int {
	return len(n.nodes)
}

// nodeWorkloadScore computes an availability score for the Node. The higher the score the higher the workload is.
// Square is used to treat Nodes with higher capacity less busy.
func nodeWorkloadScore(node *NodeExt) float64 {
	result := float64(0)
	if node.CpuCapacity != 0 {
		result += math.Pow(float64(node.CpuAvailable)/float64(node.CpuCapacity), 2)
	}
	if node.GpuCapacity != 0 {
		result += math.Pow(float64(node.GpuAvailable)/float64(node.GpuCapacity), 2)
	}
	if node.DiskCapacity != 0 {
		result += math.Pow(float64(node.DiskAvailable)/float64(node.DiskCapacity), 2)
	}
	if node.MemoryCapacity != 0 {
		result += math.Pow(float64(node.MemoryAvailable)/float64(node.MemoryCapacity), 2)
	}
	return result
}

func (n *NodesByWorkload) Less(i, j int) bool {
	return nodeWorkloadScore(n.nodes[i]) > nodeWorkloadScore(n.nodes[j])
}

func (n *NodesByWorkload) Swap(i, j int) {
	n.nodes[i], n.nodes[j] = n.nodes[j], n.nodes[i]
}

// BucketLength returns the length of the bucket of Nodes sorted by workload for the given difference percentage.
func (n *NodesByWorkload) BucketLength(differencePercentage int) int {
	if len(n.nodes) == 0 {
		return 0
	}
	boundaryScore := nodeWorkloadScore(n.nodes[0]) * (float64(differencePercentage) / 100)

	return sort.Search(len(n.nodes), func(i int) bool {
		return nodeWorkloadScore(n.nodes[i]) <= boundaryScore
	})
}

// Compile time interface check.
var _ NodeSorter = new(NodesByWorkload)

// NodesByReliability sorts the Nodes by their reliability.
type NodesByReliability struct {
	nodes []*NodeExt
}

func (n *NodesByReliability) Len() int {
	return len(n.nodes)
}

// nodeReliabilityScore computes a reliability score for the Node. The smaller the score the higher reliability is.
func nodeReliabilityScore(node *NodeExt) float64 {
	if node.ContainersScheduled == 0 {
		// Best guess for the new Nodes with no scheduled containers yet.
		return 0.5
	}
	return float64(node.ContainersNodeFailed) / float64(node.ContainersScheduled)
}

func (n *NodesByReliability) Less(i, j int) bool {
	return nodeReliabilityScore(n.nodes[i]) < nodeReliabilityScore(n.nodes[j])
}

func (n *NodesByReliability) Swap(i, j int) {
	n.nodes[i], n.nodes[j] = n.nodes[j], n.nodes[i]
}

// BucketLength returns the length of the bucket of Nodes sorted by reliability for the given difference percentage.
func (n *NodesByReliability) BucketLength(differencePercentage int) int {
	if len(n.nodes) == 0 {
		return 0
	}
	boundaryScore := nodeReliabilityScore(n.nodes[0]) * (float64(100+differencePercentage) / 100)

	return sort.Search(len(n.nodes), func(i int) bool {
		return nodeReliabilityScore(n.nodes[i]) >= boundaryScore
	})
}

// SortNodesByStrategy sorts the Nodes multiple times according to the given scheduling strategies.
func SortNodesByStrategy(nodes []*NodeExt, strategies []*scheduler_proto.SchedulingStrategy, request *ResourceRequest) {
	j := len(nodes)
	var sorter NodeSorter
	for i, strategy := range strategies {
		switch strategy.Strategy {
		case scheduler_proto.SchedulingStrategy_MOST_RELIABLE:
			sorter = &NodesByReliability{nodes: nodes[:j]}
		case scheduler_proto.SchedulingStrategy_LEAST_BUSY:
			sorter = &NodesByWorkload{nodes: nodes[:j]}
		default: // CHEAPEST
			sorter = &NodesByPrice{nodes: nodes[:j], request: request}
		}
		sort.Sort(sorter)
		if i < len(strategies)-1 {
			// Only compute the right boundary for the non-last strategy.
			j = sorter.BucketLength(int(strategy.DifferencePercentage))
		}
	}
}
