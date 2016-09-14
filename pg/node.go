package pg

import (
	"github.com/platinasystems/elib/hw"
	"github.com/platinasystems/elib/parse"
	"github.com/platinasystems/vnet"
)

const (
	next_error = iota
	next_punt
	n_next
)

const (
	error_none = iota
	tx_packets_dropped
)

type node struct {
	vnet.InterfaceNode
	vnet.HwIf
	v    *vnet.Vnet
	pool hw.BufferPool
	stream_pool
	stream_index_by_name parse.StringMap
	stream_type_map      parse.StringMap
	stream_types         []StreamType
}

func (n *node) init(v *vnet.Vnet) {
	n.v = v
	n.Next = []string{
		next_error: "error",
		next_punt:  "punt",
	}
	n.Errors = []string{
		error_none:         "packets generated",
		tx_packets_dropped: "tx packets dropped",
	}
	v.RegisterHwInterface(n, "packet-generator")
	v.RegisterInterfaceNode(n, n.Hi(), "packet-generator")

	// Link is always up for packet generator.
	n.SetLinkUp(true)
	n.SetAdminUp(true)

	t := &n.pool.BufferTemplate
	*t = *hw.DefaultBufferTemplate
	t.Size = 16 << 10
	n.pool.Name = n.Name()
	v.AddBufferPool(&n.pool)
}

func (n *node) InterfaceInput(o *vnet.RefOut) {
	s := n.get_stream(0).get_stream() // fixme
	out := &o.Outs[s.next]
	out.BufferPool = &n.pool
	t := n.GetIfThread()

	cap := out.Cap()
	np := cap
	if s.n_packets_limit != 0 {
		np = 0
		if s.n_packets_sent < s.n_packets_limit {
			max := s.n_packets_limit - s.n_packets_sent
			if max > uint64(cap) {
				np = cap
			} else {
				np = uint(max)
			}
		}
	}

	out.AllocPoolRefs(&n.pool, np)
	rs := out.Refs[:]
	nBytes := uint(0)
	for i := uint(0); i < np; i++ {
		r := &rs[i]
		n.SetError(r, error_none)
		r.SetDataLen(s.cur_size)
		nBytes += s.cur_size
		s.cur_size = s.NextSize(0)
	}
	vnet.IfRxCounter.Add(t, n.Si(), np, nBytes)
	out.SetPoolAndLen(n.Vnet, &n.pool, np)
	s.n_packets_sent += uint64(np)
	if s.n_packets_limit != 0 {
		n.Activate(s.n_packets_sent < s.n_packets_limit)
	}
}

func (n *node) InterfaceOutput(i *vnet.TxRefVecIn) {
	n.CountError(tx_packets_dropped, i.NPackets())
	n.Vnet.FreeTxRefIn(i)
}
