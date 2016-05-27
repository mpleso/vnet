package ip4

import (
	"github.com/platinasystems/elib/loop"
	"github.com/platinasystems/vnet"
)

func GetHeader(r *loop.Ref) *Header { return (*Header)(r.Data()) }

// Empty for now.
var rewriteNode vnet.Noder
var arpNode vnet.Noder
