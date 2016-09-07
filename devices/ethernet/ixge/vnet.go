package ixge

import (
	"github.com/platinasystems/vnet"
	"github.com/platinasystems/vnet/ethernet"
)

type vnet_dev struct {
	vnet.InterfaceNode
	ethernet.Interface
	ethIfConfig ethernet.InterfaceConfig
}

func (d *dev) vnetInit() {
	v := d.m.Vnet

	d.Next = []string{
		rx_next_error:                    "error",
		rx_next_punt:                     "punt",
		rx_next_ethernet_input:           "ethernet-input",
		rx_next_ip4_input_valid_checksum: "ip4-input-valid-checksum",
		rx_next_ip6_input:                "ip6-input",
	}
	d.Errors = []string{
		rx_error_none:                 "no error",
		rx_error_ip4_invalid_checksum: "invalid ip4 checksum",
	}

	ethernet.RegisterInterface(v, d, &d.ethIfConfig, "ixge%d", d.pciDev.Addr.Fn)
	v.RegisterInterfaceNode(d, d.Hi(), d.Name())
}

func (d *dev) ValidateSpeed(speed vnet.Bandwidth) (err error) {
	return
}
