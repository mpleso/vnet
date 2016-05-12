package ethernet

import (
	"github.com/platinasystems/elib/loop"
	"github.com/platinasystems/vnet"
)

func GetHeader(r *loop.Ref) *Header                 { return (*Header)(r.Data()) }
func GetPacketHeader(r *loop.Ref) vnet.PacketHeader { return GetHeader(r) }
