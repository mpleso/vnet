package ip4

import (
	"github.com/platinasystems/vnet"
	"github.com/platinasystems/vnet/ip"
)

type Prefix struct {
	Address
	Len uint32
}

// True if given destination matches prefix.
func (dst *Address) MatchesPrefix(p *Prefix) bool {
	return 0 == (dst.AsUint32()^p.Address.AsUint32())&mapFibMasks[p.Len]
}

type leaf uint32

const (
	emptyLeaf    leaf = leaf(1 + 2*ip.AdjMiss)
	rootPlyIndex uint = 0
)

func (l leaf) isTerminal() bool    { return l&1 != 0 }
func (l leaf) ResultIndex() ip.Adj { return ip.Adj(l >> 1) }
func setResult(i ip.Adj) leaf      { return leaf(1 + 2*i) }
func (l *leaf) setResult(i ip.Adj) { *l = setResult(i) }
func (l leaf) isPly() bool         { return !l.isTerminal() }
func (l leaf) plyIndex() uint      { return uint(l >> 1) }
func setPlyIndex(i uint) leaf      { return leaf(0 + 2*i) }
func (l *leaf) setPlyIndex(i uint) { *l = setPlyIndex(i) }

const plyLeaves = 1 << 8

type ply struct {
	leaves [plyLeaves]leaf

	// Prefix length of leaves.
	lens [plyLeaves]uint8

	// Number of non-empty leaves.
	nNonEmpty int

	poolIndex uint
}

//go:generate gentemplate -d Package=ip4 -id ply -d PoolType=plyPool -d Type=ply -d Data=plys github.com/platinasystems/elib/pool.tmpl

type mtrie struct {
	// Pool of plies.  Index zero is root ply.
	plyPool

	// Special case leaf for default route 0.0.0.0/0.
	// This is to avoid having to paint default leaf in all plys of trie.
	defaultLeaf leaf
}

func (m *mtrie) LookupStep(l leaf, dst byte) (lʹ leaf) {
	pi := uint(0)
	it := l.isTerminal()
	if !it {
		pi = l.plyIndex()
	}
	lʹ = m.plys[pi].leaves[dst]
	if it {
		lʹ = l
	}
	return
}

func (p *ply) init(l leaf, n uint8) {
	p.nNonEmpty = 0
	if l != emptyLeaf {
		p.nNonEmpty = len(p.leaves)
	}
	for i := 0; i < plyLeaves; i += 4 {
		p.lens[i+0] = n
		p.lens[i+1] = n
		p.lens[i+2] = n
		p.lens[i+3] = n
		p.leaves[i+0] = l
		p.leaves[i+1] = l
		p.leaves[i+2] = l
		p.leaves[i+3] = l
	}
}

func (m *mtrie) newPly(l leaf, n uint8) (lʹ leaf, ply *ply) {
	pi := m.plyPool.GetIndex()
	ply = &m.plys[pi]
	ply.poolIndex = pi
	ply.init(l, n)
	lʹ = setPlyIndex(pi)
	return
}

func (m *mtrie) plyForLeaf(l leaf) *ply { return &m.plys[l.plyIndex()] }

func (m *mtrie) freePly(p *ply) {
	isRoot := p.poolIndex == 0
	for _, l := range p.leaves {
		if !l.isTerminal() {
			m.freePly(m.plyForLeaf(l))
		}
	}
	if isRoot {
		p.init(emptyLeaf, 0)
	} else {
		m.plyPool.PutIndex(p.poolIndex)
	}
}

func (m *mtrie) Free() { m.freePly(&m.plys[0]) }

func (m *mtrie) lookup(dst *Address) ip.Adj {
	p := &m.plys[0]
	for i := range dst {
		l := p.leaves[dst[i]]
		if l.isTerminal() {
			return l.ResultIndex()
		}
		p = m.plyForLeaf(l)
	}
	panic("no terminal leaf found")
}

func (m *mtrie) setPlyWithMoreSpecificLeaf(p *ply, l leaf, n uint8) {
	for i, pl := range p.leaves {
		if !pl.isTerminal() {
			m.setPlyWithMoreSpecificLeaf(m.plyForLeaf(pl), l, n)
		} else if n >= p.lens[i] {
			p.leaves[i] = l
			p.lens[i] = n
			if pl != emptyLeaf {
				p.nNonEmpty++
			}
		}
	}
}

