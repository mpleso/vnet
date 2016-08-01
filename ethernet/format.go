package ethernet

import (
	"github.com/platinasystems/elib/parse"

	"fmt"
)

func (a *Address) String() string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", a[0], a[1], a[2], a[3], a[4], a[5])
}

func (a *Address) Parse(in *parse.Input) {
	in.Parse("%x:%x:%x:%x:%x:%x", &a[0], &a[1], &a[2], &a[3], &a[4], &a[5])
}

func (h *Header) String() (s string) {
	return fmt.Sprintf("%s: %s -> %s", h.GetType().String(), h.Src.String(), h.Dst.String())
}

func (h *Header) Parse(in *parse.Input) { in.Parse("%v: %v -> %v", &h.Type, &h.Src, &h.Dst) }

func (h *VlanHeader) String() (s string) {
	return fmt.Sprintf("%s: vlan %d", h.GetType().String(), h.Priority_cfi_and_id.ToHost()&0xfff)
}
