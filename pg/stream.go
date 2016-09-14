package pg

import (
	"github.com/platinasystems/vnet"

	"fmt"
)

type Streamer interface {
	get_stream() *Stream
	Del()
	PacketData() []byte
}

func (s *Stream) get_stream() *Stream { return s }
func (s *Stream) Del()                {}

type stream_config struct {
	random_size bool
	// Min, max packet size.
	min_size uint
	max_size uint
	// Number of packets to send or 0 for no limit.
	n_packets_limit uint64
	// Next index relative to input node for this stream.
	next uint
}

type Stream struct {
	name  string
	index uint
	r     Streamer

	random_seed int64

	cur_size uint

	n_packets_sent uint64

	data []byte

	stream_config
}

//go:generate gentemplate -d Package=pg -id stream -d PoolType=stream_pool -d Type=Streamer -d Data=elts github.com/platinasystems/elib/pool.tmpl

func (s *Stream) GetSize() uint { return s.cur_size }
func (s *Stream) NextSize(i uint) uint {
	if x := s.cur_size + 1 + i; x <= s.max_size {
		return x
	} else {
		return s.min_size + i
	}
}

func (s *Stream) SetData() {
	if s.max_size < s.min_size {
		s.max_size = s.min_size
	}
	s.cur_size = s.min_size
	s.data = s.r.PacketData()
}

func (n *node) get_stream(i uint) Streamer { return n.stream_pool.elts[i] }
func (n *node) get_stream_by_name(name string) (r Streamer) {
	if i, ok := n.stream_index_by_name[name]; ok {
		r = n.get_stream(i)
	}
	return
}

func (n *node) new_stream(r Streamer, format string, args ...interface{}) {
	name := fmt.Sprintf(format, args...)
	si, ok := n.stream_index_by_name[name]
	if ok {
		x := n.get_stream(si)
		x.Del()
	} else {
		si = n.stream_pool.GetIndex()
		n.stream_index_by_name.Set(name, si)
	}

	n.stream_pool.elts[si] = r
	s := r.get_stream()
	s.r = r
	s.index = si
	s.name = name
	return
}

func (s *Stream) clean() {
	s.data = nil
	s.name = ""
	s.r = nil
}

func (n *node) del_stream(r Streamer) {
	s := r.get_stream()
	n.stream_pool.PutIndex(s.index)
	delete(n.stream_index_by_name, s.name)
	s.index = ^uint(0)
	s.r.Del()
	s.clean()
}

func (n *node) ValidateSpeed(speed vnet.Bandwidth) (err error)                                 { return }
func (n *node) GetHwInterfaceCounters(nm *vnet.InterfaceCounterNames, t *vnet.InterfaceThread) {}
func (n *node) FormatRewrite(rw *vnet.Rewrite) string                                          { return "" }
func (n *node) SetRewrite(v *vnet.Vnet, rw *vnet.Rewrite, packetType vnet.PacketType, da []byte) {
}
