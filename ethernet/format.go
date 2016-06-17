package ethernet

import (
	"github.com/platinasystems/elib/scan"

	"fmt"
)

func (a *Address) String() string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", a[0], a[1], a[2], a[3], a[4], a[5])
}

func (a *Address) ParseElt(s *scan.Scanner, i uint) error { return (*scan.Base16Uint8)(&a[i]).Parse(s) }
func (a *Address) Parse(s *scan.Scanner) (err error) {
	cf := scan.ParseEltsConfig{
		Sep:     ':',
		MinElts: AddressBytes,
		MaxElts: AddressBytes,
	}
	return s.ParseElts(a, &cf)
}

func (h *Header) String() (s string) {
	return fmt.Sprintf("%s: %s -> %s", h.GetType().String(), h.Src.String(), h.Dst.String())
}

func (h *Header) Parse(s *scan.Scanner) (err error) {
	err = s.ParseFormat("%: % -> %", &h.Type, &h.Src, &h.Dst)
	return
}

func (h *VlanHeader) String() (s string) {
	return fmt.Sprintf("%s: vlan %d", h.GetType().String(), h.Priority_cfi_and_id.ToHost()&0xfff)
}
