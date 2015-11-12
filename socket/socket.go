package socket

import (
	"github.com/platinasystems/elib"
	"github.com/platinasystems/elib/elog"
	"github.com/platinasystems/elib/iomux"
	"github.com/platinasystems/vnet/ip4"

	"fmt"
	"sync"
	"syscall"
)

type socket struct {
	iomux.File

	flags Flags

	maxReadBytes uint

	txBufLock          sync.Mutex
	txBuffer, RxBuffer elib.ByteVec

	SelfAddr, PeerAddr syscall.Sockaddr
}

type Server struct {
	socket
}
type Client struct {
	socket
}

type Flags uint32

const (
	// Listen (server side) rather than connect (client side)
	Listen Flags = 1 << iota
	// Client accepted by server.
	AcceptedClient
	// Non-blocking connect in progress.
	ConnectInProgress
	// Use UDP instead of TCP
	UDP
	TCPDelay // set to enable Nagle algorithm (default is disabled)
)

func tst(err error, tag string) error {
	if err != nil {
		err = fmt.Errorf("%s %s", tag, err)
	}
	return err
}

func (s *socket) Close() (err error) {
	err = syscall.Close(s.Fd)
	if err != nil {
		err = fmt.Errorf("close: %s", err)
	}
	return
}

func (s *socket) WriteAvailable() bool { return len(s.txBuffer) > 0 || s.flags&ConnectInProgress != 0 }

func (s *socket) ReadReady() (err error) {
	i := len(s.RxBuffer)

	if s.maxReadBytes <= 0 {
		s.maxReadBytes = 4 << 10
	}
	s.RxBuffer.Resize(s.maxReadBytes)

	var n int
	n, err = syscall.Read(s.Fd, s.RxBuffer[i:])
	if err != nil {
		switch err {
		case syscall.EAGAIN:
			err = nil
			return
		}
		err = tst(err, "read")
		return
	}
	s.RxBuffer = s.RxBuffer[:i+n]

	if elog.Enabled() {
		s.elogData(Read, s.RxBuffer)
	}

	if n == 0 {
		iomux.Del(s)
		s.Close()
	}
	return
}

func (s *Server) AcceptClient(c *Client) (err error) {
	fd, sa, err := syscall.Accept(s.Fd)
	if err != nil {
		return
	}
	c.flags = AcceptedClient
	c.Fd = fd
	c.SelfAddr = s.SelfAddr
	c.PeerAddr = sa
	return
}

func (s *Server) ReadReady() (err error) {
	c := &Client{}
	err = s.AcceptClient(c)
	return
}

func (s *socket) ClientWriteReady() (newConnection bool, err error) {
	needUpdate := false
	defer func() {
		if needUpdate {
			iomux.Update(s)
		}
	}()
	newConnection = s.flags&ConnectInProgress != 0
	if newConnection {
		s.flags &^= ConnectInProgress
		var errno int
		errno, err = syscall.GetsockoptInt(s.Fd, syscall.SOL_SOCKET, syscall.SO_ERROR)
		if err = tst(err, "getsockopt"); err != nil {
			return
		}
		if errno != 0 {
			err = fmt.Errorf("connect: %s", syscall.Errno(errno))
			return
		}
		// Update since connection in progress implies write available.
		needUpdate = true
	}

	if len(s.txBuffer) > 0 {
		var n int

		s.txBufLock.Lock()
		defer s.txBufLock.Unlock()
		n, err = syscall.Write(s.Fd, s.txBuffer)
		if err != nil {
			err = tst(err, "write")
			return
		}
		l := len(s.txBuffer)
		elog.Gen("socket write #%d %d %d %x", s.File.Index(), n, l, s.txBuffer[0:n])
		switch {
		case n == l:
			s.txBuffer = s.txBuffer[:0]
		case n > 0:
			copy(s.txBuffer, s.txBuffer[n:])
			s.txBuffer = s.txBuffer[:l-n]
		}
		// Whole buffer written => toggle write available.
		needUpdate = true
	}
	return
}

func (s *socket) WriteReady() (err error) {
	_, err = s.ClientWriteReady()
	return
}

func (s *socket) Write(p []byte) (n int, err error) {
	s.txBufLock.Lock()
	defer s.txBufLock.Unlock()
	i := len(s.txBuffer)
	n = len(p)
	if n > 0 {
		s.txBuffer.Resize(uint(n))
		copy(s.txBuffer[i:i+n], p)
		iomux.Update(s)
	}
	return
}

func (s *socket) TxBuf() elib.ByteVec { return s.txBuffer }
func (s *socket) TxLen() int          { return len(s.txBuffer) }

func (s *socket) ErrorReady() (err error) {
	panic(s)
}

/* Return and bind to an unused port. */
func (s *socket) bindFreePort(a []byte) (port int, err error) {
	// 5000 => IPPORT_USERRESERVED
	for port = 5000; port < 1<<16; port++ {
		switch len(a) {
		case 4:
			sa := &syscall.SockaddrInet4{Port: port}
			copy(sa.Addr[:], a)
			err = syscall.Bind(s.Fd, sa)
		case 16:
			sa := &syscall.SockaddrInet6{Port: port}
			copy(sa.Addr[:], a)
			err = syscall.Bind(s.Fd, sa)
		default:
			panic(a)
		}
		if err == nil {
			return
		}
	}

	err = fmt.Errorf("bind: reached maximum port")
	return
}

