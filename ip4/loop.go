package ip4

import (
	"github.com/platinasystems/elib/loop"
)

func GetHeader(r *loop.Ref) *Header { return (*Header)(r.Data()) }
