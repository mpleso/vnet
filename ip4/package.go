package ip4

import (
	"github.com/platinasystems/vnet"
)

var packageIndex uint

func Init(v *vnet.Vnet) {
	m := &Main{}
	packageIndex = v.AddPackage("ip4", m)
}

func GetMain(v *vnet.Vnet) *Main { return v.GetPackage(packageIndex).(*Main) }

func (m *Main) Init() (err error) {
	m.Vnet.RegisterSwIfAdminUpDownHook(m.swIfAdminUpDown)
	m.Ip.Init(m.Vnet)
	return
}
