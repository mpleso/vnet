package ip

import (
	"github.com/platinasystems/elib"
	"github.com/platinasystems/vnet"
)

// Next node index stored in ip adjacencies.
type LookupNext uint32

const (
	// Packet does not match any route in table.
	LookupMiss LookupNext = iota

	// Adjacency says to drop or punt this packet.
	LookupDrop
	LookupPunt

	// This packet matches an IP address of one of our interfaces.
	LookupLocal

	// This packet matches an "interface route" and packets
	// need to be passed to ARP to find rewrite string for
	// this destination.
	LookupArp

	// This packet is to be rewritten and forwarded to the next
	// processing node.  This is typically the output interface but
	// might be another node for further output processing.
	LookupRewrite

	LookupNNext
)

type Adjacency struct {
	// Id in adjacency heap used to free adjacency.
	heapId elib.Index

	// Interface address index for local/arp adjacency.
	ifAddr

	// Number of adjecencies in block.  Greater than 1 means multipath; otherwise equal to 1.
	nAdj uint16

	// Local adjacencies: specifies next node from ip4 local.
	localNextIndex uint16

	rw vnet.Rewrite
}

// Index into adjacency table.
type Adj uint32

// Miss adjacency is always first in adjacency table.
const MissAdj Adj = 0

//go:generate gentemplate -d Package=ip -id adjacencyHeap -d HeapType=adjacencyHeap -d Data=elts -d Type=Adjacency github.com/platinasystems/elib/heap.tmpl

type adjacencyMain struct {
	adjacencyHeap
}

type weightedAdj struct {
	adj    Adj
	weight uint32
}

type adjBlock struct {
	// Heap offset of first element of block.
	offset uint32
	// Heap id (used when blocks is freed).
	id elib.Index
	// Block size.
	size uint32
}

type multipathAdjacency struct {
	// Index of first adjacency in block.
	first Adj

	// Power of 2 size of block.
	nAdj uint32

	// Number of prefixes that point to this adjacency.
	referenceCount uint32

	normalized   adjBlock
	unnormalized adjBlock
}
