package ethernet

import (
	"github.com/platinasystems/vnet"
)

func GetHeader(r *vnet.Ref) *Header                 { return (*Header)(r.Data()) }
func GetPacketHeader(r *vnet.Ref) vnet.PacketHeader { return GetHeader(r) }
