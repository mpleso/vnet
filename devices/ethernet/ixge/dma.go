package ixge

import (
	"github.com/platinasystems/elib"
	"github.com/platinasystems/elib/hw"
	"github.com/platinasystems/vnet"

	"fmt"
	"sync"
	"unsafe"
)

type addr [2]reg

func (a *addr) set(d *dev, v uint64) {
	a[0].set(d, reg(v))
	a[1].set(d, reg(v>>32))
}

type dma_regs struct {
	// [31:7] 128 byte aligned.
	descriptor_address addr

	n_descriptor_bytes reg

	// [0] rx descriptor fetch tph enable
	// [1] rx descriptor write back tph enable
	// [2] rx header data tph enable
	// [3] rx payload data tph enable
	// [5] rx/tx descriptor dca enable
	// [6] rx packet head dca enable
	// [7] rx packet tail dca enable
	// [9] rx/tx descriptor relaxed order
	// [11] rx/tx descriptor write back relaxed order
	// [13] rx/tx data write/read relaxed order
	// [15] rx head data write relaxed order
	// [31:24] apic id for cpu's cache.
	dca_control reg

	head_index reg

	// [4:0] tail buffer size (in 1k byte units)
	// [13:8] head buffer size (in 64 byte units)
	// [24:22] lo free descriptors interrupt threshold (units of 64 descriptors)
	//         interrupt is generated each time number of free descriptors is decreased to X * 64
	// [27:25] descriptor type 0 = legacy, 1 = advanced one buffer (e.g. tail),
	//   2 = advanced header splitting (head + tail), 5 = advanced header splitting (head only).
	// [28] drop if no descriptors available.
	rx_split_control reg

	tail_index reg

	// [0] rx/tx packet count
	// [1]/[2] rx/tx byte count lo/hi
	vf_stats [3]reg

	// [7:0] rx/tx prefetch threshold
	// [15:8] rx/tx host threshold
	// [24:16] rx/tx write back threshold
	// [25] rx/tx enable
	// [26] tx descriptor writeback flush
	// [30] rx strip vlan enable
	control reg

	rx_coallesce_control reg
}

type rx_dma_regs struct {
	dma_regs

	// Offset 0x30.  Only defined for queues 0-15.
	// [0] rx packet count
	// [1]/[2] rx byte count lo/hi
	// For VF, stats[1] is rx multicast packets.
	stats [3]reg

	_ reg
}

type tx_dma_regs struct {
	dma_regs

	// Offset 0x30
	_ [2]reg

	// [0] enables head write back.
	head_index_write_back_address [2]reg
}

func (q *rx_dma_queue) get_regs() *rx_dma_regs {
	if q.index < 64 {
		return &q.d.regs.rx_dma0[q.index]
	} else {
		return &q.d.regs.rx_dma1[q.index-64]
	}
}

func (q *tx_dma_queue) get_regs() *tx_dma_regs {
	return &q.d.regs.tx_dma[q.index]
}

type dma_queue struct {
	d *dev

	mu sync.Mutex

	// Queue index.
	index uint

	// Software head/tail pointers into descriptor ring.
	len, head_index, tail_index reg
}

type rx_dma_queue struct {
	vnet.RxDmaRing

	dma_queue

	rx_desc rx_from_hw_descriptor_vec
	desc_id elib.Index
}

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

//go:generate gentemplate -d Package=ixge -id rx_dma_queue -d VecType=rx_dma_queue_vec -d Type=rx_dma_queue github.com/platinasystems/elib/vec.tmpl
//go:generate gentemplate -d Package=ixge -id tx_dma_queue -d VecType=tx_dma_queue_vec -d Type=tx_dma_queue github.com/platinasystems/elib/vec.tmpl

const n_ethernet_type_filter = 8

type dma_dev struct {
	dma_config
	rx_dev
	tx_dev
	queues_for_interrupt [vnet.NRxTx]elib.BitmapVec
}

type dma_config struct {
	rx_ring_len     uint
	rx_buffer_bytes uint
	tx_ring_len     uint
}

