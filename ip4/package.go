package ip4

import (
	"github.com/platinasystems/elib/cli"
	"github.com/platinasystems/vnet"
	"github.com/platinasystems/vnet/ip"
)

var packageIndex uint

func Init(v *vnet.Vnet) {
	m := &Main{}
	packageIndex = v.AddPackage("ip4", m)
}

func GetMain(v *vnet.Vnet) *Main { return v.GetPackage(packageIndex).(*Main) }

func ipAddressStringer(a *ip.Address) string { return IpAddress(a).String() }

func (m *Main) Init() (err error) {
	m.Vnet.RegisterSwIfAdminUpDownHook(m.swIfAdminUpDown)
	cf := ip.FamilyConfig{
		Family:          ip.Ip4,
		AddressStringer: ipAddressStringer,
		RewriteNode:     rewriteNode,
		PacketType:      vnet.IP4,
		GetRoute:        m.getRoute,
		AddDelRoute:     m.addDelRoute,
	}
	m.Main.Init(m.Vnet, cf)

	cmds := [...]cli.Command{
		cli.Command{
			Name:      "show ip fib",
			ShortHelp: "show ip4 forwarding table",
			Action:    m.showIpFib,
		},
	}
	for i := range cmds {
		m.Vnet.CliAdd(&cmds[i])
	}

	return
}
