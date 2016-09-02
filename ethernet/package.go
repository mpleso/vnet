package ethernet

import (
	"github.com/platinasystems/vnet"
)

var packageIndex uint

type Main struct {
	vnet.Package
	ipNeighborMain
	nodeMain
}

func Init(v *vnet.Vnet) {
	m := &Main{}
	m.ipNeighborMain.init(v)
	m.nodeInit(v)
	packageIndex = v.AddPackage("ethernet", m)
}

func GetMain(v *vnet.Vnet) *Main { return v.GetPackage(packageIndex).(*Main) }
