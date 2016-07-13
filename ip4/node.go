package ip4

import (
	"github.com/platinasystems/vnet"
)

func GetHeader(r *vnet.Ref) *Header { return (*Header)(r.Data()) }

// Empty for now.
var rewriteNode vnet.Noder
var arpNode vnet.Noder
