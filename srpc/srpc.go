// Package srpc gives symmetric bi-directional RPC on top of GO rpc.
// Two hosts connected via are Client and server both expose RPC calls to each other.
// Either host may make RPC calls to the other: so each host is both client and server at the same time.
package srpc

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/rpc"
	"sync"

	"github.com/platinasystems/elib"
	"github.com/platinasystems/elib/elog"
)

type input struct {
	buf     []byte
	recycle bool
}

type side struct {
	wbuf     elib.ByteVec
	overflow elib.ByteVec
	rx       chan input
}

type conn struct {
	wlock         sync.Mutex
	wc            io.WriteCloser
	recycle       chan elib.ByteVec
	sides         [2]side
	eventTagIndex int
}

// Get buffer from recycler or return nil if none is available.
func (c *conn) getBuf() (b elib.ByteVec) {
	select {
	case b = <-c.recycle:
		b = b[:0]
	default:
	}
	return
}

// Recycle buffer.
func (c *conn) putBuf(b elib.ByteVec) {
	select {
	case c.recycle <- b:
	default:
	}
}

type Server struct {
	r    io.Reader
	conn conn
	c    ClientConn
	s    ServerConn
	*rpc.Client
	*rpc.Server
	EventTag string
}

func (c *conn) Read(p []byte, isClient int) (n int, err error) {
	log := elog.Enabled()
	if log {
		e, o := c.logf(eventFlag(0), isClient, "enter")
		defer func() { c.logExit(e, p, n, o, err) }()
	}

	s := &c.sides[isClient]

	// Copy from overflow.
	if len(s.overflow) > 0 {
		n = copy(p, s.overflow)
		if len(s.overflow) == n {
			s.overflow = s.overflow[:0]
		} else {
			s.overflow = s.overflow[n:]
		}
		return
	}

	// Wait for next frame sent from input routine.
	i := <-s.rx
	n = copy(p, i.buf)
	if n < len(i.buf) {
		s.overflow = append(s.overflow, i.buf[n:]...)
	}
	if i.recycle {
		c.putBuf(i.buf)
	}
	return
}

func (c *conn) Write(p []byte, isClient int) (n int, err error) {
	c.wlock.Lock()
	defer c.wlock.Unlock()

	if elog.Enabled() {
		e, o := c.logf(IsWrite, isClient, "enter")
		defer func() { c.logExit(e, p, n, o, err) }()
	}

	side := &c.sides[isClient]

	n = len(p)
	wb := framePut(side.wbuf, n, isClient)
	wb = append(wb, p...)

	l := 0
	for l < len(wb) {
		var i int
		i, err = c.wc.Write(wb[l:])
		if err != nil {
			return
		}
		l += i
	}
	// Reuse it for next call.
	side.wbuf = wb[:0]
	return
}

func (c *conn) Close(isClient int) error {
	close(c.sides[isClient^1].rx)
	return c.wc.Close()
}

type ClientConn struct{ *conn }
type ServerConn struct{ *conn }

func (c *ClientConn) Read(p []byte) (n int, err error)  { return c.conn.Read(p, 1) }
func (c *ServerConn) Read(p []byte) (n int, err error)  { return c.conn.Read(p, 0) }
func (c *ClientConn) Write(b []byte) (n int, err error) { return c.conn.Write(b, 1) }
func (c *ServerConn) Write(b []byte) (n int, err error) { return c.conn.Write(b, 0) }
func (c *ClientConn) Close() error                      { return c.conn.Close(1) }
func (c *ServerConn) Close() error                      { return c.conn.Close(0) }

// Framing encodes length plus whether frame is from client or server side.
func framePut(b []byte, l int, isClient int) []byte {
	vi := uint64(0)
	if isClient != 0 {
		vi = 1
	}
	vi += 2 * uint64(l)
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(tmp[:], vi)
	for i := 0; i < n; i++ {
		b = append(b, tmp[i])
	}
	return b
}

func frameGet(b []byte) (isClient int, l int, n int) {
	vi, n := binary.Uvarint(b)
	isClient = int(vi & 1)
	l = int(vi >> 1)
	return
}

