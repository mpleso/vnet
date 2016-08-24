package ixge

import (
	"fmt"
	"github.com/platinasystems/elib/hw"
	"github.com/platinasystems/elib/hw/pci"
	"github.com/platinasystems/vnet"
	vnetpci "github.com/platinasystems/vnet/devices/bus/pci"
	"github.com/platinasystems/vnet/ethernet"
)

type main struct {
	vnet.Package
	devices []*device
}

type device struct {
	m           *main
	regs        *regs
	mmaped_regs []byte
	pciDev      *pci.Device
}

func (d *device) bar0() []byte { return d.pciDev.Resources[0].Mem }

func (m *main) DeviceMatch(pdev *pci.Device) (dev pci.DriverDevice, err error) {
	d := &device{m: m, pciDev: pdev}
	m.devices = append(m.devices, d)
	r := &pdev.Resources[0]
	if _, err = pdev.MapResource(r); err != nil {
		return
	}
	// Can't directly use mmapped registers because of compiler's read probes/nil checks.
	d.regs = (*regs)(hw.RegsBasePointer)
	d.mmaped_regs = d.bar0()
	dev = d
	return
}

func (d *device) Init() {
	r := d.regs

	// Reset chip.
	{
		const (
			mac_reset    = 1 << 3
			device_reset = 1 << 26
		)
		v := r.control.get(d)
		v |= mac_reset | device_reset
		r.control.set(d, v)

		// Timed to take ~1e-6 secs.  No need for timeout.
		for r.control.get(d)&device_reset != 0 {
		}
	}

	// Indicate software loaded.
	r.extended_control.or(d, 1<<28)

	// Fetch ethernet address from eeprom.
	{
		var v [2]reg
		var e ethernet.Address
		for i := range v {
			v[i] = r.rx_ethernet_address0[0][i].get(d)
		}
		for i := range e {
			e[i] = byte(v[i/4] >> ((uint(i) % 4) * 8))
		}
		fmt.Printf("%s\n", &e)
	}
}

func (d *device) Interrupt() {
	panic("ga")
}

func Init(v *vnet.Vnet) {
	m := &main{}
	devs := []pci.VendorDeviceID{
		0x15ab, // X552 backplane
	}
	err := pci.SetDriver(m, pci.Intel, devs)
	if err != nil {
		panic(err)
	}

	vnetpci.Init(v)
	v.AddPackage("ixge", m)
	m.Package.DependedOnBy("pci")
}
