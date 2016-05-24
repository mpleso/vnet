package ip

import (
	"github.com/platinasystems/elib"
	"github.com/platinasystems/vnet"

	"math"
	"sort"
	"unsafe"
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
const (
	AdjMiss Adj = 0
	AdjNil  Adj = ^Adj(0)
)

//go:generate gentemplate -d Package=ip -id adjacencyHeap -d HeapType=adjacencyHeap -d Data=elts -d Type=Adjacency github.com/platinasystems/elib/heap.tmpl
//go:generate gentemplate -d Package=ip -id AdjVec -d VecType=AdjVec -d Type=Adj github.com/platinasystems/elib/vec.tmpl

type adjacencyThread struct {
	// Packet/byte counters for each adjacency.
	counters vnet.CombinedCounters
}

type adjacencyMain struct {
	adjacencyHeap

	multipathMain multipathMain

	Remap func(nRemaps uint, remaps AdjVec)

	threads []*adjacencyThread
}

// A next hop in a multipath.
type nextHop struct {
	// Adjacency index for next hop's rewrite.
	adj Adj

	// Relative weight for this next hop.
	weight uint32
}

//go:generate gentemplate -d Package=ip -id nextHopHeap -d HeapType=nextHopHeap -d Data=elts -d Type=nextHop github.com/platinasystems/elib/heap.tmpl
//go:generate gentemplate -d Package=ip -id nextHopVec -d VecType=nextHopVec -d Type=nextHop github.com/platinasystems/elib/vec.tmpl
//go:generate gentemplate -d Package=ip -id multipathAdjacencyVec -d VecType=multipathAdjacencyVec -d Type=multipathAdjacency github.com/platinasystems/elib/vec.tmpl

type multipathMain struct {
	cachedNextHopVec [2]nextHopVec

	multipathErrorTolerance float64
	nextHopHash             elib.Hash
	nextHopHeapOffsets      []uint32

	nextHopHeap

	mpAdjVec multipathAdjacencyVec
}

func (m *multipathMain) GetNextHops(i uint) []nextHop {
	return m.nextHopHeap.Slice(uint(m.nextHopHeapOffsets[i]))
}
func (m *multipathMain) HashIndex(s *elib.HashState, i uint) {
	nextHopVec(m.GetNextHops(i)).HashKey(s)
}
func (m *multipathMain) HashResize(newCap uint, rs []elib.HashResizeCopy) {
	src, dst := m.nextHopHeapOffsets, make([]uint32, newCap)
	for i := range rs {
		dst[rs[i].Dst] = src[rs[i].Src]
	}
	m.nextHopHeapOffsets = dst
}

func (a nextHopVec) HashKey(s *elib.HashState) {
	s.HashPointer(unsafe.Pointer(&a[0]), uintptr(a.Len())*unsafe.Sizeof(a[0]))
}
func (a nextHopVec) Equal(b nextHopVec) bool {
	if la, lb := a.Len(), b.Len(); la != lb {
		return false
	} else {
		for i := uint(0); i < la; i++ {
			if a[i].Compare(&b[i]) == 0 {
				return false
			}
		}
	}
	return true
}
func (a nextHopVec) HashKeyEqual(h elib.Hasher, i uint) bool {
	b := nextHopVec(h.(*multipathMain).GetNextHops(i))
	return a.Equal(b)
}

type nextHopSort nextHopVec

func (a nextHopSort) Len() int           { return len(a) }
func (a nextHopSort) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a nextHopSort) Less(i, j int) bool { return a[i].Compare(&a[j]) < 0 }

// Order by decreasing weight and increasing adj index for equal weights.
func (a *nextHop) Compare(b *nextHop) (cmp int) {
	cmp = int(b.weight) - int(a.weight)
	if cmp == 0 {
		cmp = int(a.adj) - int(b.adj)
	}
	return
}

