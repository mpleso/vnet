package pg

import (
	"github.com/platinasystems/vnet"
)

var packageIndex uint

type main struct {
	vnet.Package
	node
}

func Init(v *vnet.Vnet) {
	m := &main{}
	m.node.init(v)
	m.cli_init()
	packageIndex = v.AddPackage("pg", m)
}

func GetMain(v *vnet.Vnet) *main { return v.GetPackage(packageIndex).(*main) }
