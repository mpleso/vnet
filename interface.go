package vnet

import (
	"fmt"
)

type HwIf struct {
	ifName string

	hwIf HwIfIndex
	swIf SwIfIndex

	hwInstance  uint32
	devInstance uint32

	// Hardware link state: up or down
	linkUp bool

	// Hardware is unprovisioned.
	// Interfaces with 4 SERDES lanes will be represented as 4 interfaces.
	// Lanes may all be a single interface (1 provisioned 4 lane interface +
	// 3 unprovisioned 0 lane interfaces).
	unprovisioned bool

	speed Bandwidth

	// Max size of packet in bytes (MTU)
	maxPacketSize int
}

type IfIndex uint32
type HwIfIndex IfIndex

type HwInterfacer interface {
	GetHwIf() *HwIf
}

func (h *HwIf) GetHwIf() *HwIf         { return h }
func (h *HwIf) IfName() string         { return h.ifName }
func (h *HwIf) SetIfName(v string)     { h.ifName = v }
func (h *HwIf) Speed() Bandwidth       { return h.speed }
func (h *HwIf) SetSpeed(v Bandwidth)   { h.speed = v }
func (h *HwIf) LinkUp() bool           { return h.linkUp }
func (h *HwIf) SetLinkUp(v bool)       { h.linkUp = v }
func (h *HwIf) IsProvisioned() bool    { return !h.unprovisioned }
func (h *HwIf) SetProvisioned(v bool)  { h.unprovisioned = !v }
func (h *HwIf) MaxPacketSize() int     { return h.maxPacketSize }
func (h *HwIf) SetMaxPacketSize(v int) { h.maxPacketSize = v }
func (h *HwIf) SwIfIndex() SwIfIndex   { return h.swIf }

func (h *HwIf) SetAdminUp(v bool) {
	s := defaultVnet.SwIf(h.swIf)
	s.SetAdminUp(v)
}

func (h *HwIf) LinkString() (s string) {
	s = "down"
	if h.linkUp {
		s = "up"
	}
	return
}

type SwIfIndex IfIndex

// Software and hardware interface index.
// Alias for commonly used types.
type Si SwIfIndex
type Hi HwIfIndex

const (
	SiNil Si = ^Si(0)
	HiNil Hi = ^Hi(0)
)

type swIfKind uint16

const (
	swIfHardware swIfKind = iota + 1
	swIfSub
)

type swIfFlag uint16

const (
	swIfAdminUpIndex, swIfAdminUp swIfFlag = iota, 1 << iota
	swIfPuntIndex, swIfPunt
)

func (f swIfFlag) String() (s string) {
	s = "down"
	if f&swIfAdminUp != 0 {
		s = "up"
	}
	extra := ""
	if f&swIfPunt != 0 {
		if extra != "" {
			extra += ", "
		}
		extra += "punt"
	}
	if extra != "" {
		s += "(" + extra + ")"
	}
	return
}

type swIf struct {
	kind  swIfKind
	flags swIfFlag

	// Pool index for this interface.
	swIf SwIfIndex

	// Software interface index of super-interface.
	// Equal to index if this interface is not a sub-interface.
	supSwIf SwIfIndex

	// For hardware interface: HwIfIndex
	// For sub interface: sub interface id (e.g. vlan/vc number).
	id IfIndex
}

//go:generate gentemplate -d Package=vnet -id swIf -d PoolType=swIfPool -d Type=swIf -d Data=elts github.com/platinasystems/elib/pool.tmpl

func (m *interfaceMain) NewSwIf(kind swIfKind, id IfIndex) (si SwIfIndex) {
	si = SwIfIndex(m.swInterfaces.GetIndex())
	s := m.SwIf(si)
	s.kind = kind
	s.swIf = si
	s.supSwIf = si
	s.id = id
	m.counterValidate(si)
	return
}

