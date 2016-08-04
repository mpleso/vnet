// +build linux

package tuntap

import (
	"github.com/platinasystems/elib"
	"github.com/platinasystems/elib/parse"
	"github.com/platinasystems/vnet"
	"github.com/platinasystems/vnet/ethernet"

	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

type iovec syscall.Iovec

//go:generate gentemplate -d Package=tuntap -id iovec -d VecType=iovecVec -d Type=iovec github.com/platinasystems/elib/vec.tmpl

func rwv(fd int, iov []syscall.Iovec, isWrite bool) (n int, err error) {
	nm, sc := "readv", syscall.SYS_READV
	if isWrite {
		nm, sc = "writev", syscall.SYS_WRITEV
	}
	r0, _, e := syscall.Syscall(uintptr(sc), uintptr(fd), uintptr(unsafe.Pointer(&iov[0])), uintptr(len(iov)))
	if e != 0 {
		err = fmt.Errorf("%s: %s", nm, e)
	}
	n = int(r0)
	return
}

func readv(fd int, iov []syscall.Iovec) (int, error)  { return rwv(fd, iov, false) }
func writev(fd int, iov []syscall.Iovec) (int, error) { return rwv(fd, iov, true) }

type Interface struct {
	hi        vnet.Hi
	si        vnet.Si
	name      ifreq_name
	poolIndex uint // index in ifPool
	socket    int  // provisioning socket
	ifindex   int  // linux interface index
	flags     iff_flag
}

//go:generate gentemplate -d Package=tuntap -id ifPool -d PoolType=ifPool -d Type=Interface -d Data=elts github.com/platinasystems/elib/pool.tmpl

func (m *Main) interfaceForSi(si vnet.Si) (i *Interface) {
	i = &m.ifPool.elts[m.ifPoolIndexBySi[si]]
	return
}

type Main struct {
	vnet.Package

	v *vnet.Vnet

	// Selects whether we create tun or tap interfaces.
	isTun bool

	disable parse.Enable

	// /dev/net/tun
	dev_net_tun_fd int

	mtu_bytes, mtu_buffers uint32

	ifPool ifPool

	ifPoolIndexByName map[ifreq_name]uint
	ifPoolIndexBySi   elib.Uint32Vec
}

func Init(v *vnet.Vnet) {
	m := &Main{}
	m.v = v
	v.AddPackage("tuntap", m)
}

const (
	// TUNSETIFF ifReq flags
	iff_tun   = 1 << 0
	iff_tap   = 1 << 1
	iff_no_pi = 1 << 10

	iff_persist  = 1 << 9
	iff_nofilter = 1 << 10
)

type ifreq_name [16]byte

func (n ifreq_name) String() string { return strings.TrimRight(string(n[:]), "\x00") }

// Linux interface flags
type iff_flag int

const (
	iff_up_bit, iff_up iff_flag = iota, 1 << iota
	iff_broadcast_bit, iff_broadcast
	iff_debug_bit, iff_debug
	iff_loopback_bit, iff_loopback
	iff_pointopoint_bit, iff_pointopoint
	iff_notrailers_bit, iff_notrailers
	iff_running_bit, iff_running
	iff_noarp_bit, iff_noarp
	iff_promisc_bit, iff_promisc
	iff_allmulti_bit, iff_allmulti
	iff_master_bit, iff_master
	iff_slave_bit, iff_slave
	iff_multicast_bit, iff_multicast
	iff_portsel_bit, iff_portsel
	iff_automedia_bit, iff_automedia
	iff_dynamic_bit, iff_dynamic
	iff_lower_up_bit, iff_lower_up
	iff_dormant_bit, iff_dormant
	iff_echo_bit, iff_echo
)

var iff_flag_names = [...]string{
	iff_up_bit:          "up",
	iff_broadcast_bit:   "broadcast",
	iff_debug_bit:       "debug",
	iff_loopback_bit:    "loopback",
	iff_pointopoint_bit: "point-to-point",
	iff_notrailers_bit:  "no-trailers",
	iff_running_bit:     "running",
	iff_noarp_bit:       "no-arp",
	iff_promisc_bit:     "promiscuous",
	iff_allmulti_bit:    "all-multicast",
	iff_master_bit:      "master",
	iff_slave_bit:       "slave",
	iff_multicast_bit:   "multicast",
	iff_portsel_bit:     "portsel",
	iff_automedia_bit:   "automedia",
	iff_dynamic_bit:     "dynamic",
	iff_lower_up_bit:    "lower-up",
	iff_dormant_bit:     "dormant",
	iff_echo_bit:        "echo",
}

func (f iff_flag) String() string { return elib.FlagStringer(iff_flag_names[:], elib.Word(f)) }

type ifreq_flags struct {
	name  ifreq_name
	flags uint16
}

type ifreq_int struct {
	name ifreq_name
	i    int
}

type ifreq_sockaddr_any struct {
	name     ifreq_name
	sockaddr syscall.RawSockaddrAny
}

type ifreq_type int

const (
	ifreq_TUNSETIFF     ifreq_type = syscall.TUNSETIFF
	ifreq_TUNSETPERSIST ifreq_type = syscall.TUNSETPERSIST
	ifreq_GETIFINDEX    ifreq_type = syscall.SIOCGIFINDEX
	ifreq_GETIFFLAGS    ifreq_type = syscall.SIOCGIFFLAGS
	ifreq_SETIFFLAGS    ifreq_type = syscall.SIOCSIFFLAGS
	ifreq_SETIFHWADDR   ifreq_type = syscall.SIOCSIFHWADDR
)

var ifreq_type_names = map[ifreq_type]string{
	ifreq_TUNSETIFF:     "TUNSETIFF",
	ifreq_TUNSETPERSIST: "TUNSETPERSIST",
	ifreq_GETIFINDEX:    "GETIFINDEX",
	ifreq_GETIFFLAGS:    "GETIFFLAGS",
	ifreq_SETIFFLAGS:    "SETIFFLAGS",
	ifreq_SETIFHWADDR:   "SETIFHWADDR",
}

func (t ifreq_type) String() string {
	if s, ok := ifreq_type_names[t]; ok {
		return s
	}
	return fmt.Sprintf("0x%x", t)
}

// Create tuntap interfaces for all vnet interfaces not marked as special.
func (m *Main) okHi(hi vnet.Hi) (ok bool) { return m.v.HwIfer(hi).IsUnix() }
func (m *Main) okSi(si vnet.Si) bool      { return m.okHi(m.v.SupHi(si)) }

func (m *Main) ioctl(req ifreq_type, arg uintptr) (err error) {
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(m.dev_net_tun_fd), uintptr(req), arg)
	if e != 0 {
		err = fmt.Errorf("tuntap ioctl %s: %s", req, e)
	}
	return
}

