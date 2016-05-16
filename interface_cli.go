package vnet

import (
	"github.com/platinasystems/elib/cli"
	"github.com/platinasystems/elib/loop"
)

func (v *Vnet) showSwIfStats(c cli.Commander, w cli.Writer, s *cli.Scanner) {
}

func (v *Vnet) clearSwIfStats(c cli.Commander, w cli.Writer, s *cli.Scanner) {
}

func init() {
	loop.CliAdd(&cli.Command{
		Name:      "show interfaces",
		ShortHelp: "show interface statistics",
		Action:    defaultVnet.showSwIfStats,
	})
	loop.CliAdd(&cli.Command{
		Name:      "clear interfaces",
		ShortHelp: "clear interface statistics",
		Action:    defaultVnet.clearSwIfStats,
	})
}