// Normalize next hops: find a power of 2 sized block of next hops within error tolerance of given raw next hops.
func (raw nextHopVec) normalize(m *multipathMain, result *nextHopVec) (nAdj uint) {
	nRaw := raw.Len()

	if nRaw == 0 {
		return
	}

	// Allocate enough space for 2 copies; we'll use second copy to save original weights.
	t := *result
	t.Validate(2*nRaw - 1)
	// Save allocation for next caller.
	*result = t

	n := nRaw
	switch n {
	case 1:
		t[0] = raw[0]
		t[0].weight = 1
		t = t[:1]
		return
	case 2:
		cmp := 0
		if raw[0].Compare(&raw[1]) < 0 {
			cmp = 1
		}
		t[0], t[1] = raw[cmp], raw[cmp^1]
		if t[0].weight == t[1].weight {
			t[0].weight = 1
			t[1].weight = 1
			t = t[:2]
			return
		}

	default:
		copy(t, raw)
		sort.Sort(nextHopSort(t[0:n]))
	}

	// Find total weight to normalize weights.
	sumWeight := float64(0)
	for i := uint(0); i < n; i++ {
		sumWeight += float64(t[i].weight)
	}

	// In the unlikely case that all weights are given as 0, set them all to 1.
	if sumWeight == 0 {
		for i := uint(0); i < n; i++ {
			t[i].weight = 1
		}
		sumWeight = float64(n)
	}

	// Save copies of all next hop weights to avoid being overwritten in loop below.
	copy(t[n:], t[:n])

	if m.multipathErrorTolerance == 0 {
		m.multipathErrorTolerance = .01
	}

	// Try larger and larger power of 2 sized adjacency blocks until we
	// find one where traffic flows to within 1% of specified weights.
	nAdj = uint(elib.Word(n).MaxPow2())
	for {
		error := float64(0)
		norm := float64(nAdj) / sumWeight
		nLeft := nAdj

		i := uint(0)
		for ; i < n; i++ {
			nf := norm * float64(t[i].weight)
			n := uint(nf + .5) // round to nearest integer
			if n > nLeft {
				n = nLeft
			}
			nLeft -= n
			error += math.Abs(nf - float64(n))
			t[i].weight = uint32(n)
		}
		t[0].weight += uint32(nLeft)

		if error < m.multipathErrorTolerance*float64(nAdj) {
			// Truncate any next hops with zero weight.
			t = t[:i]
			break
		}

		// Try next power of 2 size.
		nAdj *= 2
	}
	return
}

func (m *multipathMain) allocNextHopBlock(b *nextHopBlock, key nextHopVec) {
	n := uint(len(key))
	o := m.nextHopHeap.Get(n)
	copy(m.nextHopHeap.Slice(o), key)
	b.size = uint32(n)
	b.offset = uint32(o)
}

func (m *multipathMain) freeNextHopBlock(b *nextHopBlock) {
	m.nextHopHeap.Put(uint(b.offset))
	b.offset = ^uint32(0)
	b.size = ^uint32(0)
}

func (m *multipathMain) getNextHopBlock(b *nextHopBlock) []nextHop {
	return m.nextHopHeap.Slice(uint(b.offset))
}

type nextHopBlock struct {
	// Heap offset of first next hop.
	offset uint32

	// Block size.
	size uint32
}

type multipathAdjacency struct {
	// Index of first adjacency in block.
	adj Adj

	// Power of 2 size of block.
	nAdj uint32

	// Number of prefixes that point to this adjacency.
	referenceCount uint32

	// Normalized next hops are used as hash keys: they are sorted by weight
	// and weights are chosen so they add up to nAdj (with zero-weighted next hops being deleted).
	normalizedNextHops nextHopBlock

	// Unnormalized next hops are saved so that control plane has a record of exactly
	// what the RIB told it.
	unnormalizedNextHops nextHopBlock
}