func (s *socket) Config(cfg string, flags Flags) (err error) {
	var sa syscall.Sockaddr

	/* Anything that begins with a / is a local Unix file socket. */
	if cfg[0] == '/' {
		sa = &syscall.SockaddrUnix{Name: cfg}
	} else {
		var a ip4.Socket
		if _, err = fmt.Sscanf(cfg, "%s", &a); err == nil {
			sa = &syscall.SockaddrInet4{Addr: a.Address, Port: int(a.Port)}
		} else {
			err = fmt.Errorf("failed to parse valid config: %s %s", cfg, err)
			return
		}
	}

	// Sanitize flags.
	flags &= Listen | UDP | TCPDelay

	kind := syscall.SOCK_STREAM
	if flags&UDP != 0 {
		kind = syscall.SOCK_DGRAM
	}
	s.Fd, err = syscall.Socket(syscall.AF_INET, kind, 0)
	if err = tst(err, "socket"); err != nil {
		return
	}

	nodelay := 1
	if flags&TCPDelay != 0 {
		nodelay = 0
	}
	err = syscall.SetsockoptInt(s.Fd, syscall.IPPROTO_TCP, syscall.TCP_NODELAY, nodelay)
	if err = tst(err, "setsockopt TCP_NODELAY"); err != nil {
		goto out
	}

	if flags&Listen != 0 {
		needBind := true
		switch v := sa.(type) {
		case *syscall.SockaddrInet4:
			if ip4.Port(v.Port) == ip4.NilPort {
				v.Port, err = s.bindFreePort(v.Addr[:])
				if err != nil {
					goto out
				}
				needBind = false
			}
		case *syscall.SockaddrUnix:
			syscall.Unlink(v.Name)
		default:
			panic(v)
		}

		err = syscall.SetsockoptInt(s.Fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
		if err = tst(err, "setsockopt SO_REUSEADDR"); err != nil {
			goto out
		}

		if needBind {
			err = syscall.Bind(s.Fd, sa)
			if err = tst(err, "bind"); err != nil {
				goto out
			}
		}

		err = syscall.Listen(s.Fd, syscall.SOMAXCONN)
		if err = tst(err, "listen"); err != nil {
			goto out
		}

		s.SelfAddr = sa
	} else {
		s.PeerAddr = sa

		err = syscall.Connect(s.Fd, sa)
		if err = tst(err, "connect"); err != nil {
			goto out
		}

		s.SelfAddr, err = syscall.Getsockname(s.Fd)
		if err = tst(err, "getsockname"); err != nil {
			goto out
		}
		flags |= ConnectInProgress
	}

	s.flags = flags
	return

out:
	s.Close()
	return
}

func NewServer(cfg string) (s *Server, err error) {
	s = &Server{}
	err = s.Config(cfg, Listen)
	return
}

func NewClient(cfg string) (s *Client, err error) {
	s = &Client{}
	err = s.Config(cfg, 0)
	return
}

func SockaddrString(a syscall.Sockaddr) string {
	switch v := a.(type) {
	case *syscall.SockaddrInet4:
		s := ip4.Socket{Address: v.Addr, Port: ip4.Port(v.Port)}
		return s.String()
	case *syscall.SockaddrUnix:
		return fmt.Sprintf("unix:%s", v.Name)
	default:
		panic(v)
	}
}

func (s *socket) String() string {
	return fmt.Sprintf("%s -> %s", SockaddrString(s.SelfAddr), SockaddrString(s.PeerAddr))
}

// Event logging.
type event struct {
	flags eventFlag
	s     [elog.EventDataBytes - 1]byte
}

//go:generate gentemplate -d Package=socket -id event -d Type=event github.com/platinasystems/elib/elog/event.tmpl

type eventFlag uint8

const (
	// low 4 bits are op code
	Read   eventFlag = 0
	Write  eventFlag = 1
	IsData eventFlag = 1 << (iota + 4)
)

var opNames = []string{
	Read:  "read",
	Write: "write",
}

func (s *socket) elogf(f eventFlag, format string, args ...interface{}) (e event) {
	e = event{flags: f}
	b := elog.PutUvarint(e.s[:], int(s.File.Index()))
	elog.Printf(b, format, args...)
	e.Log()
	return
}

func (s *socket) elogData(f eventFlag, p []byte) (e event) {
	e = event{flags: f | IsData}
	b := elog.PutUvarint(e.s[:], int(s.File.Index()))
	b = elog.PutUvarint(b, len(p))
	copy(b, p)
	e.Log()
	return
}

func (e *event) String() string {
	op := opNames[e.flags&0xf]
	var d string
	b := e.s[:]
	b, fi := elog.Uvarint(b)
	if e.flags&IsData != 0 {
		d = elog.HexData(b)
	} else {
		d = elog.String(b)
	}
	return fmt.Sprintf("socket #%d %s %s", fi, op, d)
}
