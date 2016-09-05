package ixge

import (
	"github.com/platinasystems/elib/hw"
	"github.com/platinasystems/elib/hw/pci"
	"github.com/platinasystems/vnet"
	vnetpci "github.com/platinasystems/vnet/devices/bus/pci"
	"github.com/platinasystems/vnet/ethernet"

	"time"
)

type main struct {
	vnet.Package
	vnet.InterfaceCounterNames
	devs []dever
}

type phy struct {
	mdio_address reg

	// 32 bit ID read from ID registers.
	id uint32
}

type dev struct {
	m           *main
	d           dever
	regs        *regs
	mmaped_regs []byte
	pciDev      *pci.Device

	interruptsEnabled bool
	active_count      int32

	/* Phy index (0 or 1) and address on MDI bus. */
	phy_index uint

	phys [2]phy

	vnet_dev
	dma_dev
}

type dever interface {
	get() *dev
	get_semaphore()
	put_semaphore()
	phy_init()
}

func (d *dev) get() *dev    { return d }
func (d *dev) bar0() []byte { return d.pciDev.Resources[0].Mem }

var is_x540 = map[dev_id]bool{
	dev_id_x540t:          true,
	dev_id_x540t1:         true,
	dev_id_x550t:          true,
	dev_id_x550em_x_kx4:   true,
	dev_id_x550em_x_kr:    true,
	dev_id_x550em_x_sfp:   true,
	dev_id_x550em_x_10g_t: true,
	dev_id_x550em_x_1g_t:  true,
	dev_id_x550_vf_hv:     true,
	dev_id_x550_vf:        true,
	dev_id_x550em_x_vf:    true,
	dev_id_x550em_x_vf_hv: true,
}

func (m *main) DeviceMatch(pdev *pci.Device) (dd pci.DriverDevice, err error) {
	id := dev_id(pdev.DeviceID())
	var dr dever
	switch {
	case is_x540[id]:
		dr = &dev_x540{}
	default:
		dr = &dev_82599{}
	}
	d := dr.get()

	d.d = dr
	d.m = m
	d.pciDev = pdev
	m.devs = append(m.devs, dr)

	r := &pdev.Resources[0]
	if _, err = pdev.MapResource(r); err != nil {
		return
	}
	// Can't directly use mmapped registers because of compiler's read probes/nil checks.
	d.regs = (*regs)(hw.RegsBasePointer)
	d.mmaped_regs = d.bar0()
	return d, nil
}

// Write flush by reading status register.
func (d *dev) write_flush() { d.regs.status_read_only.get(d) }

func (d *dev) Init() {
	r := d.regs

	// Reset chip.
	{
		const (
			mac_reset = 1 << 3
			dev_reset = 1 << 26
		)
		v := r.control.get(d)
		v |= mac_reset | dev_reset
		r.control.set(d, v)

		// Timed to take ~1e-6 secs.  No need for timeout.
		for r.control.get(d)&dev_reset != 0 {
		}
	}

	// Indicate software loaded.
	r.extended_control.or(d, 1<<28)

	// Fetch ethernet address from eeprom.
	{
		var q [2]reg
		a := ethernet.Address{0xea, 0xeb, 0xec, 0xed, 0xee, 0xef}
		q[0] = reg(a[0]) | reg(a[1])<<8 | reg(a[2])<<16 | reg(a[3])<<24
		q[1] = 1<<31 | reg(a[4]) | reg(a[5])<<8
		r.rx_ethernet_address1[0][0].set(d, q[0])
		r.rx_ethernet_address1[0][1].set(d, q[1])
	}

	d.vnetInit()

	d.d.phy_init()

	d.clear_counters()
	d.rx_dma_init(0)
	d.tx_dma_init(0)
	d.set_queue_interrupt_mapping(vnet.Rx, 0, 0)
	d.set_queue_interrupt_mapping(vnet.Tx, 0, 1)

	// Accept all broadcast packets.
	// Multicasts must be explicitly added to dst_ethernet_address register array.
	d.regs.filter_control.set(d, 1<<10|0<<9)

	// Enable frames up to size in mac frame size register.
	// Set max frame size so we never drop frames.
	d.regs.xge_mac.control.or(d, 1<<2)
	d.regs.xge_mac.rx_max_frame_size.set(d, 0xffff<<16)

	if true {
		// Put mac in loopback.
		d.regs.xge_mac.control.or(d, 1<<15)
		// Force link up.
		d.regs.xge_mac.mac_control.or(d, 1<<0)
	}

	// Enable all interrupts.
	d.InterruptEnable(true)
}

const (
	sw_fw_eeprom   = 1 << 0
	sw_fw_phy_0    = 1 << 1
	sw_fw_phy_1    = 1 << 2
	sw_fw_mac_regs = 1 << 3
	sw_fw_flash    = 1 << 4
	sw_fw_i2c_0    = 1 << 11
	sw_fw_i2c_1    = 1 << 12
)

func (d *dev) software_firmware_sync(sw_mask_0_4, sw_mask_11_12 reg) {
	r := d.regs
	sw_mask := sw_mask_0_4 | sw_mask_11_12
	fw_mask := sw_mask_0_4<<5 | sw_mask_11_12<<2
	done := false
	for {
		d.d.get_semaphore()
		m := r.software_firmware_sync.get(d)
		if done = m&fw_mask == 0; done {
			r.software_firmware_sync.set(d, m|sw_mask)
		}
		d.d.put_semaphore()
		if done {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (d *dev) software_firmware_sync_release(sw_mask_0_4, sw_mask_11_12 reg) {
	sw_mask := sw_mask_0_4 | sw_mask_11_12
	d.regs.software_firmware_sync.andnot(d, sw_mask)
}

func Init(v *vnet.Vnet) {
	m := &main{}
	devs := []pci.VendorDeviceID{}
	for id, _ := range dev_id_names {
		devs = append(devs, pci.VendorDeviceID(id))
	}
	err := pci.SetDriver(m, pci.Intel, devs)
	if err != nil {
		panic(err)
	}

	vnetpci.Init(v)
	v.AddPackage("ixge", m)
	m.Package.DependedOnBy("pci-discovery")

	m.cliInit()
}