func (m *adjacencyMain) get(unnorm nextHopVec, create bool) (madj *multipathAdjacency, ok bool) {
	mp := &m.multipathMain
	nAdj := unnorm.normalize(mp, &mp.cachedNextHopVec[1])
	norm := mp.cachedNextHopVec[1]

	// Use normalized next hops to see if we've seen a block equivalent to this one before.
	if _, ok = mp.nextHopHash.Get(norm); ok || !create {
		return
	}

	// Copy next hops into adjacency block one for each weight.
	ai, as := m.New(nAdj, nil)
	i := uint(0)
	for nhi := range norm {
		nh := &norm[nhi]
		nextHopAdjacency := &m.adjacencyHeap.elts[nh.adj]
		for w := uint32(0); w < nh.weight; w++ {
			as[i] = *nextHopAdjacency
			as[i].nAdj = uint16(nAdj)
			i++
		}
	}

	aid := m.adjacencyHeap.Id(uint(ai))
	madj = mp.mpAdjVec.Validate(uint(aid))

	madj.adj = ai
	madj.nAdj = uint32(nAdj)
	madj.referenceCount = 0 // caller will set to 1

	mp.allocNextHopBlock(&madj.normalizedNextHops, norm)
	mp.allocNextHopBlock(&madj.unnormalizedNextHops, unnorm)

	i, _ = mp.nextHopHash.Set(norm)
	mp.nextHopHeapOffsets[i] = madj.normalizedNextHops.offset

	// fixme adj add del callbacks

	ok = true
	return
}
func (m *adjacencyMain) create(unnorm nextHopVec) (*multipathAdjacency, bool) {
	return m.get(unnorm, true)
}

func (m *adjacencyMain) mpAdjForAdj(a Adj) (ma *multipathAdjacency) {
	aid := uint(m.adjacencyHeap.Id(uint(a)))
	if aid < m.multipathMain.mpAdjVec.Len() {
		ma = &m.multipathMain.mpAdjVec[aid]
	}
	return
}

func (m *adjacencyMain) addDelNextHop(oldAdj Adj, isDel bool, nextHopAdj Adj, nextHopWeight uint) (newAdj Adj, ok bool) {
	mm := &m.multipathMain
	var (
		old, new *multipathAdjacency
		nhs      []nextHop
		nnh, nhi uint
	)

	if oldAdj != AdjNil {
		ma := m.mpAdjForAdj(oldAdj)
		if ma.normalizedNextHops.size > 0 {
			old = ma
			nhs = mm.getNextHopBlock(&old.unnormalizedNextHops)

			// Linear search for target nexthop.
			nhi, nnh = 0, uint(len(nhs))
			for nhi < nnh {
				if nhs[nhi].adj == nextHopAdj {
					break
				}
			}

			// For delete next hop must be found.
			if nhi >= nnh && isDel {
				return
			}
		}
	}

	hashNhs := mm.cachedNextHopVec[0]
	hashNhs.Validate(nnh + 1)
	mm.cachedNextHopVec[0] = hashNhs // save for next call

	if isDel {
		if nhi > 0 {
			copy(hashNhs[0:nhi], nhs[0:nhi])
		}
		if nhi+1 < nnh {
			copy(hashNhs[nhi:], nhs[nhi+1:nnh])
		}
		hashNhs = hashNhs[:nnh-1]
	} else {
		// If next hop is already there with the same weight, we have nothing to do.
		if nhi < nnh && nhs[nhi].weight == uint32(nextHopWeight) {
			newAdj = AdjNil
			ok = true
			return
		}

		// Copy old next hops to lookup key.
		copy(hashNhs, nhs)

		var nh *nextHop
		if nhi < nnh {
			// Change weight of existing next hop.
			nh = &hashNhs[nhi]
		} else {
			// Add a new next hop.
			nh = &hashNhs[nnh]
			nh.adj = nextHopAdj
		}
		nh.weight = uint32(nextHopWeight)
	}

	if len(hashNhs) > 0 {
		new, _ = m.create(hashNhs)
		if old != nil {
			// Fetch again since pool may have moved.
			old = m.mpAdjForAdj(oldAdj)
		}
	}

	if new != old {
		if old != nil {
			old.referenceCount--
		}
		if new != nil {
			new.referenceCount++
		}
	}
	if old != nil && old.referenceCount == 0 {
		old.free(m)
	}

	if new != nil {
		ok = true
		newAdj = new.adj
	}

	return
}

