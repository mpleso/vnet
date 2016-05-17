package vnet

import (
	"github.com/platinasystems/elib"
	"github.com/platinasystems/elib/cli"
	"github.com/platinasystems/elib/loop"

	"sort"
)

type swIfIndices struct {
	*Vnet
	ifs []SwIfIndex
}

func (x *swIfIndices) Init(v *Vnet) {
	x.Vnet = v
	for i := range v.swInterfaces.elts {
		if !v.swInterfaces.IsFree(uint(i)) {
			x.ifs = append(x.ifs, SwIfIndex(i))
		}
	}
}

func (h *swIfIndices) Less(i, j int) bool { return h.SwLessThan(h.SwIf(h.ifs[i]), h.SwIf(h.ifs[j])) }
func (h *swIfIndices) Swap(i, j int)      { h.ifs[i], h.ifs[j] = h.ifs[j], h.ifs[i] }
func (h *swIfIndices) Len() int           { return len(h.ifs) }

type showSwIf struct {
	Name    string `format:"%-30s" align:"left"`
	State   string `format:"%-12s" align:"left"`
	Counter string `format:"%-30s" align:"left"`
	Count   uint64 `format:"%16d" align:"center"`
}
type showSwIfs []showSwIf

func (v *Vnet) showSwIfs(c cli.Commander, w cli.Writer, s *cli.Scanner) {
	swIfs := &swIfIndices{}
	swIfs.Init(v)
	sort.Sort(swIfs)

	sifs := showSwIfs{}
	verbose := false
	for i := range swIfs.ifs {
		si := v.SwIf(swIfs.ifs[i])
		first := true
		v.foreachCounter(verbose, si.index, func(counter string, count uint64) {
			s := showSwIf{
				Counter: counter,
				Count:   count,
			}
			if first {
				first = false
				s.Name = si.IfName(v)
				s.State = si.flags.String()
			}
			sifs = append(sifs, s)
		})
	}
	elib.TabulateWrite(w, sifs)
}

func (v *Vnet) clearSwIfs(c cli.Commander, w cli.Writer, s *cli.Scanner) {
	v.clearIfCounters()
}

type showHwIf struct {
	Name     string `format:"%-30s"`
	Link     string `width:12`
	Hardware string `format:"%30s"`
}
type showHwIfs []showHwIf

func (ns showHwIfs) Less(i, j int) bool { return ns[i].Name < ns[j].Name }
func (ns showHwIfs) Swap(i, j int)      { ns[i], ns[j] = ns[j], ns[i] }
func (ns showHwIfs) Len() int           { return len(ns) }

func (v *Vnet) showHwIfs(c cli.Commander, w cli.Writer, s *cli.Scanner) {
	ifs := showHwIfs{}
	for _, hi := range v.hwInterfaces {
		h := hi.GetHwIf()
		if h.unprovisioned {
			continue
		}
		ifs = append(ifs, showHwIf{
			Name:     h.ifName,
			Link:     h.LinkString(),
			Hardware: "tbd",
		})
	}
	sort.Sort(ifs)
	elib.TabulateWrite(w, ifs)
}

func init() {
	cmds := [...]cli.Command{
		cli.Command{
			Name:      "show interfaces",
			ShortHelp: "show interface statistics",
			Action:    defaultVnet.showSwIfs,
		},
		cli.Command{
			Name:      "clear interfaces",
			ShortHelp: "clear interface statistics",
			Action:    defaultVnet.clearSwIfs,
		},
		cli.Command{
			Name:      "show hardware-interfaces",
			ShortHelp: "show hardware interface statistics",
			Action:    defaultVnet.showHwIfs,
		},
	}
	for i := range cmds {
		loop.CliAdd(&cmds[i])
	}
}