func (i *Interface) ioctl(req ifreq_type, arg uintptr) (err error) {
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(i.socket), uintptr(req), arg)
	if e != 0 {
		err = fmt.Errorf("tuntap %s ioctl %s: %s", i.name, req, e)
	}
	return
}

func (m *Main) SwIfAddDel(v *vnet.Vnet, si vnet.Si, isDel bool) (err error) {
	hi := m.v.SupHi(si)
	if !m.okHi(hi) {
		return
	}

	// Tuntap interfaces are never deleted; only created.
	if isDel {
		return
	}

	intf := Interface{
		hi: hi,
		si: si,
	}

	copy(intf.name[:], si.Name(v))

	// Create interface (set flags) and make persistent (e.g. interface stays around when we die).
	{
		r := ifreq_flags{name: intf.name}
		r.flags = iff_no_pi
		if m.isTun {
			r.flags |= iff_tun
		} else {
			r.flags |= iff_tap
		}
		if err = m.ioctl(ifreq_TUNSETIFF, uintptr(unsafe.Pointer(&r))); err != nil {
			return
		}
		if err = m.ioctl(ifreq_TUNSETPERSIST, 1); err != nil {
			return
		}
	}

	// Create provisioning socket.
	eth_p_all := uint16(vnet.Uint16(syscall.ETH_P_ALL).FromHost())
	if intf.socket, err = syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(eth_p_all)); err != nil {
		err = fmt.Errorf("tuntap socket: %s", err)
		return
	}
	defer func() {
		if err != nil {
			syscall.Close(intf.socket)
		}
	}()

	if err = syscall.SetNonblock(intf.socket, true); err != nil {
		err = fmt.Errorf("tuntap set non-blocking: %s", err)
		return
	}

	// Find linux interface index.
	{
		r := ifreq_int{name: intf.name}
		if err = intf.ioctl(ifreq_GETIFINDEX, uintptr(unsafe.Pointer(&r))); err != nil {
			return
		}
		intf.ifindex = r.i
	}

	// Bind the provisioning socket to the interface.
	{
		sa := syscall.SockaddrLinklayer{
			Ifindex:  intf.ifindex,
			Protocol: eth_p_all,
		}
		if err = syscall.Bind(intf.socket, &sa); err != nil {
			err = fmt.Errorf("tuntap bind: %s", err)
			return
		}
	}

	// Fetch initial interface flags.
	{
		r := ifreq_int{name: intf.name}
		if err = intf.ioctl(ifreq_GETIFFLAGS, uintptr(unsafe.Pointer(&r))); err != nil {
			return
		}
		intf.flags = iff_flag(r.i)
	}

	if eifer, ok := m.v.HwIfer(hi).(ethernet.HwInterfacer); ok {
		ei := eifer.GetInterface()

		// For tap interfaces, set ethernet address of interface.
		if !m.isTun {
			r := ifreq_sockaddr_any{name: intf.name}
			r.sockaddr.Addr.Family = syscall.ARPHRD_ETHER
			for i := range ei.Address {
				r.sockaddr.Addr.Data[i] = int8(ei.Address[i])
			}
			if err = intf.ioctl(ifreq_SETIFHWADDR, uintptr(unsafe.Pointer(&r))); err != nil {
				return
			}
		}
	}

	intf.poolIndex = m.ifPool.GetIndex()
	m.ifPool.elts[intf.poolIndex] = intf

	m.ifPoolIndexBySi.ValidateInit(uint(si), ^uint32(0))
	m.ifPoolIndexBySi[si] = uint32(intf.poolIndex)
	if m.ifPoolIndexByName == nil {
		m.ifPoolIndexByName = make(map[ifreq_name]uint)
	}
	m.ifPoolIndexByName[intf.name] = intf.poolIndex
	return
}