func (q *dma_queue) start(d *dev, dr *dma_regs) {
	v := dr.control.get(d)
	// prefetch threshold
	v = (v &^ (0x3f << 0)) | (32 << 0)
	// writeback theshold
	v = (v &^ (0x3f << 16)) | (16 << 16)
	// enable
	v |= 1 << 25
	dr.control.set(d, v)

	// wait for hardware to initialize.
	for dr.control.get(d)&(1<<25) == 0 {
	}

	// Set head/tail.
	dr.head_index.set(d, q.head_index)
	dr.tail_index.set(d, q.tail_index)
}

func (d *dev) init_rx_pool() {
	p := &d.rx_pool
	t := &p.BufferTemplate

	p.Name = fmt.Sprintf("ixge %s rx", d.pciDev)

	*t = *hw.DefaultBufferTemplate
	t.Size = d.rx_buffer_bytes

	// Set interface for rx buffers.
	ref := (*vnet.Ref)(unsafe.Pointer(&t.Ref))
	ref.Si = d.HwIf.Si()

	d.m.Vnet.AddBufferPool(p)
}

func (d *dev) rx_dma_init(queue uint) {
	q := d.rx_queues.Validate(queue)
	q.d = d
	q.index = queue

	// DMA buffer pool init.
	if len(d.rx_pool.Name) == 0 {
		if d.rx_buffer_bytes == 0 {
			d.rx_buffer_bytes = 1024
		}
		d.rx_buffer_bytes = uint(elib.Word(d.rx_buffer_bytes).RoundPow2(1024))
		d.init_rx_pool()
	}

	if d.rx_ring_len == 0 {
		d.rx_ring_len = 2 * vnet.MaxVectorLen
	}
	q.rx_desc, q.desc_id = rx_from_hw_descriptorAlloc(int(d.rx_ring_len))

	flags := vnet.RxDmaDescriptorFlags(rx_desc_is_ip4 | rx_desc_is_ip4_checksummed)
	q.RxDmaRingInit(d.m.Vnet, q, flags, &d.rx_pool, d.rx_ring_len)

	// Put even buffers on ring; odd buffers will be used for refill.
	{
		i := uint(0)
		ri := q.RingIndex(i)
		for i < d.rx_ring_len {
			r := q.RxDmaRing.RxRef(ri)
			q.rx_desc[i].refill(r)
			i++
			ri = ri.NextRingIndex(1)
		}
	}

	dr := q.get_regs()
	dr.descriptor_address.set(d, uint64(q.rx_desc[0].PhysAddress()))
	n_desc := reg(len(q.rx_desc))
	dr.n_descriptor_bytes.set(d, n_desc*reg(unsafe.Sizeof(q.rx_desc[0])))

	{
		v := reg(d.rx_buffer_bytes/24) << 0
		// Set lo free descriptor interrupt threshold to 1 * 64 descriptors.
		v |= 1 << 22
		// Descriptor type: advanced one buffer descriptors.
		v |= 1 << 25
		// Drop if out of descriptors.
		v |= 1 << 28
		dr.rx_split_control.set(d, v)
	}

	// Give hardware all but last cache line of descriptors.
	q.tail_index = n_desc - 4

	// enable [9] rx/tx descriptor relaxed order
	// enable [11] rx/tx descriptor write back relaxed order
	// enable [13] rx/tx data write/read relaxed order
	dr.dca_control.or(d, 1<<9|1<<11|1<<13)

	hw.MemoryBarrier()

	// Make sure rx is enabled.
	d.regs.rx_enable.or(d, 1<<0)

	q.start(d, &dr.dma_regs)
}

func (d *dev) tx_dma_init(queue uint) {
	if d.tx_ring_len == 0 {
		d.tx_ring_len = 2 * vnet.MaxVectorLen
	}
	q := d.tx_queues.Validate(queue)
	q.d = d
	q.index = queue
	q.tx_descriptors, q.desc_id = tx_descriptorAlloc(int(d.tx_ring_len))

	dr := q.get_regs()
	dr.descriptor_address.set(d, uint64(q.tx_descriptors[0].PhysAddress()))
	n_desc := reg(len(q.tx_descriptors))
	dr.n_descriptor_bytes.set(d, n_desc*reg(unsafe.Sizeof(q.tx_descriptors[0])))

	hw.MemoryBarrier()

	// Make sure tx is enabled.
	d.regs.tx_dma_control.or(d, 1<<0)

	q.start(d, &dr.dma_regs)
}
