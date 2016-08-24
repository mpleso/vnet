package ixge

import (
	"fmt"
	"github.com/platinasystems/elib/hw"
	"github.com/platinasystems/elib/hw/pci"
	"github.com/platinasystems/vnet"
	vnetpci "github.com/platinasystems/vnet/devices/bus/pci"
	"github.com/platinasystems/vnet/devices/phy/xge"
	"github.com/platinasystems/vnet/ethernet"
	"time"
)

type main struct {
	vnet.Package
	devs []*dev
}

type phy struct {
	mdio_address reg

	// 32 bit ID read from ID registers.
	id uint32
}

type dev struct {
	m           *main
	regs        *regs
	mmaped_regs []byte
	pciDev      *pci.Device

	/* Phy index (0 or 1) and address on MDI bus. */
	phy_index uint

	phys [2]phy
}

func (d *dev) bar0() []byte { return d.pciDev.Resources[0].Mem }

func (m *main) DeviceMatch(pdev *pci.Device) (dd pci.DriverDevice, err error) {
	d := &dev{m: m, pciDev: pdev}
	m.devs = append(m.devs, d)
	r := &pdev.Resources[0]
	if _, err = pdev.MapResource(r); err != nil {
		return
	}
	// Can't directly use mmapped registers because of compiler's read probes/nil checks.
	d.regs = (*regs)(hw.RegsBasePointer)
	d.mmaped_regs = d.bar0()
	return d, err
}

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
		var v [2]reg
		var e ethernet.Address
		for i := range v {
			v[i] = r.rx_ethernet_address0[0][i].get(d)
		}
		for i := range e {
			e[i] = byte(v[i/4] >> ((uint(i) % 4) * 8))
		}
		fmt.Printf("%s\n", d.get_dev_id())
		fmt.Printf("%s\n", &e)
	}

	if ok := d.probe_phy(); ok {
		fmt.Printf("found phy id %x\n", d.phys[d.phy_index].id)
	}
}

func (d *dev) get_semaphore() {
	r := d.regs
	start := time.Now()
	for r.software_semaphore.get(d)&(1<<0) == 0 {
		if time.Since(start) > 100*time.Millisecond {
			panic("ixge: semaphore get timeout")
		}
		time.Sleep(100 * time.Microsecond)
	}
	for {
		r.software_semaphore.or(d, 1<<1)
		if r.software_semaphore.get(d)&(1<<1) != 0 {
			break
		}
		if time.Since(start) > 100*time.Millisecond {
			panic("ixge: semaphore get timeout")
		}
	}
}

func (d *dev) release_semaphore() { d.regs.software_semaphore.andnot(d, 3) }

