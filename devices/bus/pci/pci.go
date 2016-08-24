package pci

import (
	"github.com/platinasystems/elib/hw/pci"
	"github.com/platinasystems/vnet"
)

type pciDiscover struct{ vnet.Package }

func (d *pciDiscover) Init() error { return pci.DiscoverDevices() }
func Init(v *vnet.Vnet) {
	if _, ok := v.PackageByName("pci"); !ok {
		v.AddPackage("pci", &pciDiscover{})
	}
}
