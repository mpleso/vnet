package ixge

import (
	"github.com/platinasystems/elib"
	"github.com/platinasystems/vnet"

	"fmt"
)

func (d *dev) set_queue_interrupt_mapping(rt vnet.RxTx, queue uint, irq interrupt) {
	i0, i1 := queue/2, queue%2
	v := d.regs.interrupt.queue_mapping[i0].get(d)
	shift := 16 * i1
	if rt == vnet.Tx {
		shift += 8
	}
	m := reg(0xff) << shift
	const valid = 1 << 7
	x := (valid | (reg(irq) & 0x1f)) << shift
	v = (v & m) | x
	d.regs.interrupt.queue_mapping[i0].set(d, v)
	d.queues_for_interrupt[rt].Validate(uint(irq))
	b := d.queues_for_interrupt[rt][irq]
	b = b.Set(queue)
	d.queues_for_interrupt[rt][irq] = b
}

func (d *dev) foreach_queue_for_interrupt(rt vnet.RxTx, i interrupt, f func(queue uint)) {
	g := func(queue uint) (err error) { f(queue); return }
	d.queues_for_interrupt[rt][i].ForeachSetBit(g)
}

type interrupt uint

const (
	irq_n_queue                 = 16
	irq_queue_0       interrupt = iota
	irq_flow_director           = iota + 16
	irq_rx_missed_packet
	irq_pcie_exception
	irq_mailbox
	irq_link_state_change
	irq_link_security
	irq_manageability
	_
	irq_time_sync
	irq_gpio_0
	irq_gpio_1
	irq_gpio_2
	irq_ecc_error
	irq_phy
	irq_tcp_timer_expired
	irq_other
)

var irqStrings = [...]string{
	irq_flow_director:     "flow director",
	irq_rx_missed_packet:  "rx missed packet",
	irq_pcie_exception:    "pcie exception",
	irq_mailbox:           "mailbox",
	irq_link_state_change: "link state change",
	irq_link_security:     "link security",
	irq_manageability:     "manageability",
	irq_time_sync:         "time sync",
	irq_gpio_0:            "gpio 0",
	irq_gpio_1:            "gpio 1",
	irq_gpio_2:            "gpio 2",
	irq_ecc_error:         "ecc error",
	irq_phy:               "phy",
	irq_tcp_timer_expired: "tcp timer expired",
	irq_other:             "other",
}

func (i interrupt) String() (s string) {
	if i < irq_n_queue {
		s = fmt.Sprintf("queue irq %d", i)
	} else {
		s = elib.StringerHex(irqStrings[:], int(i))
	}
	return
}

func (d *dev) interrupt_dispatch(i uint) {
	irq := interrupt(i)
	switch {
	case irq < irq_n_queue:
		d.foreach_queue_for_interrupt(vnet.Rx, irq, d.rx_queue_interrupt)
		d.foreach_queue_for_interrupt(vnet.Tx, irq, d.tx_queue_interrupt)
	case irq == irq_link_state_change:
		d.link_state_change()
	default:
		panic(fmt.Errorf("ixge unexpected interrupt: %s", irq))
	}
}

func (d *dev) Interrupt() {
	// Get status and ack interrupt.
	s := d.regs.interrupt.status_write_1_to_set.get(d)
	if s != 0 {
		d.regs.interrupt.status_write_1_to_clear.set(d, s)
	}
	elib.Word(s).ForeachSetBit(d.interrupt_dispatch)
}
