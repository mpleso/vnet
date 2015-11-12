// Generic devices on PCI bus.
package pci

import (
	"fmt"
	"unsafe"
)

// Under PCI, each device has 256 bytes of configuration address space,
// of which the first 64 bytes are standardized as follows:
type ConfigHeader struct {
	DeviceID
	Command
	Status

	Revision uint8

	// Distinguishes programming interface for device.
	// For example, different standards for USB controllers.
	SoftwareInterface

	DeviceClass

	CacheSize    uint8
	LatencyTimer uint8

	// If bit 7 of this register is set, the device has multiple functions;
	// otherwise, it is a single function device.
	Tp uint8

	Bist uint8
}

type HeaderType uint8

func (c ConfigHeader) Type() HeaderType {
	return HeaderType(c.Tp &^ (1 << 7))
}

const (
	Normal HeaderType = iota
	Bridge
	CardBus
)

type SoftwareInterface uint8

func (x SoftwareInterface) String() string {
	return fmt.Sprintf("0x%02x", uint8(x))
}

type Command uint16

const (
	IOEnable Command = 1 << iota
	MemoryEnable
	BusMasterEnable
	SpecialCycles
	WriteInvalidate
	VgaPaletteSnoop
	Parity
	AddressDataStepping
	SERR
	BackToBackWrite
	INTxEmulationDisable
)

type Status uint16

// Device/vendor ID from PCI config space.
type VendorID uint16
type VendorDeviceID uint16

func (d VendorDeviceID) String() string {
	return fmt.Sprintf("0x%04x", uint16(d))
}

// Vendor/Device pair
type DeviceID struct {
	Vendor VendorID
	Device VendorDeviceID
}

type BaseAddress uint32

func (b BaseAddress) IsMem() bool {
	return b&(1<<0) == 0
}

func (b BaseAddress) Addr() uint32 {
	return uint32(b &^ 0xf)
}

func (b BaseAddress) Valid() bool {
	return uint32(b) != 0
}

func (b BaseAddress) String() string {
	if b == 0 {
		return "{}"
	}
	x := uint32(b)
	tp := "mem"
	loc := ""
	if !b.IsMem() {
		tp = "i/o"
	} else {
		switch (x >> 1) & 3 {
		case 0:
			loc = "32-bit "
		case 1:
			loc = "< 1M "
		case 2:
			loc = "64-bit "
		case 3:
			loc = "unknown "
		}
		if x&(1<<3) != 0 {
			loc += "prefetchable "
		}
	}
	return fmt.Sprintf("{%s: %s0x%08x}", tp, loc, b.Addr())
}

/* Header type 0 (normal devices) */
type DeviceConfig struct {
	Hdr ConfigHeader

	// Base addresses specify locations in memory or I/O space.
	// Decoded size can be determined by writing a value of 0xffffffff to the register, and reading it back.
	// Only 1 bits are decoded.
	BaseAddress [6]BaseAddress

	CardBusCIS uint32

	SubID DeviceID

	RomAddress uint32

	// Config space offset of start of capability list.
	CapabilityOffset uint8
	_                [7]byte

	InterruptLine uint8
	InterruptPin  uint8
	MinGrant      uint8
	MaxLatency    uint8
}

type Capability uint8

const (
	PowerManagement Capability = iota + 1
	AGP
	VitalProductData
	SlotIdentification
	MSI
	CompactPCIHotSwap
	PCIX
	HyperTransport
	VendorSpecific
	DebugPort
	CompactPciCentralControl
	PCIHotPlugController
	SSVID
	AGP3
	SecureDevice
	PCIE
	MSIX
	SATA
	AdvancedFeatures
)

// Common header for capabilities.
type CapabilityHeader struct {
	Capability

	// Pointer to next capability header
	NextCapabilityHeader uint8
}

type BusAddress struct {
	Domain        uint16
	Bus, Slot, Fn uint8
}

func (a BusAddress) String() string {
	return fmt.Sprintf("%04x:%02x:%02x.%01x", a.Domain, a.Bus, a.Slot, a.Fn)
}

type Resource struct {
	Index      uint32 // index of BAR
	BAR        BaseAddress
	Base, Size uint64
	Mem        []byte
}

func (r Resource) String() string {
	return fmt.Sprintf("{%d: 0x%x-0x%x}", r.Index, r.Base, r.Base+r.Size-1)
}

type Device struct {
	Addr        BusAddress
	Config      DeviceConfig
	configBytes []byte
	Resources   []Resource
}

type Hardware interface {
	DeviceMatch(d *Device) (err error)
}

var registeredDevs map[DeviceID]Hardware = make(map[DeviceID]Hardware)

func Register(h Hardware, devs ...DeviceID) (err error) {
	for _, d := range devs {
		if _, exists := registeredDevs[d]; exists {
			return fmt.Errorf("duplicate registration: %v", d)
		}
		registeredDevs[d] = h
	}
	return
}

func (d *Device) ForeachCap(f func(h *CapabilityHeader, contents []byte) (done bool, err error)) (err error) {
	o := int(d.Config.CapabilityOffset)
	if o >= len(d.configBytes) {
		return
	}
	done := false
	for o < len(d.configBytes) {
		var h CapabilityHeader
		h.Capability = Capability(d.configBytes[o+0])
		h.NextCapabilityHeader = uint8(d.configBytes[o+1])
		b := d.configBytes[o+0:] // include CapabilityHeader
		done, err = f(&h, b)
		if err != nil || done {
			return
		}
		o = int(h.NextCapabilityHeader)
		if o < 0x40 || o == 0xff {
			break
		}
	}
	return
}

func (d *Device) FindCap(c Capability) (b []byte, found bool) {
	d.ForeachCap(func(h *CapabilityHeader, contents []byte) (done bool, err error) {
		found = h.Capability == c
		if found {
			b = contents
			done = true
		}
		return
	})
	return
}

func init() {
	if unsafe.Sizeof(DeviceConfig{}) != 64 {
		panic("device config size != 64")
	}
}

//go:generate stringer -type=Capability,HeaderType
