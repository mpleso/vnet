package vnet

import "fmt"

type RxTx int

const (
	Rx RxTx = iota
	Tx
	NRxTx
)

// To clarify units for speed attribute: 1e9 * net.BitsPerSec
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

// Interface can loopback at MAC or PHY.
type IntfLoopbackType int

const (
	None IntfLoopbackType = iota + 1
	Mac
	Phy
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

type IfIndex uint32

// Hardware interface
type HwIntf struct {
	Index   IfIndex
	Name    string
	Speed   Bandwidth
	AdminUp bool
	// Hardware link state: up or down
	LinkUp bool

	// Max size of packet in bytes (MTU)
	MaxPacketSize int
}

// Per-interface counter.
type IntfCounter struct {
	Name        string
	Description string
	// Counter values: current, at last clear, difference from last update.
	current   uint64
	lastClear uint64
	lastDiff  uint64
}

// Package sort.Interface for sorting counters by name
type counterByName []IntfCounter

func (a counterByName) Len() int           { return len(a) }
func (a counterByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a counterByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

// Standard counter types (valid for all network interfaces).
type IntfCounterType int

const (
	RxPackets IntfCounterType = iota + 1
	RxBytes
	TxBytes
	TxPackets
)

//go:generate stringer -type=RxTx,IntfCounterType,IntfLoopbackType
