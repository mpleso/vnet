package pg

import (
	"github.com/platinasystems/elib"
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
	pool vnet.BufferPool
	stream_pool
	stream_index_by_name parse.StringMap
	stream_type_map      parse.StringMap
	stream_types         []StreamType
	buffer_type_pool
	orphan_refs vnet.RefVec
	node_validate
}

const (
	buffer_type_nil = 0xffffffff
)

type buffer_type struct {
	index             uint32
	stream_index      uint
	data_index        uint
	data              []byte
	free_refs         vnet.RefVec
	validate_sequence uint
}

//go:generate gentemplate -d Package=pg -id buffer_type_pool -d PoolType=buffer_type_pool -d Type=buffer_type -d Data=elts github.com/platinasystems/elib/pool.tmpl

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

	p := &n.pool
	t := &p.BufferTemplate
	*t = vnet.DefaultBufferPool.BufferTemplate
	r := p.GetRefTemplate()
	n.SetError(r, error_none)
	t.Buffer.SetSave(buffer_type_nil)
	p.Name = n.Name()
	v.AddBufferPool(p)
}

func (n *node) free_buffer_type(t *buffer_type) {
	if l := t.free_refs.Len(); l > 0 {
		// Mark buffers as no longer being of this type.
		for i := range t.free_refs {
			b := t.free_refs[i].GetBuffer()
			b.SetSave(buffer_type_nil)
		}
		n.pool.FreeRefs(&t.free_refs[0], l, false)
	}
	t.free_refs = t.free_refs[:0]
	t.data = nil
}

func buffer_type_for_size(size, unit uint) (n uint) {
	for size > unit {
		n++
		size -= unit
	}
	return
}

func (n *node) setData(s *Stream) {
	// Return cached refs in pool.
	{
		rs := n.pool.AllocCachedRefs()
		n.free_buffers(rs, nil)
	}

	// Free previously used buffer types.
	for _, t := range s.buffer_types {
		n.free_buffer_type(&n.buffer_type_pool.elts[t])
		n.buffer_type_pool.PutIndex(uint(t))
	}

	n_data := uint(len(s.data))
	size := n.pool.BufferTemplate.Size
	n_size := 1 + buffer_type_for_size(n_data, size)
	s.buffer_types.Validate(n_size - 1)
	s.buffer_types = s.buffer_types[:n_size]
	i, j := uint(0), uint(0)
	for i < n_data {
		this_size := size
		if i+this_size > n_data {
			this_size = n_data - i
		}
		bi := uint32(n.buffer_type_pool.GetIndex())
		s.buffer_types[j] = bi
		t := &n.buffer_type_pool.elts[bi]
		t.index = bi
		t.stream_index = s.index
		t.data_index = j
		t.data = s.data[i : i+this_size]
		j++
		i += this_size
	}
}

func (n *node) free_buffers(refs []vnet.Ref, t *buffer_type) {
	for i := range refs {
		r0 := &refs[i]
		b0 := r0.GetBuffer()
		ti0 := uint32(b0.GetSave())
		if ti0 == buffer_type_nil {
			if t != nil {
				ti0 = t.index
				r0.SetDataLen(uint(len(t.data)))
				n.SetError(r0, error_none)
				copy(r0.DataSlice(), t.data)
				b0.SetSave(hw.BufferSave(ti0))
			} else {
				n.orphan_refs = append(n.orphan_refs, *r0)
				continue
			}
		}
		t0 := &n.buffer_type_pool.elts[ti0]
		r0.SetDataLen(uint(len(t0.data)))
		t0.validate_ref(r0)
		t0.free_refs = append(t0.free_refs, *r0)
	}
	if l := n.orphan_refs.Len(); l > 0 {
		n.pool.FreeRefs(&n.orphan_refs[0], l, false)
		n.orphan_refs = n.orphan_refs[:0]
	}
}

