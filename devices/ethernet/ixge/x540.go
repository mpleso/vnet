package ixge

type dev_x540 struct {
	dev
}

func (d *dev_x540) get_put_semaphore(is_put bool) (x reg) {
	const (
		driver   = 1 << 0
		register = 1 << 31
	)
	if is_put {
		x = d.regs.software_semaphore.put_semaphore(&d.dev, driver|register)
	} else {
		d.regs.software_semaphore.get_semaphore(&d.dev, "sw", driver)
		x = d.regs.software_semaphore.get_semaphore(&d.dev, "reg", register)
	}
	return
}

func (d *dev_x540) get_semaphore() { d.get_put_semaphore(false) }
func (d *dev_x540) put_semaphore() { d.get_put_semaphore(true) }

func (d *dev_x540) phy_init() {
}
