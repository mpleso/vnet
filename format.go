package vnet

import (
	"github.com/platinasystems/elib/scan"

	"fmt"
)

func (v *Vnet) ParseHwIf(s *scan.Scanner) (hi Hi, err error) {
	var i uint
	if i, err = v.hwIfIndexByName.Parse(s); err == nil {
		hi = Hi(i)
	}
	return
}

func (v *Vnet) ParseSwIf(s *scan.Scanner) (si Si, err error) {
	var hi Hi
	if hi, err = v.ParseHwIf(s); err != nil {
		return
	}
	// Initially get software interface from hardware interface.
	hw := v.HwIf(hi)
	si = hw.si
	if s.AdvanceIf('.') {
		var id IfIndex
		if err = (*scan.Base10Uint32)(&id).Parse(s); err != nil {
			err = fmt.Errorf("bad id in interface NAME.ID: %s", err)
			return
		}

		var ok bool
		if si, ok = hw.subSiById[id]; !ok {
			err = fmt.Errorf("unkown sub interface id: %d", id)
		}
	}
	return
}
