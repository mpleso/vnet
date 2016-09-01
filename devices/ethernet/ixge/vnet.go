package ixge

import (
	"github.com/platinasystems/vnet"
	"github.com/platinasystems/vnet/ethernet"
)

type vnet_dev struct {
	// Packet interfaces are output only.  Input is via rx node.
	vnet.InterfaceNode
	ethernet.Interface
	ethIfConfig ethernet.InterfaceConfig
}

func (d *dev) vnetInit() {
	ethernet.RegisterInterface(d.m.Vnet, d, &d.ethIfConfig, "ixge%s", d.pciDev.Addr.String())
}

func (d *dev) GetHwInterfaceCounters(n *vnet.InterfaceCounterNames, th *vnet.InterfaceThread) {
	panic("not yet")
}

func (d *dev) ValidateSpeed(speed vnet.Bandwidth) (err error) {
	return
}
