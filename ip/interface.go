package ip

import (
	"github.com/platinasystems/vnet"

	"strconv"
)

// Generic ip4/ip6 address: big enough for either.
type Address [16]uint8

type Prefix struct {
	Address
	Len uint32
}

type Addresser interface {
	String(a *Address) string
}

func (p *Prefix) String(m *ifAddressMain) string {
	return m.String(&p.Address) + "/" + strconv.Itoa(int(p.Len))
}

type ifAddr uint32

const ifAddrNone = ^ifAddr(0)

//go:generate gentemplate -d Package=ip -id ifAddr -d VecType=ifAddrVec -d Type=ifAddr github.com/platinasystems/elib/vec.tmpl

type IfAddress struct {
	// ip4/ip6 address and map key plus length.
	Prefix Prefix

	// Interface which has this address.
	Si vnet.Si

	// Next and previous pointers in doubly-linked list of interface addresses for this interface.
	next, prev ifAddr
}

//go:generate gentemplate -d Package=ip -id ifaddress -d PoolType=ifAddressPool -d Type=IfAddress -d Data=ifAddrs github.com/platinasystems/elib/pool.tmpl

type ifAddressMain struct {
	*vnet.Vnet

	Addresser

	ifAddressPool

	// Maps ip4/ip6 address to pool index.
	addrMap map[Address]ifAddr

	// Head of doubly-linked list indexed by software interface.
	headBySwIf []ifAddr
}

func (m *ifAddressMain) GetIfAddress(a []uint8) (ia *IfAddress) {
	var k Address
	copy(k[:], a)
	if i, ok := m.addrMap[k]; ok {
		ia = &m.ifAddrs[i]
	}
	return
}
func (i ifAddr) Get(m *ifAddressMain) *IfAddress                       { return &m.ifAddrs[i] }
func (m *ifAddressMain) IfFirstAddress(i vnet.Si) *IfAddress           { return m.headBySwIf[i].Get(m) }
func (m *ifAddressMain) IfAddressForAdjacency(a *Adjacency) *IfAddress { return a.ifAddr.Get(m) }

func (m *ifAddressMain) ForeachIfAddress(si vnet.Si, f func(i *IfAddress)) {
	i := m.headBySwIf[si]
	for i != ifAddrNone {
		ia := i.Get(m)
		f(ia)
		i = ia.next
	}
}

func (m *ifAddressMain) ifAddressAddDel(si vnet.Si, p *Prefix, isDel bool) {
	var (
		a  *IfAddress
		ai ifAddr
	)
	if ai, ok := m.addrMap[p.Address]; ok {
		a = ai.Get(m)
	}

	if isDel {
		if a == nil {
			m.Fatalf("%s: address %s not found", si.IfName(m.Vnet), p.String(m))
		}
		if a.prev != ifAddrNone {
			prev := a.prev.Get(m)
			prev.next = a.next
		} else {
			// Delete list head.
			m.headBySwIf[si] = ifAddrNone
		}
		if a.next != ifAddrNone {
			next := a.next.Get(m)
			next.prev = a.prev
		}

		delete(m.addrMap, p.Address)
		m.ifAddressPool.PutIndex(uint(ai))
	} else if a == nil {
		ai = ifAddr(m.ifAddressPool.GetIndex())
		a = ai.Get(m)

		if m.addrMap == nil {
			m.addrMap = make(map[Address]ifAddr)
		}
		m.addrMap[p.Address] = ai
		a.Prefix = *p
		a.Si = si

		pi := m.headBySwIf[si]
		a.next = ifAddrNone
		a.prev = pi

		// Make previous head point to added element and set added element as new head.
		if pi != ifAddrNone {
			p := pi.Get(m)
			a.next = pi
			p.prev = ai
		}
		m.headBySwIf[si] = ai
	}
}
