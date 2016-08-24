//+build debug

package ixge

import (
	"github.com/platinasystems/elib/hw"

	"unsafe"
)

func check(tag string, p unsafe.Pointer, expect uint) {
	hw.CheckRegAddr(tag, uint(uintptr(p)-hw.RegsBaseAddress), expect)
}

// Validate memory map.
func init() {
	r := (*regs)(hw.RegsBasePointer)
	check("pf_0", unsafe.Pointer(&r.pf_0), 0x700)
	check("interrupt", unsafe.Pointer(&r.interrupt), 0x800)
	check("rx_dma0", unsafe.Pointer(&r.rx_dma0[0]), 0x1000)
	check("rx_dma_control", unsafe.Pointer(&r.rx_dma_control), 0x2f00)
	check("rx_enable", unsafe.Pointer(&r.rx_enable), 0x3000)
	check("rx_dma1", unsafe.Pointer(&r.rx_dma1[0]), 0xd000)
	check("ethernet_type_queue_select", unsafe.Pointer(&r.ethernet_type_queue_select[0]), 0xec00)
	check("fcoe_redirection", unsafe.Pointer(&r.fcoe_redirection), 0xed00)
	check("flow_director", unsafe.Pointer(&r.flow_director), 0xee00)
	check("pf_1", unsafe.Pointer(&r.pf_1), 0xf000)
	check("eeprom_flash_control", unsafe.Pointer(&r.eeprom_flash_control), 0x10010)
	check("pcie", unsafe.Pointer(&r.pcie), 0x11000)
	check("sfp_i2c", unsafe.Pointer(&r.sfp_i2c), 0x15f58)
}
