package ip

import (
	"github.com/platinasystems/vnet"
)

type Ip struct {
	fibMain
	adjacencyMain
	ifAddressMain
}

func (m *Ip) Init(v *vnet.Vnet) {
	m.adjacencyMain.init()
	m.ifAddressMain.init(v)
}
