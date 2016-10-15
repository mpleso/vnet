package unix

import (
	"github.com/platinasystems/vnet"
)

func Init(v *vnet.Vnet) {
	m := &Main{}
	m.v = v
	m.tuntapMain.Init(v)
	m.netlinkMain.Init(m)
	v.AddPackage("tuntap", m)
}
