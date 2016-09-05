package ixge

import (
	"github.com/platinasystems/elib"
	"github.com/platinasystems/elib/elog"
	"github.com/platinasystems/elib/hw"
	"github.com/platinasystems/vnet"

	"unsafe"
)

type tx_dma_queue struct {
	dma_queue
	tx_descriptors        tx_descriptor_vec
	desc_id               elib.Index
	head_index_write_back *uint32
	tx_fifo               chan tx_in
	tx_irq_fifo           chan tx_in
	current_tx_in         tx_in
	n_current_tx_in       reg
}

//go:generate gentemplate -d Package=ixge -id tx_dma_queue -d VecType=tx_dma_queue_vec -d Type=tx_dma_queue github.com/platinasystems/elib/vec.tmpl

type tx_descriptor struct {
	buffer_address      uint64
	n_bytes_this_buffer uint16
	status0             uint16
	status1             uint32
}

//go:generate gentemplate -d Package=ixge -id tx_descriptor -d Type=tx_descriptor -d VecType=tx_descriptor_vec github.com/platinasystems/elib/hw/dma_mem.tmpl

const (
	tx_desc_status0_log2_is_end_of_packet   = 8 + 0
	tx_desc_status0_is_end_of_packet        = 1 << tx_desc_status0_log2_is_end_of_packet
	tx_desc_status0_insert_crc              = 1 << (8 + 1)
	tx_desc_status0_log2_report_status      = (8 + 3)
	tx_desc_status0_report_status           = (1 << tx_desc_status0_log2_report_status)
	tx_desc_status0_is_advanced             = 1 << (8 + 5)
	tx_desc_status0_vlan_enable             = 1 << (8 + 6)
	tx_desc_status1_is_owned_by_software    = 1 << 0
	tx_desc_status1_insert_tcp_udp_checksum = 1 << (8 + 1)
	tx_desc_status1_insert_ip4_checksum     = 1 << (8 + 0)

	// Only valid if is_advanced is set.
	tx_desc_status0_advanced_context = 2 << 4
	tx_desc_status0_advanced_data    = 3 << 4
)

func (d *dev) tx_dma_init(queue uint) {
	if d.tx_ring_len == 0 {
		d.tx_ring_len = 2 * vnet.MaxVectorLen
	}
	q := d.tx_queues.Validate(queue)
	q.d = d
	q.index = queue
	q.tx_descriptors, q.desc_id = tx_descriptorAlloc(int(d.tx_ring_len))
	q.len = reg(d.tx_ring_len)

	dr := q.get_regs()
	dr.descriptor_address.set(d, uint64(q.tx_descriptors[0].PhysAddress()))
	n_desc := reg(len(q.tx_descriptors))
	dr.n_descriptor_bytes.set(d, n_desc*reg(unsafe.Sizeof(q.tx_descriptors[0])))

	hw.MemoryBarrier()

	// Make sure tx is enabled.
	d.regs.tx_dma_control.or(d, 1<<0)

	q.start(d, &dr.dma_regs)

	{
		const base = tx_desc_status0_insert_crc
		d.tx_desc_status0_by_next_valid_flag[0] = base | tx_desc_status0_is_end_of_packet
		d.tx_desc_status0_by_next_valid_flag[vnet.NextValid] = base
	}
}

func (d *dev) set_tx_descriptor(rs []vnet.Ref, ds []tx_descriptor, ri, di reg) {
	r0, d0 := &rs[ri+0], &ds[di+0]
	d0.buffer_address = uint64(r0.DataPhys())
	d0.n_bytes_this_buffer = uint16(r0.DataLen())
	f0 := r0.NextValidFlag()
	d0.status0 = d.tx_desc_status0_by_next_valid_flag[f0]
	// Owned by hardware.
	d0.status1 = 0
}

func (d *dev) set_4_tx_descriptors(rs []vnet.Ref, ds []tx_descriptor, ri, di reg) {
	r0, r1, r2, r3 := &rs[ri+0], &rs[ri+1], &rs[ri+2], &rs[ri+3]
	d0, d1, d2, d3 := &ds[di+0], &ds[di+1], &ds[di+2], &ds[di+3]

	d0.buffer_address = uint64(r0.DataPhys())
	d1.buffer_address = uint64(r1.DataPhys())
	d2.buffer_address = uint64(r2.DataPhys())
	d3.buffer_address = uint64(r3.DataPhys())

	d0.n_bytes_this_buffer = uint16(r0.DataLen())
	d1.n_bytes_this_buffer = uint16(r1.DataLen())
	d2.n_bytes_this_buffer = uint16(r2.DataLen())
	d3.n_bytes_this_buffer = uint16(r3.DataLen())

	f0, f1, f2, f3 := r0.NextValidFlag(), r1.NextValidFlag(), r2.NextValidFlag(), r3.NextValidFlag()

	d0.status0 = d.tx_desc_status0_by_next_valid_flag[f0]
	d1.status0 = d.tx_desc_status0_by_next_valid_flag[f1]
	d2.status0 = d.tx_desc_status0_by_next_valid_flag[f2]
	d3.status0 = d.tx_desc_status0_by_next_valid_flag[f3]

	d0.status1 = 0
	d1.status1 = 0
	d2.status1 = 0
	d3.status1 = 0
}

