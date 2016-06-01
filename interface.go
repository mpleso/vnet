package vnet

import (
	"github.com/platinasystems/elib/loop"
	"github.com/platinasystems/elib/scan"

	"errors"
	"fmt"
)

type HwIf struct {
	vnet *Vnet

	name string

	hi Hi
	si Si

	devicer Devicer

	// Hardware link state: up or down
	linkUp bool

	// Hardware is unprovisioned.
	// Interfaces with 4 SERDES lanes will be represented as 4 interfaces.
	// Lanes may all be a single interface (1 provisioned 4 lane interface +
	// 3 unprovisioned 0 lane interfaces).
	unprovisioned bool

	speed Bandwidth

	// Max size of packet in bytes (MTU)
	maxPacketSize uint

	subSiById map[IfIndex]Si
}

type IfIndex uint32

type HwInterfacer interface {
	HwIfClasser
	GetHwIf() *HwIf
}

func (h *HwIf) GetHwIf() *HwIf { return h }
func (h *HwIf) Name() string   { return h.name }
func (h *HwIf) Si() Si         { return h.si }
func (h *HwIf) Hi() Hi         { return h.hi }

func (h *HwIf) SetName(v *Vnet, name string) {
	h.name = name
	v.hwIfIndexByName.Set(name, uint(h.hi))
}

func (h *HwIf) SetSubInterface(id IfIndex, si Si) {
	if h.subSiById == nil {
		h.subSiById = make(map[IfIndex]Si)
	}
	h.subSiById[id] = si
}

func (h *HwIf) LinkString() (s string) {
	s = "down"
	if h.linkUp {
		s = "up"
	}
	return
}

// Software and hardware interface index.
// Alias for commonly used types.
type Si IfIndex
type Hi IfIndex

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
	si Si

	// Software interface index of super-interface.
	// Equal to index if this interface is not a sub-interface.
	supSi Si

	// For hardware interface: HwIfIndex
	// For sub interface: sub interface id (e.g. vlan/vc number).
	id IfIndex
}

//go:generate gentemplate -d Package=vnet -id swIf -d PoolType=swIfPool -d Type=swIf -d Data=elts github.com/platinasystems/elib/pool.tmpl

func (m *Vnet) NewSwIf(kind swIfKind, id IfIndex) (si Si) {
	si = Si(m.swInterfaces.GetIndex())
	s := m.SwIf(si)
	s.kind = kind
	s.si = si
	s.supSi = si
	s.id = id
	m.counterValidate(si)

	isDel := false
	for i := range m.swIfAddDelHooks.hooks {
		err := m.swIfAddDelHooks.Get(i)(m, s.si, isDel)
		if err != nil {
			panic(err) // how to recover?
		}
	}
	return
}

func (m *interfaceMain) SwIf(i Si) *swIf { return &m.swInterfaces.elts[i] }
func (m *interfaceMain) SupSwIf(s *swIf) (sup *swIf) {
	sup = s
	if s.supSi != s.si {
		sup = m.SwIf(s.supSi)
	}
	return
}
func (m *interfaceMain) HwIfer(i Hi) HwInterfacer { return m.hwInterfaces[i] }
func (m *interfaceMain) HwIf(i Hi) *HwIf          { return m.hwInterfaces[i].GetHwIf() }
func (m *interfaceMain) SupHwIf(s *swIf) *HwIf {
	sup := m.SupSwIf(s)
	return m.HwIf(Hi(sup.id))
}

func (s *swIf) IfName(vn *Vnet) (v string) {
	v = vn.SupHwIf(s).name
	if s.kind != swIfHardware {
		v += fmt.Sprintf(".%d", s.id)
	}
	return
}
func (s Si) IfName(v *Vnet) string { return v.SwIf(s).IfName(v) }

func (i *swIf) Id() IfIndex { return i.id }

func (i *swIf) IsAdminUp() bool { return i.flags&swIfAdminUp != 0 }

func (sw *swIf) SetAdminUp(v *Vnet, wantUp bool) (err error) {
	isUp := sw.flags&swIfAdminUp != 0
	if isUp == wantUp {
		return
	}
	sw.flags ^= swIfAdminUp
	for i := range v.swIfAdminUpDownHooks.hooks {
		err = v.swIfAdminUpDownHooks.Get(i)(v, sw.si, wantUp)
		if err != nil {
			return
		}
	}
	return
}

