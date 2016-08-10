package ip4

import (
	"github.com/platinasystems/elib/cli"
	"github.com/platinasystems/vnet/ip"

	"fmt"
	"sort"
)

type showIpFibRoute struct {
	table  ip.FibIndex
	prefix Prefix
	adj    ip.Adj
}

type showIpFibRoutes []showIpFibRoute

func (x showIpFibRoutes) Less(i, j int) bool {
	if cmp := int(x[i].table) - int(x[j].table); cmp != 0 {
		return cmp < 0
	}
	return x[i].prefix.LessThan(&x[j].prefix)
}

func (x showIpFibRoutes) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x showIpFibRoutes) Len() int      { return len(x) }

func (m *Main) showIpFib(c cli.Commander, w cli.Writer, in *cli.Input) (err error) {

	for !in.End() {
		switch {
		default:
			err = cli.ParseError
		}
	}

	rs := []showIpFibRoute{}
	for fi := range m.fibs {
		fib := m.fibs[fi]
		fib.foreach(func(p *Prefix, a ip.Adj) {
			rs = append(rs, showIpFibRoute{table: ip.FibIndex(fi), prefix: *p, adj: a})
		})
	}
	sort.Sort(showIpFibRoutes(rs))

	fmt.Fprintf(w, "%6s%30s%20s\n", "Table", "Destination", "Adjacency")
	for ri := range rs {
		r := &rs[ri]
		lines := []string{}
		adjs := m.GetAdj(r.adj)
		for ai := range adjs {
			initialSpace := "  "
			line := initialSpace
			if len(adjs) > 1 {
				line += fmt.Sprintf("%d: ", ai)
			}
			ss := adjs[ai].String(&m.Main)
			for _, s := range ss {
				lines = append(lines, line+s)
				line = initialSpace
			}
		}
		for i := range lines {
			if i == 0 {
				fmt.Fprintf(w, "%6d%30s%s\n", r.table, &r.prefix, lines[i])
			} else {
				fmt.Fprintf(w, "%6s%30s%s\n", "", "", lines[i])
			}
		}
	}

	return
}