func (d *dev) set_tx_descriptors(rs []vnet.Ref, ds []tx_descriptor, ri0, di0, rn, dn reg) (ri, di, nd reg) {
	ri, di = ri0, di0
	for ri+4 <= rn && di+4 <= dn {
		d.set_4_tx_descriptors(rs, ds, ri, di)
		ri += 4
		di += 4
	}
	for ri < rn && di < dn {
		d.set_tx_descriptor(rs, ds, ri, di)
		ri += 1
		di += 1
	}
	nd = ri - ri0
	return
}

type tx_in struct {
	in   *vnet.RefVecIn
	free chan *vnet.RefVecIn
}

func (x *tx_in) Len() uint { return x.in.Refs.Len() }

type tx_dev struct {
	tx_queues                          tx_dma_queue_vec
	tx_desc_status0_by_next_valid_flag [vnet.NextValid + 1]uint16
}

func (q *tx_dma_queue) output() {
	d := q.d
	for {
		x := <-q.tx_fifo

		nr := reg(x.Len())

		head, tail := q.head_index, q.tail_index
		// Free slots are after tail and before head.
		n_free := head - tail
		if int32(n_free) <= 0 {
			n_free += q.len
		}

		// No room?
		if n_free < nr {
			panic("ga")
			x.free <- x.in
			continue
		}

		ds, rs := q.tx_descriptors, x.in.Refs

		ri, n_tx := reg(0), reg(0)

		// From tail (halt index) to end of ring.
		di := tail
		n_end := n_free
		if tail+n_end > q.len {
			n_end = q.len - tail
		}
		if n_end > 0 {
			var nd reg
			ri, di, nd = d.set_tx_descriptors(rs, ds, ri, di, nr, di+n_end)
			n_free -= nd
			n_tx += nd
		}

		// From start of ring to head.
		n_start := n_free
		if n_start > head {
			n_start = head
		}
		if n_start > 0 && ri < nr {
			var nd reg
			ri, di, nd = d.set_tx_descriptors(rs, ds, ri, 0, nr, n_start)
			n_free -= nd
			n_tx += nd
		}

		// Ring wrap.
		if di >= q.len {
			di = 0
		}

		if elog.Enabled() {
			elog.GenEventf("ixge tx %d new tail %d head %d tail %d", n_tx, di, head, tail)
		}

		// Re-start dma engine when tail advances.
		if di != q.tail_index {
			q.tail_index = di

			// Report status when done.
			i := di - 1
			if di == 0 {
				i = q.len - 1
			}
			ds[i].status0 |= tx_desc_status0_report_status

			hw.MemoryBarrier()

			dr := q.get_regs()
			dr.tail_index.set(d, di)
		}

		q.tx_irq_fifo <- x
	}
}

func (d *dev) InterfaceOutput(in *vnet.RefVecIn, free chan *vnet.RefVecIn) {
	q := &d.tx_queues[0]

	if q.tx_fifo == nil {
		q.tx_fifo = make(chan tx_in, 64)
		q.tx_irq_fifo = make(chan tx_in, 64)
		go q.output()
	}

	q.tx_fifo <- tx_in{in: in, free: free}
}

func (d *dev) tx_queue_interrupt(queue uint) {
	q := &d.tx_queues[0]

	// Mutually excludes real interrupt and polled calls from interfering with each other.
	q.mu.Lock()
	defer q.mu.Unlock()

	dr := q.get_regs()
	di := dr.head_index.get(d)
	n_advance := di - q.head_index
	if di <= q.head_index {
		n_advance += q.len
	}
	q.head_index = di
	if elog.Enabled() {
		tail := dr.tail_index.get(d)
		elog.GenEventf("ixge tx irq adv %d head %d tail %d", n_advance, di, tail)
	}

	for n_advance > 0 {
		if q.n_current_tx_in == 0 {
			q.current_tx_in = <-q.tx_irq_fifo
			q.n_current_tx_in = reg(q.current_tx_in.Len())
		}
		if n_advance < q.n_current_tx_in {
			q.n_current_tx_in -= n_advance
			break
		}
		n_advance -= q.n_current_tx_in
		q.n_current_tx_in = 0
		q.current_tx_in.free <- q.current_tx_in.in
	}
}
