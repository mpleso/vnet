package ixge

import (
	"github.com/platinasystems/vnet"
)

type tx_descriptor struct {
	buffer_address      uint64
	n_bytes_this_buffer uint16
	status0             uint16
	status1             uint32
}

//go:generate gentemplate -d Package=ixge -id tx_descriptor -d Type=tx_descriptor -d VecType=tx_descriptor_vec github.com/platinasystems/elib/hw/dma_mem.tmpl

func (d *dev) InterfaceOutput(in *vnet.RefVecIn, free chan *vnet.RefVecIn) {
	panic("not yet")
}

func (d *dev) tx_queue_interrupt(queue uint) {
}
