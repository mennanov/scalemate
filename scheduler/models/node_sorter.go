package models

import "math"

type NodeWithPricing struct {
	*Node
	*NodePricing
}

type NodesByPrice struct {
	nodes   []*NodeWithPricing
	request *ResourceRequest
}

func (n *NodesByPrice) Len() int {
	return len(n.nodes)
}

func (n *NodesByPrice) nodePrice(pricing *NodePricing) uint64 {
	return uint64(n.request.Cpu)*pricing.CpuPrice + uint64(n.request.Gpu)*pricing.GpuPrice +
		uint64(n.request.Disk)*pricing.DiskPrice + uint64(n.request.Memory)*pricing.MemoryPrice
}

func (n *NodesByPrice) Less(i, j int) bool {
	return n.nodePrice(n.nodes[i].NodePricing) < n.nodePrice(n.nodes[j].NodePricing)
}

func (n *NodesByPrice) Swap(i, j int) {
	n.nodes[i], n.nodes[j] = n.nodes[j], n.nodes[i]
}

type NodesByAvailability struct {
	nodes []*Node
}

func (n *NodesByAvailability) Len() int {
	return len(n.nodes)
}

// nodeAvailabilityScore computes an availability score for the Node. The higher the score the higher availability is.
func nodeAvailabilityScore(node *Node) float64 {
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

func (n *NodesByAvailability) Less(i, j int) bool {
	return nodeAvailabilityScore(n.nodes[i]) > nodeAvailabilityScore(n.nodes[j])
}

func (n *NodesByAvailability) Swap(i, j int) {
	n.nodes[i], n.nodes[j] = n.nodes[j], n.nodes[i]
}

type NodesByReliability struct {
	nodes []*Node
}

func (n *NodesByReliability) Len() int {
	return len(n.nodes)
}

// nodeReliabilityScore computes a reliability score for the Node. The smaller the score the higher reliability is.
func nodeReliabilityScore(node *Node) float64 {
	if node.ContainersFinished == 0 {
		// Treat Nodes with no containers
		return math.Inf(1)
	}
	return float64(node.ContainersFailed) / float64(node.ContainersFinished)
}

func (n *NodesByReliability) Less(i, j int) bool {
	return nodeReliabilityScore(n.nodes[i]) < nodeReliabilityScore(n.nodes[j])
}

func (n *NodesByReliability) Swap(i, j int) {
	n.nodes[i], n.nodes[j] = n.nodes[j], n.nodes[i]
}
