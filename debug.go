//+build debug

package vnet

import (
	"github.com/platinasystems/elib/hw"

	"fmt"
	"unsafe"
)

func init() {
	if got, want := unsafe.Sizeof(Ref{}), unsafe.Sizeof(hw.Ref{}); got != want {
		panic(fmt.Errorf("ref size %d %d", got, want))
	}
}