func (r *Server) Input(b []byte) (n int) {
	var lastContentLen, lastFrameIsClient int
	nLeft := len(b)

	c := &r.conn
	log := elog.Enabled()
	if log {
		e := inputEvent{}
		t := elog.PutUvarint(e.s[:], c.eventTagIndex)
		elog.PutData(t, b)
		e.Log()
	}

	for {
		frameIsClient, contentLen, headerLen := frameGet(b[n:])
		frameLen := contentLen + headerLen

		frameValid := headerLen > 0 && nLeft >= frameLen

		if lastContentLen > 0 {
			// Client frames go to server;
			// server frames are read by client.
			s := &r.conn.sides[lastFrameIsClient^1]
			i := input{buf: b[n-lastContentLen : n]}

			// Only recycle buffer for last frame in buffer.
			i.recycle = !frameValid
			s.rx <- i
		}

		if !frameValid {
			break
		}

		lastContentLen = contentLen
		lastFrameIsClient = frameIsClient
		n += frameLen
		nLeft -= frameLen
	}
	return
}

func (r *Server) reader() {
	var b elib.ByteVec
	for {
		i := len(b)
		b.Resize(4096)
		n, err := r.r.Read(b[i:])
		if err != nil {
			panic(err)
		}
		b = b[0 : i+n]
		l := r.Input(b)
		nextBuf := r.conn.getBuf()
		if l < len(b) {
			nextBuf = append(nextBuf, b[l:]...)
		}
		b = nextBuf
	}
}

func (r *Server) init(wc io.WriteCloser, regs []interface{}) {
	c := &r.conn
	c.wc = wc
	for i := range c.sides {
		c.sides[i].rx = make(chan input, 64)
	}
	r.c.conn = c
	r.s.conn = c
	r.Client = rpc.NewClient(&r.c)
	r.Server = rpc.NewServer()
	for i := range regs {
		r.Register(regs[i])
	}

	if len(r.EventTag) > 0 {
		c.eventTagIndex = eventType.TagIndex(r.EventTag)
	}

	go r.Server.ServeConn(&r.s)
}

func (r *Server) Init(rwc io.ReadWriteCloser, regs ...interface{}) {
	r.r = rwc
	r.conn.recycle = make(chan elib.ByteVec, 64)
	r.init(rwc, regs)
	go r.reader()
}

func (r *Server) InitWriter(wc io.WriteCloser, recycle chan elib.ByteVec, regs ...interface{}) {
	r.r = nil
	r.conn.recycle = recycle
	r.init(wc, regs)
}

// Event logging.
type event struct {
	flags eventFlag
	s     [elog.EventDataBytes - 1]byte
}

//go:generate gentemplate -d Package=srpc -id event -d Type=event github.com/platinasystems/elib/elog/event.tmpl

type eventFlag uint8

const (
	IsWrite eventFlag = 1 << iota
	IsClient
	IsData
)

func (c *conn) logf(f eventFlag, isClient int, format string, args ...interface{}) (e event, ti int) {
	e = event{flags: f}
	if isClient != 0 {
		e.flags |= IsClient
	}
	ti = binary.PutUvarint(e.s[:], uint64(c.eventTagIndex))
	elog.Printf(e.s[ti:], format, args...)
	e.Log()
	return
}

func (c *conn) logExit(e event, p []byte, n, o int, err error) {
	if err != nil {
		elog.Printf(e.s[o:], "error %s", err)
	} else {
		e.setData(p, n, o)
	}
	e.Log()
}

func (e *event) setData(p []byte, n, o int) {
	e.flags |= IsData
	i := binary.PutUvarint(e.s[o:], uint64(n))
	copy(e.s[o+i:], p[:n])
}

func (e *event) String() string {
	side := "server"
	rw := "read"
	if e.flags&IsClient != 0 {
		side = "client"
	}
	if e.flags&IsWrite != 0 {
		rw = "write"
	}
	b, tagIndex := elog.Uvarint(e.s[:])
	tag := eventType.Tag(int(tagIndex), " ")
	if e.flags&IsData != 0 {
		return fmt.Sprintf("rpc %s%s %s %s", tag, side, rw, elog.HexData(b))
	} else {
		return fmt.Sprintf("rpc %s%s %s %s", tag, side, rw, elog.String(b))
	}
}

type inputEventFlag uint8

const (
	isFrame inputEventFlag = 1 << iota
)

type inputEvent struct {
	flags inputEventFlag
	s     [elog.EventDataBytes - 1]byte
}

//go:generate gentemplate -d Package=srpc -id inputEvent -d Type=inputEvent github.com/platinasystems/elib/elog/event.tmpl

func (e *inputEvent) String() string {
	b, tagIndex := elog.Uvarint(e.s[:])
	tag := eventType.Tag(int(tagIndex), " ")
	s := ""
	if e.flags&isFrame != 0 {
		s = "frame "
	}
	return fmt.Sprintf("rpc %sinput %s%s", tag, s, elog.HexData(b))
}