func (p *ply) replaceLeaf(new, old leaf, i uint8) {
	p.leaves[i] = new
	if old != emptyLeaf {
		p.nNonEmpty++
	}
}

type setUnsetLeaf struct {
	key    Address
	keyLen uint8
	result ip.Adj
}

func (s *setUnsetLeaf) setLeafHelper(m *mtrie, oldPlyIndex, keyByteIndex uint) {
	nBits := int(s.keyLen) - 8*int(keyByteIndex+1)
	k := s.key[keyByteIndex]
	oldPly := &m.plys[oldPlyIndex]

	// Number of bits next plies <= 0 => insert leaves this ply.
	if nBits <= 0 {
		nBits = -nBits
		for i := k; i < k+1<<uint(nBits); i++ {
			oldLeaf := oldPly.leaves[i]
			oldTerm := oldLeaf.isTerminal()

			// Is leaf to be inserted more specific?
			if s.keyLen >= oldPly.lens[i] {
				newLeaf := setResult(s.result)
				if oldTerm {
					oldPly.lens[i] = s.keyLen
					oldPly.replaceLeaf(newLeaf, oldLeaf, i)
				} else {
					// Existing leaf points to another ply.
					// We need to place new_leaf into all more specific slots.
					newPly := m.plyForLeaf(oldLeaf)
					m.setPlyWithMoreSpecificLeaf(newPly, newLeaf, s.keyLen)
				}
			} else if !oldTerm {
				s.setLeafHelper(m, oldLeaf.plyIndex(), keyByteIndex+1)
			}
		}
	} else {
		oldLeaf := oldPly.leaves[k]
		oldTerm := oldLeaf.isTerminal()
		var newPly *ply
		if !oldTerm {
			newPly = m.plyForLeaf(oldLeaf)
		} else {
			var newLeaf leaf
			newLeaf, newPly = m.newPly(oldLeaf, oldPly.lens[k])
			// Refetch since newPly may move pool.
			oldPly = &m.plys[oldPlyIndex]
			oldPly.leaves[k] = newLeaf
			oldPly.lens[k] = 0
			if oldLeaf != emptyLeaf {
				oldPly.nNonEmpty--
			}
			// Account for the ply we just created.
			oldPly.nNonEmpty++
		}
		s.setLeafHelper(m, newPly.poolIndex, keyByteIndex+1)
	}
}

func (s *setUnsetLeaf) unsetLeafHelper(m *mtrie, oldPlyIndex, keyByteIndex uint) (oldPlyWasDeleted bool) {
	k := s.key[keyByteIndex]
	nBits := int(s.keyLen) - 8*int(keyByteIndex+1)
	if nBits <= 0 {
		nBits = -nBits
		k &^= 1<<uint(nBits) - 1
		if nBits > 8 {
			nBits = 8
		}
	}
	delLeaf := setResult(s.result)
	oldPly := &m.plys[oldPlyIndex]
	for i := k; i < k+1<<uint(nBits); i++ {
		oldLeaf := oldPly.leaves[i]
		oldTerm := oldLeaf.isTerminal()
		if oldLeaf == delLeaf ||
			(!oldTerm && s.unsetLeafHelper(m, oldLeaf.plyIndex(), keyByteIndex+1)) {
			oldPly.leaves[i] = emptyLeaf
			oldPly.lens[i] = 0
			oldPly.nNonEmpty--
			oldPlyWasDeleted = oldPly.nNonEmpty == 0 && keyByteIndex > 0
			if oldPlyWasDeleted {
				m.plyPool.PutIndex(oldPly.poolIndex)
				// Nothing more to do.
				break
			}
		}
	}

	return
}

func (s *setUnsetLeaf) set(m *mtrie)        { s.setLeafHelper(m, rootPlyIndex, 0) }
func (s *setUnsetLeaf) unset(m *mtrie) bool { return s.unsetLeafHelper(m, rootPlyIndex, 0) }

func (m *mtrie) init() {
	m.defaultLeaf = emptyLeaf
	// Make root ply.
	l, _ := m.newPly(emptyLeaf, 0)
	if l.plyIndex() != 0 {
		panic("root ply must be index 0")
	}
}

