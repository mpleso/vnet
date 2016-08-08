package unix

import (
	"github.com/platinasystems/elib/loop"
	"github.com/platinasystems/netlink"

	"fmt"
)

type netlinkMain struct {
	loop.Node
	m *Main
	s *netlink.Socket
	c chan netlink.Message
}

type netlinkEvent struct {
	m   *Main
	msg netlink.Message
}

func (m *netlinkMain) EventHandler() {}

func (e *netlinkEvent) String() string { return "netlink-message" }

func (e *netlinkEvent) EventAction() {
	var err error
	vn := e.m.v
	switch v := e.msg.(type) {
	case *netlink.IfInfoMessage:
		intf := e.m.getInterface(v.Index)
		// Respect flag admin state changes from unix shell via ifconfig or "ip link" commands.
		err = intf.si.SetAdminUp(vn, v.Flags&netlink.IFF_UP != 0)
	default:
		err = fmt.Errorf("unkown")
	}
	if err != nil {
		e.m.v.Logf("netlink %s: %s\n", err, e.msg.String())
	}
}

// Ignore non-tuntap interfaces (e.g. eth0).
func (m *Main) getInterface(ifindex uint32) (intf *Interface) {
	intf = m.ifByIndex[int(ifindex)]
	return
}
func (m *Main) knownInterface(i uint32) bool { return nil != m.getInterface(i) }

func (m *Main) msgGeneratesEvent(msg netlink.Message) (ok bool) {
	ok = true
	switch v := msg.(type) {
	case *netlink.IfInfoMessage:
		ok = m.knownInterface(v.Index)
	case *netlink.IfAddrMessage:
		ok = m.knownInterface(v.Index)
	case *netlink.RouteMessage:
		ok = m.knownInterface(uint32(v.Attrs[netlink.RTA_OIF].(netlink.Uint32Attr)))
	case *netlink.NeighborMessage:
		ok = m.knownInterface(v.Index)
	case *netlink.DoneMessage:
		ok = false // ignore done messages
	default:
		panic("unknown netlink message")
	}
	return
}

func (m *Main) listener(l *loop.Loop) {
	nm := &m.netlinkMain
	for msg := range nm.c {
		if m.msgGeneratesEvent(msg) {
			l.AddEvent(&netlinkEvent{m: m, msg: msg}, nm)
		}
	}
}

func (nm *netlinkMain) LoopInit(l *loop.Loop) {
	go nm.s.Listen()
	go nm.m.listener(l)
}

func (nm *netlinkMain) Init(m *Main) (err error) {
	nm.m = m
	l := nm.m.v.GetLoop()
	l.RegisterNode(nm, "netlink-listener")
	nm.c = make(chan netlink.Message, 64)
	nm.s, err = netlink.New(nm.c)
	return
}