func (n *node) buffer_type_get_refs(dst []vnet.Ref, want, ti uint) {
	t := &n.buffer_type_pool.elts[ti]
	var got uint
	for {
		if got = t.free_refs.Len(); got >= want {
			break
		}
		var tmp [vnet.MaxVectorLen]vnet.Ref
		n.pool.AllocRefs(tmp[:])
		n.free_buffers(tmp[:], t)
	}

	copy(dst, t.free_refs[got-want:got])

	if elib.Debug {
		for i := uint(0); i < want; i++ {
			t.validate_ref(&dst[i])
		}
	}

	t.free_refs = t.free_refs[:got-want]
	return
}

type node_validate struct {
	validate_data     []byte
	validate_sequence uint
}

func (n *node) generate_n_types(s *Stream, dst []vnet.Ref, n_packets, n_types uint) (n_bytes uint) {
	var tmp [4][vnet.MaxVectorLen]vnet.Ref
	var prev, prev_prev []vnet.Ref
	this := dst
	save := s.cur_size
	d := (n_types - 1) * n.pool.Size
	n_bytes = d * n_packets
	for i := uint(0); i < n_types; i++ {
		n.buffer_type_get_refs(this, n_packets, uint(s.buffer_types[i]))
		if i+1 >= n_types {
			for j := uint(0); j < n_packets; j++ {
				last_size := s.cur_size - d
				this[j].SetDataLen(last_size)
				n_bytes += last_size
				s.cur_size = s.next_size(s.cur_size, 0)
			}
		}
		if prev != nil {
			var pp *hw.RefHeader
			if prev_prev != nil {
				pp = &prev_prev[0].RefHeader
			}
			hw.LinkRefs(pp, &prev[0].RefHeader, &this[0].RefHeader, 1+i, n_packets)
		}
		prev_prev = prev
		prev = this
		this = tmp[i&3][:]
	}

	if elib.Debug {
		save, s.cur_size = s.cur_size, save
		for i := uint(0); i < n_packets; i++ {
			n.validate_ref(&dst[i], s)
			s.cur_size = s.next_size(s.cur_size, 0)
		}
		s.cur_size = save
	}

	return
}

func (n *node) generate(s *Stream, dst []vnet.Ref, n_packets uint) (n_bytes uint) {
	nt := 1 + buffer_type_for_size(s.cur_size, n.pool.Size)
	n_left := n_packets
	for {
		n_this := n_left
		if s.max_size != s.min_size {
			n_this = 1 + s.max_size - s.cur_size
			if next := 1 + nt*n.pool.Size - s.cur_size; n_this > next {
				n_this = next
			}
			if n_this > n_left {
				n_this = n_left
			}
		}
		n_bytes += n.generate_n_types(s, dst[n_packets-n_left:], n_this, nt)
		n_left -= n_this
		if n_left == 0 {
			break
		}
		nt++
		if nt > uint(len(s.buffer_types)) {
			nt = 1
		}
	}
	return
}

func (s *Stream) n_packets_this_input(cap uint) (n uint) {
	if s.n_packets_limit == 0 { // unlimited
		n = cap
	} else if s.n_packets_sent < s.n_packets_limit {
		max := s.n_packets_limit - s.n_packets_sent
		if max > uint64(cap) {
			n = cap
		} else {
			n = uint(max)
		}
	}
	return
}

func (n *node) stream_input(o *vnet.RefOut, s *Stream) (done bool) {
	out := &o.Outs[s.next]
	out.BufferPool = &n.pool
	t := n.GetIfThread()

	n_packets := s.n_packets_this_input(out.Cap())
	n_bytes := n.generate(s, out.Refs[:], n_packets)
	vnet.IfRxCounter.Add(t, n.Si(), n_packets, n_bytes)
	out.SetPoolAndLen(n.Vnet, &n.pool, n_packets)
	s.n_packets_sent += uint64(n_packets)
	done = s.n_packets_limit != 0 && s.n_packets_sent >= s.n_packets_limit
	return
}

func (n *node) InterfaceInput(o *vnet.RefOut) {
	all_done := true
	n.stream_pool.Foreach(func(s Streamer) {
		done := n.stream_input(o, s.get_stream())
		all_done = all_done && done
	})
	n.Activate(!all_done)
}

func (n *node) InterfaceOutput(i *vnet.TxRefVecIn) {
	n.CountError(tx_packets_dropped, i.NPackets())
	n.Vnet.FreeTxRefIn(i)
}