func (h *HwIf) SetAdminUp(isUp bool) (err error) {
	if h.unprovisioned {
		err = errors.New("hardware interface is unprovisioned")
		return
	}

	s := h.vnet.SwIf(h.si)
	err = s.SetAdminUp(h.vnet, isUp)
	return
}

func (h *HwIf) IsProvisioned() bool { return !h.unprovisioned }

func (h *HwIf) SetProvisioned(v bool) (err error) {
	if !h.unprovisioned == v {
		return
	}
	vn := h.vnet
	for i := range vn.hwIfProvisionHooks.hooks {
		err = vn.hwIfProvisionHooks.Get(i)(vn, h.hi, v)
		if err != nil {
			break
		}
	}
	// Toggle provisioning hooks show no error.
	if err != nil {
		h.unprovisioned = !v
	}
	return
}

func (h *HwIf) LinkUp() bool { return h.linkUp }

func (h *HwIf) SetLinkUp(v bool) (err error) {
	if h.linkUp == v {
		return
	}
	h.linkUp = v
	vn := h.vnet
	for i := range vn.hwIfLinkUpDownHooks.hooks {
		err = vn.hwIfLinkUpDownHooks.Get(i)(vn, h.hi, v)
		if err != nil {
			return
		}
	}
	return
}

func (h *HwIf) MaxPacketSize() uint { return h.maxPacketSize }

func (h *HwIf) SetMaxPacketSize(v uint) (err error) {
	h.maxPacketSize = v
	// fixme call hooks
	return
}

func (h *HwIf) Speed() Bandwidth { return h.speed }

func (h *HwIf) SetSpeed(v Bandwidth) (err error) {
	vn := h.vnet
	for i := range vn.hwIfSetSpeedHooks.hooks {
		err := vn.hwIfSetSpeedHooks.Get(i)(vn, h.hi, v)
		if err != nil {
			break
		}
	}
	if err != nil {
		h.speed = v
	}
	return
}

type interfaceMain struct {
	hwInterfaces             []HwInterfacer
	hwIfIndexByName          scan.StringMap
	swInterfaces             swIfPool
	ifThreads                ifThreadVec
	swIfCounterNames         []string
	swIfCombinedCounterNames []string
	swIfAddDelHooks          SwIfAddDelHookVec
	swIfAdminUpDownHooks     SwIfAdminUpDownHookVec
	hwIfAddDelHooks          HwIfAddDelHookVec
	hwIfLinkUpDownHooks      HwIfLinkUpDownHookVec
	hwIfProvisionHooks       HwIfProvisionHookVec
	hwIfSetSpeedHooks        HwIfSetSpeedHookVec
}

//go:generate gentemplate -d Package=vnet -id ifThread -d VecType=ifThreadVec -d Type=*interfaceThread github.com/platinasystems/elib/vec.tmpl