type mapFib struct {
	// Maps for /0 through /32; key in network byte order.
	maps [1 + 32]map[uint32]ip.Adj
}

var mapFibMasks [33]uint32

func init() {
	for i := range mapFibMasks {
		m := ^vnet.Uint32(0)
		if i < 32 {
			m = vnet.Uint32(1<<uint(i)-1) << uint(32-i)
		}
		mapFibMasks[i] = uint32(vnet.Uint32(m).FromHost())
	}
}

func (a *Address) mapFibKey(l uint) uint32 { return a.AsUint32() & mapFibMasks[l] }

func (m *mapFib) set(a *Address, l uint, r ip.Adj) {
	if m.maps[l] == nil {
		m.maps[l] = make(map[uint32]ip.Adj)
	}
	k := a.mapFibKey(l)
	m.maps[l][k] = r
}

func (m *mapFib) unset(a *Address, l uint) (ok bool) {
	k := a.mapFibKey(l)
	if _, ok = m.maps[l][k]; ok {
		delete(m.maps[l], k)
	}
	return
}

func (m *mapFib) get(a *Address, l uint) (r ip.Adj, ok bool) {
	k := a.mapFibKey(l)
	r, ok = m.maps[l][k]
	return
}

func (m *mapFib) lookup(a *Address) ip.Adj {
	for l := 32; l >= 0; l-- {
		if m.maps[l] == nil {
			continue
		}
		k := a.mapFibKey(uint(l))
		if r, ok := m.maps[l][k]; ok {
			return r
		}
	}
	return ip.AdjMiss
}

type FibSetUnsetHook func(a *Address, l uint, r ip.Adj, isSet bool)

//go:generate gentemplate -id FibSetUnsetHook -d Package=ip4 -d DepsType=FibSetUnsetHookVec -d Type=FibSetUnsetHook -d Data=hooks github.com/platinasystems/elib/dep/dep.tmpl

type Fib struct {
	// Mtrie for fast lookups.
	mtrie
	// Map fib for general accounting and to maintain mtrie (e.g. setLessSpecific).
	mapFib
	// Hooks to call on set/unset.
	Hooks FibSetUnsetHookVec
}

func (f *Fib) callHooks(a *Address, l uint, r ip.Adj, isSet bool) {
	for i := range f.Hooks.hooks {
		f.Hooks.Get(i)(a, l, r, isSet)
	}
}

func (f *Fib) setUnset(a *Address, l uint, r ip.Adj, isSet bool) {
	// Call hooks before unset.
	if !isSet {
		f.callHooks(a, l, r, isSet)
	}

	// Add/delete in map fib.
	if isSet {
		f.mapFib.set(a, l, r)
	} else {
		f.mapFib.unset(a, l)
	}

	// Add/delete in mtrie fib.
	m := &f.mtrie

	if len(m.plys) == 0 {
		m.init()
	}

	s := setUnsetLeaf{
		key:    *a,
		keyLen: uint8(l),
		result: r,
	}
	if isSet {
		if l == 0 {
			m.defaultLeaf = setResult(s.result)
		} else {
			s.set(m)
		}
	} else {
		if l == 0 {
			m.defaultLeaf = emptyLeaf
		} else {
			s.unset(m)
			f.setLessSpecific(a)
		}
	}

	// Call hooks after set.
	if isSet {
		f.callHooks(a, l, r, isSet)
	}
}

// Find first less specific route matching address and insert into mtrie.
func (f *Fib) setLessSpecific(a *Address) {
	// No need to consider length 0 since that's not in mtrie.
	for l := uint(32); l >= 1; l-- {
		if f.maps[l] == nil {
			continue
		}
		k := a.mapFibKey(l)
		if r, ok := f.maps[l][k]; ok {
			s := setUnsetLeaf{
				result: r,
				keyLen: uint8(l),
			}
			s.key.FromUint32(k)
			s.set(&f.mtrie)
			break
		}
	}
}

func (f *Fib) Set(a *Address, l uint, r ip.Adj) { f.setUnset(a, l, r, true) }
func (f *Fib) Unset(a *Address, l uint)         { f.setUnset(a, l, ip.AdjMiss, true) }
func (f *Fib) Lookup(a *Address) (r ip.Adj) {
	r = f.mtrie.lookup(a)
	return
}
