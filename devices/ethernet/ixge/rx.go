package ixge

import (
	"github.com/platinasystems/elib/hw"
	"github.com/platinasystems/vnet"
	"github.com/platinasystems/vnet/ethernet"

	"unsafe"
)

type rx_dev struct {
	rx_queues              rx_dma_queue_vec
	rx_pool                hw.BufferPool
	rx_next_by_layer2_type [n_ethernet_type_filter]rx_next
}

// Only advanced descriptors are supported.
type rx_to_hw_descriptor struct {
	tail_buffer_address uint64
	head_buffer_address uint64
}

func (d *rx_from_hw_descriptor) to_hw() *rx_to_hw_descriptor {
	return (*rx_to_hw_descriptor)(unsafe.Pointer(d))
}

func (d *rx_from_hw_descriptor) refill(r *vnet.Ref) {
	t := d.to_hw()
	t.tail_buffer_address = uint64(r.DataPhys())
}

func (d *rx_from_hw_descriptor) rx_dma_flags() vnet.RxDmaDescriptorFlags {
	return vnet.RxDmaDescriptorFlags(d.status[0]) | vnet.RxDmaDescriptorFlags(d.status[2])<<32
}

const (
	// If packet is layer2 bits [2:0] contain ethernet_type_queue_filter match.
	rx_desc_is_layer2   = 1 << (4 + 11)
	rx_desc_layer2_type = 0x7 << 0

	// Valid if not layer2
	rx_desc_is_ip4     = 1 << (4 + 0)
	rx_desc_is_ip4_ext = 1 << (4 + 1)
	rx_desc_is_ip6     = 1 << (4 + 2)
	rx_desc_is_ip6_ext = 1 << (4 + 3)
	rx_desc_is_tcp     = 1 << (4 + 4)
	rx_desc_is_udp     = 1 << (4 + 5)

	log2_rx_desc_is_owned_by_software, rx_desc_is_owned_by_software = iota + 32, 1 << (iota + 32)
	log2_rx_desc_is_end_of_packet, rx_desc_is_end_of_packet

	rx_desc_is_flow_director_filter_match = 1 << (iota + 32 + 2)
	rx_desc_is_vlan
	rx_desc_is_udp_checksummed
	rx_desc_is_tcp_checksummed
	rx_desc_is_ip4_checksummed
	rx_desc_not_unicast
	_
	rx_desc_is_double_vlan
	rx_desc_is_udp_invalid_checksum
	// Extended errors
	rx_desc_is_ethernet_error       = 1 << (32 + 20 + 9)
	rx_desc_is_tcp_invalid_checksum = 1 << (32 + 20 + 10)
	rx_desc_is_ip4_invalid_checksum = 1 << (32 + 20 + 11)
)

// Rx writeback descriptor format.
type rx_from_hw_descriptor struct {
	status [3]uint32

	n_bytes_this_descriptor uint16
	vlan_tag                uint16
}

//go:generate gentemplate -d Package=ixge -id rx_from_hw_descriptor -d Type=rx_from_hw_descriptor -d VecType=rx_from_hw_descriptor_vec github.com/platinasystems/elib/hw/dma_mem.tmpl

type rx_next uint8

const (
	rx_next_error rx_next = iota
	rx_next_punt
	rx_next_ethernet_input
	rx_next_ip4_input_valid_checksum
	rx_next_ip6_input
)

type rx_error uint32

const (
	rx_error_none rx_error = iota
	rx_error_ip4_invalid_checksum
)

func (q *rx_dma_queue) GetRefState(f vnet.RxDmaDescriptorFlags) (s vnet.RxDmaRefState) {
	error, next, advance := rx_error_none, rx_next_ethernet_input, 0

	is_ip4 := f&rx_desc_is_ip4_checksummed != 0
	if is_ip4 {
		next = rx_next_ip4_input_valid_checksum
	}
	if f&rx_desc_is_ip4_invalid_checksum != 0 {
		error = rx_error_ip4_invalid_checksum
	}

	if f&rx_desc_is_ip6 != 0 {
		next = rx_next_ip6_input
	}
	if f&rx_desc_is_layer2 != 0 {
		next = q.d.rx_next_by_layer2_type[f&rx_desc_layer2_type]
	}

	if error != rx_error_none {
		next = rx_next_error
	}

	if next != rx_next_ethernet_input {
		advance = ethernet.HeaderBytes
		if f&rx_desc_is_vlan != 0 {
			advance += ethernet.VlanHeaderBytes
		}
		if f&rx_desc_is_double_vlan != 0 {
			advance += ethernet.VlanHeaderBytes
		}
	}

	s.Next = uint(next)
	s.Si = q.d.Si()
	s.SetError(&q.d.Node, uint(error))
	s.Advance = advance
	return
}

func (q *rx_dma_queue) rx_no_wrap(n_descriptors reg) (done bool) {
	d := q.d
	n_left := n_descriptors
	i := q.head_index

	ri := q.RingIndex(uint(i))
	for n_left >= 4 {
		d0, d1, d2, d3 := &q.rx_desc[i+0], &q.rx_desc[i+1], &q.rx_desc[i+2], &q.rx_desc[i+3]

		f0, f1, f2, f3 := d0.rx_dma_flags(), d1.rx_dma_flags(), d2.rx_dma_flags(), d3.rx_dma_flags()

		// Skip to single loop for any hardware owned descriptors found.
		if f0&f1&f2&f3&rx_desc_is_owned_by_software == 0 {
			break
		}

		b0, b1 := uint(d0.n_bytes_this_descriptor), uint(d1.n_bytes_this_descriptor)
		b2, b3 := uint(d2.n_bytes_this_descriptor), uint(d3.n_bytes_this_descriptor)

		d0.refill(q.RefillRef(ri.NextRingIndex(0)))
		d1.refill(q.RefillRef(ri.NextRingIndex(1)))
		d2.refill(q.RefillRef(ri.NextRingIndex(2)))
		d3.refill(q.RefillRef(ri.NextRingIndex(3)))

		ri = q.Rx4Descriptors(ri, b0, b1, b2, b3, f0, f1, f2, f3)

		n_left -= 4
		i += 4
	}

	for n_left > 0 {
		d0 := &q.rx_desc[i+0]
		f0 := d0.rx_dma_flags()

		if f0&rx_desc_is_owned_by_software == 0 {
			done = true
			break
		}

		b0 := uint(d0.n_bytes_this_descriptor)

		d0.refill(q.RefillRef(ri))

		ri = q.Rx1Descriptor(ri, b0, f0)

		n_left -= 1
		i += 1
	}

	if i >= reg(d.rx_ring_len) {
		i = 0
	}

	q.head_index = i
	return
}

func (d *dev) rx_queue_interrupt(queue uint) {
	q := &d.rx_queues[queue]
	dr := q.get_regs()

	// Fetch head from hardware and compare to where we think we are.
	hw_head_index := dr.head_index.get(d)
	sw_head_index := q.head_index
	if hw_head_index == sw_head_index {
		return
	}

	done := q.rx_no_wrap(reg(d.rx_ring_len) - sw_head_index)
	if !done && sw_head_index > 0 {
		q.RxDmaRing.WrapRefill()
		q.rx_no_wrap(hw_head_index - sw_head_index)
	}

	// Give tail back to hardware.
	hw.MemoryBarrier()
	q.tail_index = q.head_index - 1
	if q.head_index == 0 {
		q.tail_index = q.len - 1
	}
	dr.tail_index.set(d, q.tail_index)

	// Flush enqueue and counters.
	q.RxDmaRing.Flush()
}