func (m *adjacencyMain) PoisonAdj(a Adj) {
	as := m.adjacencyHeap.Slice(uint(a))
	elib.PointerPoison(unsafe.Pointer(&as[0]), uintptr(len(as))*unsafe.Sizeof(as[0]))
}

func (m *adjacencyMain) FreeAdj(a Adj, delMultipath bool) {
	if delMultipath {
		m.delMultipath(a)
	}
	m.adjacencyHeap.Put(uint(a))
}

func (m *adjacencyMain) Del(a Adj) { m.FreeAdj(a, true) }

func (nhs nextHopVec) find(target Adj) (i uint, ok bool) {
	for i = 0; i < uint(len(nhs)); i++ {
		if ok = nhs[i].adj == target; ok {
			break
		}
	}
	return
}

func (m *multipathMain) del(nhs nextHopVec, result *nextHopVec, nhi uint) {
	r := *result
	nnh := uint(len(nhs))
	r.Validate(nnh)
	if nhi > 0 {
		copy(r[0:nhi], nhs[0:nhi])
	}
	if nhi+1 < nnh {
		copy(r[nhi:], nhs[nhi+1:nnh])
	}
	r = r[:nnh-1]
	*result = r
}

func (m *adjacencyMain) delMultipath(toDel Adj) {
	mm := &m.multipathMain

	remaps := make([]Adj, len(m.adjacencyHeap.elts))
	nRemaps := uint(0)
	for i := range mm.mpAdjVec {
		ma := &mm.mpAdjVec[i]
		if !ma.isValid() {
			continue
		}
		nhs := nextHopVec(mm.getNextHopBlock(&ma.unnormalizedNextHops))
		var (
			ok  bool
			nhi uint
		)
		if nhi, ok = nhs.find(toDel); !ok {
			continue
		}

		var newMa *multipathAdjacency
		if len(nhs) > 1 {
			mm.del(nhs, &mm.cachedNextHopVec[0], nhi)
			hashNhs := mm.cachedNextHopVec[0]
			newMa, _ := m.get(hashNhs, true)
			if newMa == ma { // fixme
				continue
			}
		}

		remaps[ma.adj] = AdjNil
		if newMa != nil {
			remaps[ma.adj] = 1 + newMa.adj
			nRemaps++
		}
		ma.free(m)
	}

	m.Remap(nRemaps, remaps)
}

func (a *multipathAdjacency) invalidate()   { a.nAdj = 0 }
func (a *multipathAdjacency) isValid() bool { return a.nAdj != 0 }

func (ma *multipathAdjacency) free(m *adjacencyMain) {
	mm := &m.multipathMain
	nhs := nextHopVec(mm.getNextHopBlock(&ma.unnormalizedNextHops))
	i, ok := mm.nextHopHash.Unset(nhs)
	if !ok {
		panic("unknown multipath adjacency")
	}
	mm.nextHopHeapOffsets[i] = 0
	mm.freeNextHopBlock(&ma.unnormalizedNextHops)
	mm.freeNextHopBlock(&ma.normalizedNextHops)

	// fixme adj add del callbacks

	m.PoisonAdj(ma.adj)
	m.FreeAdj(ma.adj, ma.referenceCount == 0)

	// Nothing to free since multipath adjacencies are indexed by adjacency index.

	ma.invalidate()
}

func (m *adjacencyMain) validateCounter(a Adj) {
	for _, t := range m.threads {
		t.counters.Validate(uint(a))
	}
}
func (m *adjacencyMain) clearCounter(a Adj) {
	for _, t := range m.threads {
		t.counters.Clear(uint(a))
	}
}

func (m *adjacencyMain) Get(a Adj) (as []Adjacency) { return m.adjacencyHeap.Slice(uint(a)) }

func (m *adjacencyMain) New(n uint, template *Adjacency) (ai Adj, as []Adjacency) {
	ai = Adj(m.adjacencyHeap.Get(n))
	m.validateCounter(ai)
	as = m.Get(ai)
	for i := range as {
		if template != nil {
			as[i] = *template
		}
		as[i].rw.Si = vnet.SiNil
		as[i].nAdj = uint16(n)
		m.clearCounter(ai + Adj(i))
	}
	return
}
