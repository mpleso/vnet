package ixge

import (
	"github.com/platinasystems/elib"
	"github.com/platinasystems/elib/hw"
	"github.com/platinasystems/vnet"

	"fmt"
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

	// Offset 0x30
	// [0] rx packet count
	// [1]/[2] rx byte count lo/hi
	// For VF, stats[1] is rx multicast packets.
	stats [3]reg

	_ reg
}

func (q *rx_dma_queue) get_regs() *rx_dma_regs {
	if q.index < 64 {
		return &q.d.regs.rx_dma0[q.index]
	} else {
		return &q.d.regs.rx_dma1[q.index-64]
	}
}

type tx_dma_regs struct {
	dma_regs

	// Offset 0x30
	_ [2]reg

	// [0] enables head write back.
	head_index_write_back_address [2]reg
}

// Only advanced descriptors are supported.
type rx_to_hw_descriptor struct {
	tail_buffer_address uint64
	head_buffer_address uint64
}

func (d *rx_from_hw_descriptor) to_hw() *rx_to_hw_descriptor {
	return (*rx_to_hw_descriptor)(unsafe.Pointer(d))
}

// Rx writeback descriptor format.
type rx_from_hw_descriptor struct {
	status [3]uint32

	n_bytes_this_descriptor uint16
	vlan_tag                uint16
}

type tx_descriptor struct {
	buffer_address      uint64
	n_bytes_this_buffer uint16
	status0             uint16
	status1             uint32
}

//go:generate gentemplate -d Package=ixge -id tx_descriptor -d Type=tx_descriptor -d VecType=tx_descriptor_vec github.com/platinasystems/elib/hw/dma_mem.tmpl
//go:generate gentemplate -d Package=ixge -id rx_from_hw_descriptor -d Type=rx_from_hw_descriptor -d VecType=rx_from_hw_descriptor_vec github.com/platinasystems/elib/hw/dma_mem.tmpl

type dma_queue struct {
	d *dev

	// Queue index.
	index uint

	// Software head/tail pointers into descriptor ring.
	head_index, tail_index reg
}

type rx_dma_queue struct {
	rxDmaRing vnet.RxDmaRing

	dma_queue

	rx_from_hw_descriptors rx_from_hw_descriptor_vec
	desc_id                elib.Index
}

type tx_dma_queue struct {
	dma_queue
	tx_descriptors        tx_descriptor_vec
	desc_id               elib.Index
	head_index_write_back *uint32
}

//go:generate gentemplate -d Package=ixge -id rx_dma_queue -d VecType=rx_dma_queue_vec -d Type=rx_dma_queue github.com/platinasystems/elib/vec.tmpl
//go:generate gentemplate -d Package=ixge -id tx_dma_queue -d VecType=tx_dma_queue_vec -d Type=tx_dma_queue github.com/platinasystems/elib/vec.tmpl

type dma_dev struct {
	dma_config
	rx_queues rx_dma_queue_vec
	rx_pool   hw.BufferPool
	tx_queues tx_dma_queue_vec
}

type dma_config struct {
	rx_ring_len     uint
	rx_buffer_bytes uint
	tx_ring_len     uint
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
		d.rx_pool.BufferTemplate = *hw.DefaultBufferTemplate
		d.rx_pool.BufferTemplate.Size = d.rx_buffer_bytes
		d.rx_pool.Name = fmt.Sprintf("ixge %s rx", d.pciDev)
		d.m.Vnet.AddBufferPool(&d.rx_pool)
	}

	if d.rx_ring_len == 0 {
		d.rx_ring_len = 2 * vnet.MaxVectorLen
	}
	q.rx_from_hw_descriptors, q.desc_id = rx_from_hw_descriptorAlloc(int(d.rx_ring_len))

	q.rxDmaRing.Init(&d.rx_pool, d.rx_ring_len)

	// Put even buffers on ring; odd buffers will be used for refill.
	{
		i := uint(0)
		ri := q.rxDmaRing.Index(i)
		for i < d.rx_ring_len {
			r, _ := q.rxDmaRing.Get(ri)
			d := q.rx_from_hw_descriptors[i].to_hw()
			d.tail_buffer_address = uint64(r.DataPhys())
			i++
			ri = ri.Next()
		}
	}

	{
		dr := q.get_regs()
		dr.descriptor_address.set(d, uint64(q.rx_from_hw_descriptors[0].PhysAddress()))
		n_desc := reg(len(q.rx_from_hw_descriptors))
		dr.n_descriptor_bytes.set(d, n_desc*reg(unsafe.Sizeof(q.rx_from_hw_descriptors[0])))

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
	}

	return
}

func (d *dma_dev) tx_dma_init(queue uint) {
	if d.tx_ring_len == 0 {
		d.tx_ring_len = 2 * vnet.MaxVectorLen
	}
	q := d.tx_queues.Validate(queue)
	q.tx_descriptors, q.desc_id = tx_descriptorAlloc(int(d.tx_ring_len))
	return
}
