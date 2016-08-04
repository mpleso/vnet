package tuntap

import (
	"github.com/platinasystems/elib/iomux"
	"github.com/platinasystems/vnet"
	"github.com/platinasystems/vnet/ethernet"

	"fmt"
	"syscall"
	"unsafe"
)

type nodeMain struct {
	packetPool chan *packet
}

func (m *nodeMain) Init() {
	m.packetPool = make(chan *packet, 64)
}

type node struct {
	readyPackets chan *packet
	ethernet.Interface
	vnet.InterfaceNode
	i *Interface
}

type rxNext int

const (
	rxNextTx rxNext = iota
)

func (intf *Interface) interfaceNodeInit(m *Main) {
	ifName := intf.Name()
	vnetName := ifName + " unix"
	n := &intf.node
	n.i = intf
	n.readyPackets = make(chan *packet, 64)
	m.v.RegisterHwInterface(n, vnetName)
	n.Next = []string{
		rxNextTx: ifName,
	}
	m.v.RegisterInterfaceNode(n, n.Hi(), vnetName)
	iomux.Add(intf)
}

func (n *node) GetHwInterfaceCounters(nm *vnet.InterfaceCounterNames, t *vnet.InterfaceThread) {}
func (n *node) ValidateSpeed(speed vnet.Bandwidth) (err error)                                 { return }

type iovec syscall.Iovec

//go:generate gentemplate -d Package=tuntap -id iovec -d VecType=iovecVec -d Type=iovec github.com/platinasystems/elib/vec.tmpl

func rwv(fd int, iov []iovec, isWrite bool) (n int, err error) {
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

func readv(fd int, iov []iovec) (int, error)  { return rwv(fd, iov, false) }
func writev(fd int, iov []iovec) (int, error) { return rwv(fd, iov, true) }

type packet struct {
	iovs  iovecVec
	chain vnet.RefChain
	refs  vnet.RefVec
}

func (p *packet) sizeForInterface(m *Main, intf *Interface) {
	n := intf.mtuBuffers
	p.iovs.Validate(n - 1)
	p.refs.Validate(n - 1)
	p.iovs = p.iovs[:n]
	p.refs = p.refs[:n]
	m.bufferPool.AllocRefs(&p.refs[0].RefHeader, n)
	for i := uint(0); i < n; i++ {
		p.iovs[i].Base = (*byte)(p.refs[i].Data())
		p.iovs[i].Len = uint64(m.bufferPool.Size)
	}
}

func (m *Main) getPacket(intf *Interface) (p *packet) {
	select {
	case p = <-m.packetPool:
	default:
		p = &packet{}
	}
	p.sizeForInterface(m, intf)
	return
}

func (m *Main) putPacket(p *packet) { m.packetPool <- p }

func (n *node) InterfaceInput(o *vnet.RefOut) {
	m := n.i.m
	toTx := &o.Outs[rxNextTx]
	toTx.BufferPool = m.bufferPool
	t := n.GetIfThread()
	nPackets, nBytes := uint(0), uint(0)
loop:
	for {
		select {
		case p := <-n.readyPackets:
			nBytes += p.chain.Len()
			toTx.Refs[nPackets] = p.chain.Done()
			nPackets++
			if nPackets >= uint(len(toTx.Refs)) {
				break loop
			}
		default:
			break loop
		}
	}
	vnet.IfRxCounter.Add(t, n.Si(), nPackets, nBytes)
	toTx.SetLen(m.v, nPackets)
	n.Activate(false)
}

func (n *node) InterfaceOutput(i *vnet.RefVecIn, free chan *vnet.RefVecIn) {
	panic("not yet")
}

func (intf *Interface) ErrorReady() (err error)   { return }
func (intf *Interface) WriteReady() (err error)   { return }
func (intf *Interface) WriteAvailable() (ok bool) { return }

func (intf *Interface) ReadReady() (err error) {
	m := intf.m
	p := m.getPacket(intf)
	var nRead int
	nRead, err = readv(intf.Fd, p.iovs)
	if err != nil {
		t := intf.node.GetIfThread()
		vnet.IfDrops.Add(t, intf.node.Si(), 1)
		return
	}
	size := m.bufferPool.Size
	nLeft := uint(nRead)
	for i := 0; nLeft > 0; i++ {
		l := size
		if nLeft < l {
			l = nLeft
		}
		p.refs[i].SetDataLen(l)
		p.chain.Append(&p.refs[i])
		nLeft -= l
	}
	intf.node.readyPackets <- p
	intf.node.Activate(true)
	return
}
