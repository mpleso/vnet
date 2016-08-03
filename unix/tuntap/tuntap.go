// +build linux

package tuntap

import (
	"github.com/platinasystems/elib/parse"
	"github.com/platinasystems/vnet"

	"fmt"
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
	hi vnet.Hi
	si vnet.Si
}

type Main struct {
	vnet.Package

	v *vnet.Vnet

	// Selects whether we create tun or tap interfaces.
	isTap bool

	disable parse.Enable

	// /dev/net/tun
	dev_net_tun_fd int

	mtu_bytes, mtu_buffers uint32
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

type ifReq struct {
	name  [16]byte
	flags uint16
	_     [0x28 - 16 - 2]byte
}

func (m *Main) setupInterface(hw *vnet.HwIf) {
}

// Create tuntap interfaces for all vnet interfaces not marked as special.
func (m *Main) okHi(hi vnet.Hi) (ok bool) { return m.v.HwIfer(hi).IsUnix() }
func (m *Main) okSi(si vnet.Si) bool      { return m.okHi(m.v.SupHi(si)) }

func (m *Main) SwIfAddDel(v *vnet.Vnet, si vnet.Si, isDel bool) (err error) {
	if !m.okSi(si) {
		return
	}
	panic("SwIfAddDel " + si.Name(v))
	return
}

func (m *Main) SwIfAdminUpDown(v *vnet.Vnet, si vnet.Si, isUp bool) (err error) {
	if !m.okSi(si) {
		return
	}
	panic("SwIfAdminUpDown " + si.Name(v))
	return
}

func (m *Main) HwIfLinkUpDown(v *vnet.Vnet, hi vnet.Hi, isUp bool) (err error) {
	if !m.okHi(hi) {
		return
	}
	panic("HwIfLinkUpDown " + hi.Name(v))
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

func (m *Main) Configure(in *parse.Input) {
	// Suitable defaults for an Ethernet-like tun/tap device.
	if m.mtu_bytes == 0 {
		m.mtu_bytes = 4096 + 256
	}

	for !in.End() {
		switch {
		case in.Parse("mtu %d", &m.mtu_bytes):
		case in.Parse("tap"):
			m.isTap = true
		case in.Parse("tun"):
			m.isTap = false
		case in.Parse("en*able"):
			m.disable = false
		case in.Parse("dis*able"):
			m.disable = true
		default:
			panic(parse.ErrInput)
		}
	}
}
