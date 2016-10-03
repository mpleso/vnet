package ip4

import (
	"github.com/platinasystems/vnet"
	"github.com/platinasystems/vnet/ip"
)

var packageIndex uint

func Init(v *vnet.Vnet) {
	m := &Main{}
	packageIndex = v.AddPackage("ip4", m)
	m.DependsOn("pg")
}

func GetMain(v *vnet.Vnet) *Main { return v.GetPackage(packageIndex).(*Main) }

func ipAddressStringer(a *ip.Address) string { return IpAddress(a).String() }

type Main struct {
	vnet.Package
	ip.Main
	fibMain
	ifAddrAddDelHooks IfAddrAddDelHookVec
	nodeMain
	pgMain
}

func (m *Main) Init() (err error) {
	v := m.Vnet
	v.RegisterSwIfAdminUpDownHook(m.swIfAdminUpDown)
	cf := ip.FamilyConfig{
		Family:          ip.Ip4,
		AddressStringer: ipAddressStringer,
		RewriteNode:     &m.rewriteNode,
		PacketType:      vnet.IP4,
		GetRoute:        m.getRoute,
		AddDelRoute:     m.addDelRoute,
	}
	m.Main.Init(v, cf)
	m.nodeInit(v)
	m.pgInit(v)
	m.cliInit(v)
	return
}
