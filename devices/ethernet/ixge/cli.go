package ixge

import (
	"github.com/platinasystems/elib/cli"

	"fmt"
)

func (m *main) showDevs(c cli.Commander, w cli.Writer, in *cli.Input) (err error) {
	for _, dr := range m.devs {
		d := dr.get()

		for i := range d.tx_queues {
			q := &d.tx_queues[i]
			dr := q.get_regs()
			fmt.Fprintf(w, "txq %d: head %x tail %x\n", i, dr.head_index.get(d), dr.tail_index.get(d))
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