func (v *Vnet) RegisterHwInterface(hi HwInterfacer, format string, args ...interface{}) (err error) {
	l := len(v.hwInterfaces)
	v.hwInterfaces = append(v.hwInterfaces, hi)
	h := hi.GetHwIf()
	h.vnet = v
	h.hi = Hi(l)
	h.si = v.NewSwIf(swIfHardware, IfIndex(h.hi))
	h.SetName(v, fmt.Sprintf(format, args...))
	return
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
func (n *Node) GetIfThread() *interfaceThread { return n.Vnet.GetIfThread(n.ThreadId()) }

// Interface ordering for output.
func (m *interfaceMain) HwLessThan(a, b *HwIf) bool { return a.name < b.name }

func (m *interfaceMain) SwLessThan(a, b *swIf) bool {
	ha, hb := m.SupHwIf(a), m.SupHwIf(b)
	if ha != hb {
		return m.HwLessThan(ha, hb)
	}
	return a.id < b.id
}

// Interface can loopback at MAC or PHY.
type IfLoopbackType int

const (
	None IfLoopbackType = iota
	Mac
	Phy
)

// To clarify units: 1e9 * vnet.Bps
const (
	Bps    = 1e0
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

func (b *Bandwidth) Parse(s *scan.Scanner) (err error) {
	var f scan.Float64
	if err = f.Parse(s); err == nil {
		unit := Bps
		switch {
		case s.AdvanceIfMulti('K', 'k'):
			unit = Kbps
		case s.AdvanceIfMulti('M', 'm'):
			unit = Mbps
		case s.AdvanceIfMulti('G', 'g'):
			unit = Gbps
		case s.AdvanceIfMulti('T', 't'):
			unit = Tbps
		case s.Peek() != scan.EOF:
			tok, text := s.Scan()
			err = s.UnexpectedError(scan.EOF, tok, text)
			return
		}
		*b = Bandwidth(float64(f) * unit)
		return
	}
	return
}

// Class of hardware interfaces, for example, ethernet, sonet, srp, docsis, etc.
type HwIfClasser interface {
	SetRewrite(v *Vnet, r *Rewrite, t PacketType, dstAddr []byte)
}

type Devicer interface {
	Noder
	loop.InputLooper
	loop.OutputLooper
}

type SwIfAddDelHook func(v *Vnet, si Si, isDel bool) error
type SwIfAdminUpDownHook func(v *Vnet, si Si, isUp bool) error
type HwIfAddDelHook func(v *Vnet, hi Hi, isDel bool) error
type HwIfLinkUpDownHook func(v *Vnet, hi Hi, isUp bool) error
type HwIfProvisionHook func(v *Vnet, hi Hi, isProvisioned bool) error
type HwIfSetSpeedHook func(v *Vnet, hi Hi, speed Bandwidth) error

//go:generate gentemplate -id SwIfAddDelHook -d Package=vnet -d DepsType=SwIfAddDelHookVec -d Type=SwIfAddDelHook -d Data=hooks github.com/platinasystems/elib/dep/dep.tmpl
//go:generate gentemplate -id SwIfAdminUpDownHook -d Package=vnet -d DepsType=SwIfAdminUpDownHookVec -d Type=SwIfAdminUpDownHook -d Data=hooks github.com/platinasystems/elib/dep/dep.tmpl
//go:generate gentemplate -id HwIfAddDelHook -d Package=vnet -d DepsType=HwIfAddDelHookVec -d Type=HwIfAddDelHook -d Data=hooks github.com/platinasystems/elib/dep/dep.tmpl
//go:generate gentemplate -id HwIfLinkUpDownHook -d Package=vnet -d DepsType=HwIfLinkUpDownHookVec -d Type=HwIfLinkUpDownHook -d Data=hooks github.com/platinasystems/elib/dep/dep.tmpl
//go:generate gentemplate -id HwIfProvisionHook -d Package=vnet -d DepsType=HwIfProvisionHookVec -d Type=HwIfProvisionHook -d Data=hooks github.com/platinasystems/elib/dep/dep.tmpl
//go:generate gentemplate -id HwIfSetSpeedHook -d Package=vnet -d DepsType=HwIfSetSpeedHookVec -d Type=HwIfSetSpeedHook -d Data=hooks github.com/platinasystems/elib/dep/dep.tmpl

func (m *interfaceMain) RegisterSwIfAddDelHook(h SwIfAddDelHook) {
	m.swIfAddDelHooks.Add(h)
}
func (m *interfaceMain) RegisterSwIfAdminUpDownHook(h SwIfAdminUpDownHook) {
	m.swIfAdminUpDownHooks.Add(h)
}
func (m *interfaceMain) RegisterHwIfAddDelHook(h HwIfAddDelHook) {
	m.hwIfAddDelHooks.Add(h)
}
func (m *interfaceMain) RegisterHwIfLinkUpDownHook(h HwIfLinkUpDownHook) {
	m.hwIfLinkUpDownHooks.Add(h)
}
func (m *interfaceMain) RegisterHwIfProvisionHook(h HwIfProvisionHook) {
	m.hwIfProvisionHooks.Add(h)
}
func (m *interfaceMain) RegisterHwIfSetSpeedHook(h HwIfSetSpeedHook) {
	m.hwIfSetSpeedHooks.Add(h)
}
