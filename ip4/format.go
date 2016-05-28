package ip4

import (
	"github.com/platinasystems/elib/scan"

	"fmt"
)

func (a *Address) String() string                         { return fmt.Sprintf("%d.%d.%d.%d", a[0], a[1], a[2], a[3]) }
func (a *Address) ParseElt(s *scan.Scanner, i uint) error { return (*scan.Base10Uint8)(&a[i]).Parse(s) }
func (a *Address) Parse(s *scan.Scanner) error {
	cf := scan.ParseEltsConfig{
		Sep:     '.',
		MinElts: AddressBytes,
		MaxElts: AddressBytes,
	}
	return s.ParseElts(a, &cf)
}

func (p *Prefix) String() string { return fmt.Sprintf("%s/%d", &p.Address, p.Len) }

func (p *Prefix) Parse(s *scan.Scanner) (err error) {
	err = s.ParseFormat("%/%", &p.Address, (*scan.Base10Uint32)(&p.Len))
	if p.Len > 32 {
		err = fmt.Errorf("prefix length must be <= 32: %d", p.Len)
	}
	return
}

func (h *Header) String() (s string) {
	s = fmt.Sprintf("%s: %s -> %s", h.Protocol.String(), h.Src.String(), h.Dst.String())
	if h.Ip_version_and_header_length != 0x45 {
		s += fmt.Sprintf(", version: 0x%02x", h.Ip_version_and_header_length)
	}
	if got, want := h.Checksum, h.ComputeChecksum(); got != want {
		s += fmt.Sprintf(", checksum: 0x%04x (should be 0x%04x)", got.ToHost(), want.ToHost())
	}
	return
}

func (h *Header) Parse(s *scan.Scanner) (err error) {
	h.Ip_version_and_header_length = 0x45
	err = s.ParseFormat("%: % -> %", &h.Protocol, &h.Src, &h.Dst)
	return
}