func (m *Main) maybeChangeFlag(si vnet.Si, isUp bool, flag iff_flag) (err error) {
	intf := m.interfaceForSi(si)
	change := false
	switch {
	case isUp && intf.flags&flag == 0:
		change = true
		intf.flags |= flag
	case !isUp && intf.flags&flag != 0:
		change = true
		intf.flags &^= flag
	}
	if change {
		r := ifreq_int{
			name: intf.name,
			i:    int(intf.flags),
		}
		err = intf.ioctl(ifreq_SETIFFLAGS, uintptr(unsafe.Pointer(&r)))
	}
	return
}

func (m *Main) SwIfAdminUpDown(v *vnet.Vnet, si vnet.Si, isUp bool) (err error) {
	if !m.okSi(si) {
		return
	}
	err = m.maybeChangeFlag(si, isUp, iff_up)
	return
}

func (m *Main) HwIfLinkUpDown(v *vnet.Vnet, hi vnet.Hi, isUp bool) (err error) {
	if !m.okHi(hi) {
		return
	}
	err = m.maybeChangeFlag(v.HwIf(hi).Si(), isUp, iff_running)
	return
}

func (m *Main) Init() (err error) {
	var fd int
	if fd, err = syscall.Open("/dev/net/tun", syscall.O_RDWR, 0); err != nil {
		return
	}
	defer func() {
		if err != nil {
			syscall.Close(fd)
		}
	}()

	if err = syscall.SetNonblock(fd, true); err != nil {
		return
	}

	m.v.RegisterSwIfAddDelHook(m.SwIfAddDel)
	m.v.RegisterSwIfAdminUpDownHook(m.SwIfAdminUpDown)
	m.v.RegisterHwIfLinkUpDownHook(m.HwIfLinkUpDown)
	m.dev_net_tun_fd = fd
	return
}

// Shutdown interfaces on main loop exit.
func (m *Main) Exit() (err error) {
	m.ifPool.Foreach(func(intf Interface) {
		intf.flags &^= iff_up | iff_running
		r := ifreq_int{
			name: intf.name,
			i:    int(intf.flags),
		}
		intf.ioctl(ifreq_SETIFFLAGS, uintptr(unsafe.Pointer(&r)))
		syscall.Close(intf.socket)
	})
	syscall.Close(m.dev_net_tun_fd)
	return
}

func (m *Main) Configure(in *parse.Input) {
	// Suitable defaults for an Ethernet-like tun/tap device.
	if m.mtu_bytes == 0 {
		m.mtu_bytes = 4096 + 256
	}

	for !in.End() {
		switch {
		case in.Parse("mtu %d", &m.mtu_bytes):
		case in.Parse("tap"):
			m.isTun = false
		case in.Parse("tun"):
			m.isTun = true
		case in.Parse("en*able"):
			m.disable = false
		case in.Parse("dis*able"):
			m.disable = true
		default:
			panic(parse.ErrInput)
		}
	}
}
