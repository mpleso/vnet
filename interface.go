package vnet

import (
	"fmt"
)

// Interface can loopback at MAC or PHY.
type IntfLoopbackType int

const (
	None IntfLoopbackType = iota
	Mac
	Phy
)

// To clarify units: 1e9 * vnet.Bps
const (
	Bps    = 1
	Kbps   = 1e3
	Mbps   = 1e6
	Gbps   = 1e9
	Tbps   = 1e12
	Bytes  = 1
	Kbytes = 1 << 10
	Mbytes = 1 << 20
	Gbytes = 1 << 30
)

type Bandwidth float64

func (b Bandwidth) String() string {
	unit := Bandwidth(1)
	prefix := ""
	switch {
	case b < Kbps:
		break
	case b <= Mbps:
		unit = Kbps
		prefix = "K"
	case b <= Gbps:
		unit = Mbps
		prefix = "M"
	case b <= Tbps:
		unit = Gbps
		prefix = "G"
	default:
		unit = Tbps
		prefix = "T"
	}
	b /= unit
	return fmt.Sprintf("%g%sbps", b, prefix)
}

type HwIf struct {
	Name string

	Speed Bandwidth

	// Admin state: up or down
	AdminUp bool

	// Hardware link state: up or down
	LinkUp bool

	// Hardware is unprovisioned.
	// Interfaces with 4 SERDES lanes will be represented as 4 interfaces.
	// Lanes may all be a single interface (1 provisioned 4 lane interface +
	// 3 unprovisioned 0 lane interfaces).
	Unprovisioned bool

	// Max size of packet in bytes (MTU)
	MaxPacketSize int
}
