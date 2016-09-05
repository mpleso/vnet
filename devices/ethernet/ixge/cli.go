package ixge

import (
	"github.com/platinasystems/elib/cli"

	"fmt"
)

func (m *main) showDevs(c cli.Commander, w cli.Writer, in *cli.Input) (err error) {
	for _, dr := range m.devs {
		d := dr.get()

		var v [4]reg
		v[0] = d.regs.interrupt.status_write_1_to_clear.get(d)
		v[1] = d.regs.tx_dma_control.get(d)
		v[2] = d.regs.rx_enable.get(d)
		v[3] = d.regs.xge_mac.mac_control.get(d)
		fmt.Fprintf(w, "%s: %x\n", d.Hi().Name(m.Vnet), v)
		for i := range d.tx_queues {
			q := &d.tx_queues[i]
			dr := q.get_regs()
			fmt.Fprintf(w, "txq %d: head %d tail %d\n", i, dr.head_index.get(d), dr.tail_index.get(d))
			v[0] = dr.descriptor_address[0].get(d)
			v[1] = dr.descriptor_address[1].get(d)
			v[2] = dr.n_descriptor_bytes.get(d)
			v[3] = dr.control.get(d)
			fmt.Fprintf(w, "%x\n", v)
		}

		for i := range d.rx_queues {
			q := &d.rx_queues[i]
			dr := q.get_regs()
			fmt.Fprintf(w, "rxq %d: head %d tail %d\n", i, dr.head_index.get(d), dr.tail_index.get(d))
			v[0] = dr.descriptor_address[0].get(d)
			v[1] = dr.descriptor_address[1].get(d)
			v[2] = dr.n_descriptor_bytes.get(d)
			v[3] = dr.control.get(d)
			fmt.Fprintf(w, "%x\n", v)
		}

	}
	return
}

func (m *main) cliInit() {
	v := m.Vnet
	cmds := [...]cli.Command{
		cli.Command{
			Name:      "show ixge",
			ShortHelp: "show Intel 10G interfaces",
			Action:    m.showDevs,
		},
	}
	for i := range cmds {
		v.CliAdd(&cmds[i])
	}
}