func (d *dev) software_firmware_sync(sw_mask reg) {
	r := d.regs
	fw_mask := sw_mask << 5
	done := false
	for {
		d.get_semaphore()
		m := r.software_firmware_sync.get(d)
		if done = m&fw_mask == 0; done {
			r.software_firmware_sync.set(d, m|sw_mask)
		}
		d.release_semaphore()
		if !done {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (d *dev) software_firmware_sync_release(sw_mask reg) {
	d.get_semaphore()
	d.regs.software_firmware_sync.andnot(d, sw_mask)
	d.release_semaphore()
}

func (d *dev) rw_phy_reg(dev_type, reg_index, v reg, is_read bool) (w reg) {
	const busy_bit = 1 << 30
	sync_mask := reg(1) << (1 + d.phy_index)
	d.software_firmware_sync(sync_mask)
	if !is_read {
		d.regs.xge_mac.phy_data.set(d, v)
	}
	// Address cycle.
	x := reg_index | dev_type<<16 | d.phys[d.phy_index].mdio_address<<21
	d.regs.xge_mac.phy_command.set(d, x|busy_bit)
	for d.regs.xge_mac.phy_command.get(d)&busy_bit != 0 {
	}
	cmd := reg(1)
	if is_read {
		cmd = 2
	}
	d.regs.xge_mac.phy_command.set(d, x|busy_bit|cmd<<26)
	for d.regs.xge_mac.phy_command.get(d)&busy_bit != 0 {
	}
	if is_read {
		w = d.regs.xge_mac.phy_data.get(d)
	} else {
		w = v
	}
	d.software_firmware_sync_release(sync_mask)
	return
}

func (d *dev) read_phy_reg(dev_type, reg_index reg) reg {
	return d.rw_phy_reg(dev_type, reg_index, 0, true)
}
func (d *dev) write_phy_reg(dev_type, reg_index, v reg) {
	d.rw_phy_reg(dev_type, reg_index, v, false)
}

func (d *dev) probe_phy() (ok bool) {
	phy := &d.phys[d.phy_index]

	phy.mdio_address = ^phy.mdio_address // poison
	for i := reg(0); i < 32; i++ {
		phy.mdio_address = i
		v := d.read_phy_reg(xge.PHY_DEV_TYPE_PMA_PMD, xge.PHY_ID1)
		if ok = v != 0xffff && v != 0; ok {
			phy.id = uint32(v)
			break
		}
	}
	return
}

// PCI dev IDs
const (
	dev_id_82598                 = 0x10b6
	dev_id_82598_bx              = 0x1508
	dev_id_82598af_dual_port     = 0x10c6
	dev_id_82598af_single_port   = 0x10c7
	dev_id_82598eb_sfp_lom       = 0x10db
	dev_id_82598at               = 0x10c8
	dev_id_82598at2              = 0x150b
	dev_id_82598eb_cx4           = 0x10dd
	dev_id_82598_cx4_dual_port   = 0x10ec
	dev_id_82598_da_dual_port    = 0x10f1
	dev_id_82598_sr_dual_port_em = 0x10e1
	dev_id_82598eb_xf_lr         = 0x10f4
	dev_id_82599_kx4             = 0x10f7
	dev_id_82599_kx4_mezz        = 0x1514
	dev_id_82599_kr              = 0x1517
	dev_id_82599_t3_lom          = 0x151c
	dev_id_82599_cx4             = 0x10f9
	dev_id_82599_sfp             = 0x10fb
	sub_dev_id_82599_sfp         = 0x11a9
	sub_dev_id_82599_sfp_wol0    = 0x1071
	sub_dev_id_82599_rndc        = 0x1f72
	sub_dev_id_82599_560flr      = 0x17d0
	sub_dev_id_82599_sp_560flr   = 0x211b
	sub_dev_id_82599_ecna_dp     = 0x0470
	sub_dev_id_82599_lom_sfp     = 0x8976
	dev_id_82599_backplane_fcoe  = 0x152a
	dev_id_82599_sfp_fcoe        = 0x1529
	dev_id_82599_sfp_em          = 0x1507
	dev_id_82599_sfp_sf2         = 0x154d
	dev_id_82599en_sfp           = 0x1557
	sub_dev_id_82599en_sfp_ocp1  = 0x0001
	dev_id_82599_xaui_lom        = 0x10fc
	dev_id_82599_combo_backplane = 0x10f8
	sub_dev_id_82599_kx4_kr_mezz = 0x000c
	dev_id_82599_ls              = 0x154f
	dev_id_x540t                 = 0x1528
	dev_id_82599_sfp_sf_qp       = 0x154a
	dev_id_82599_qsfp_sf_qp      = 0x1558
	dev_id_x540t1                = 0x1560
	dev_id_x550t                 = 0x1563
	dev_id_x550em_x_kx4          = 0x15aa
	dev_id_x550em_x_kr           = 0x15ab
	dev_id_x550em_x_sfp          = 0x15ac
	dev_id_x550em_x_10g_t        = 0x15ad
	dev_id_x550em_x_1g_t         = 0x15ae
	dev_id_x550_vf_hv            = 0x1564
	dev_id_x550_vf               = 0x1565
	dev_id_x550em_x_vf           = 0x15a8
	dev_id_x550em_x_vf_hv        = 0x15a9
)

type dev_id pci.VendorDeviceID

func (d *dev) get_dev_id() dev_id { return dev_id(d.pciDev.DeviceID()) }
func (d dev_id) String() (v string) {
	var ok bool
	if v, ok = dev_id_names[d]; !ok {
		v = fmt.Sprintf("unknown %04x", d)
	}
	return
}

var dev_id_names = map[dev_id]string{
	dev_id_82598:                 "82598",
	dev_id_82598_bx:              "82598_BX",
	dev_id_82598af_dual_port:     "82598AF_DUAL_PORT",
	dev_id_82598af_single_port:   "82598AF_SINGLE_PORT",
	dev_id_82598eb_sfp_lom:       "82598EB_SFP_LOM",
	dev_id_82598at:               "82598AT",
	dev_id_82598at2:              "82598AT2",
	dev_id_82598eb_cx4:           "82598EB_CX4",
	dev_id_82598_cx4_dual_port:   "82598_CX4_DUAL_PORT",
	dev_id_82598_da_dual_port:    "82598_DA_DUAL_PORT",
	dev_id_82598_sr_dual_port_em: "82598_SR_DUAL_PORT_EM",
	dev_id_82598eb_xf_lr:         "82598EB_XF_LR",
	dev_id_82599_kx4:             "82599_KX4",
	dev_id_82599_kx4_mezz:        "82599_KX4_MEZZ",
	dev_id_82599_kr:              "82599_KR",
	dev_id_82599_t3_lom:          "82599_T3_LOM",
	dev_id_82599_cx4:             "82599_CX4",
	dev_id_82599_sfp:             "82599_SFP",
	dev_id_82599_backplane_fcoe:  "82599_BACKPLANE_FCOE",
	dev_id_82599_sfp_fcoe:        "82599_SFP_FCOE",
	dev_id_82599_sfp_em:          "82599_SFP_EM",
	dev_id_82599_sfp_sf2:         "82599_SFP_SF2",
	dev_id_82599en_sfp:           "82599EN_SFP",
	dev_id_82599_xaui_lom:        "82599_XAUI_LOM",
	dev_id_82599_combo_backplane: "82599_COMBO_BACKPLANE",
	dev_id_82599_ls:              "82599_LS",
	dev_id_x540t:                 "X540T",
	dev_id_82599_sfp_sf_qp:       "82599_SFP_SF_QP",
	dev_id_82599_qsfp_sf_qp:      "82599_QSFP_SF_QP",
	dev_id_x540t1:                "X540T1",
	dev_id_x550t:                 "X550T",
	dev_id_x550em_x_kx4:          "X550EM_X_KX4",
	dev_id_x550em_x_kr:           "X550EM_X_KR",
	dev_id_x550em_x_sfp:          "X550EM_X_SFP",
	dev_id_x550em_x_10g_t:        "X550EM_X_10G_T",
	dev_id_x550em_x_1g_t:         "X550EM_X_1G_T",
	dev_id_x550_vf_hv:            "X550_VF_HV",
	dev_id_x550_vf:               "X550_VF",
	dev_id_x550em_x_vf:           "X550EM_X_VF",
	dev_id_x550em_x_vf_hv:        "X550EM_X_VF_HV",
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
}

func (d *dev) Interrupt() {
	panic("ga")
}
