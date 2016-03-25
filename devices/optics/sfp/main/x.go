package main

import (
	"github.com/platinasystems/vnet/devices/optics/sfp"

	"fmt"
)

func main() {
	m := &sfp.QsfpModule{
		BusIndex:   10,
		BusAddress: 20,
	}
	m.Present()
	fmt.Printf("%+v\n", m)
}