func (m *interfaceMain) SwIf(i SwIfIndex) *swIf { return &m.swInterfaces.elts[i] }
func (m *interfaceMain) SupSwIf(s *swIf) (sup *swIf) {
	sup = s
	if s.supSwIf != s.swIf {
		sup = m.SwIf(s.supSwIf)
	}
	return
}
func (m *interfaceMain) HwIf(i HwIfIndex) *HwIf { return m.hwInterfaces[i].GetHwIf() }
func (m *interfaceMain) SupHwIf(s *swIf) *HwIf {
	sup := m.SupSwIf(s)
	return m.HwIf(HwIfIndex(sup.id))
}

func (s *swIf) IfName(vn *Vnet) (v string) {
	v = vn.SupHwIf(s).ifName
	if s.kind != swIfHardware {
		v += fmt.Sprintf(".%d", s.id)
	}
	return
}
func (s SwIfIndex) IfName(v *Vnet) string { return v.SwIf(s).IfName(v) }

func (i *swIf) AdminUp() bool     { return i.flags&swIfAdminUp != 0 }
func (i *swIf) SetAdminUp(v bool) { i.flags |= swIfAdminUp }

type interfaceMain struct {
	hwInterfaces             []HwInterfacer
	swInterfaces             swIfPool
	ifThreads                ifThreadVec
	swIfCounterNames         []string
	swIfCombinedCounterNames []string
}

//go:generate gentemplate -d Package=vnet -id ifThread -d VecType=ifThreadVec -d Type=*interfaceThread github.com/platinasystems/elib/vec.tmpl

func (v *Vnet) RegisterHwInterface(hi HwInterfacer, format string, args ...interface{}) {
	l := len(v.hwInterfaces)
	v.hwInterfaces = append(v.hwInterfaces, hi)
	h := hi.GetHwIf()
	h.hwIf = HwIfIndex(l)
	h.swIf = v.NewSwIf(swIfHardware, IfIndex(h.hwIf))
	h.SetIfName(fmt.Sprintf(format, args...))
}

type interfaceThread struct {
	// This threads' interface counters indexed by counter kind.
	singleCounters   CountersVec
	combinedCounters CombinedCountersVec
}

func (v *Vnet) newInterfaceThread() (t *interfaceThread) {
	t = &interfaceThread{}
	v.counterInit(t)
	return
}

func (v *Vnet) GetIfThread(id uint) (t *interfaceThread) {
	v.ifThreads.Validate(uint(id))
	if t = v.ifThreads[id]; t == nil {
		t = v.newInterfaceThread()
		v.ifThreads[id] = t
	}
	return
}
func (n *Node) GetIfThread() *interfaceThread { return defaultVnet.GetIfThread(n.ThreadId()) }

// Interface ordering for output.
func (m *interfaceMain) HwLessThan(a, b *HwIf) bool {
	if a.ifName == b.ifName {
		return a.hwInstance < b.hwInstance
	}
	return a.ifName < b.ifName
}

func (m *interfaceMain) SwLessThan(a, b *swIf) bool {
	ha, hb := m.SupHwIf(a), m.SupHwIf(b)
	if ha != hb {
		return m.HwLessThan(ha, hb)
	}
	return a.id < b.id
}

// Interface can loopback at MAC or PHY.
type IntfLoopbackType int

const (
	None IntfLoopbackType = iota
	Mac
	Phy
)

// To clarify units: 1e9 * vnet.Bps
const (
	Bps    = 1
	Kbps   = 1e3
	Mbps   = 1e6
	Gbps   = 1e9
	Tbps   = 1e12
	Bytes  = 1
	Kbytes = 1 << 10
	Mbytes = 1 << 20
	Gbytes = 1 << 30
)

type Bandwidth float64

func (b Bandwidth) String() string {
	unit := Bandwidth(1)
	prefix := ""
	switch {
	case b < Kbps:
		break
	case b <= Mbps:
		unit = Kbps
		prefix = "K"
	case b <= Gbps:
		unit = Mbps
		prefix = "M"
	case b <= Tbps:
		unit = Gbps
		prefix = "G"
	default:
		unit = Tbps
		prefix = "T"
	}
	b /= unit
	return fmt.Sprintf("%g%sbps", b, prefix)
}
