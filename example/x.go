package main

import (
	"github.com/platinasystems/elib/cli"
	"github.com/platinasystems/elib/hw"
	"github.com/platinasystems/elib/parse"
	"github.com/platinasystems/vnet"
	"github.com/platinasystems/vnet/arp"
	"github.com/platinasystems/vnet/devices/ethernet/ixge"
	"github.com/platinasystems/vnet/ethernet"
	"github.com/platinasystems/vnet/ip"
	"github.com/platinasystems/vnet/ip4"
	"github.com/platinasystems/vnet/ip6"
	"github.com/platinasystems/vnet/unix"

	"fmt"
	"os"
)

type myNode struct {
	vnet.InterfaceNode
	ethernet.Interface
	vnet.Package
	pool     hw.BufferPool
	nPackets uint
	next     uint
	isUnix   bool
}

var (
	MyNode        = &myNode{}
	myNodePackage uint
)

const (
	error_one = iota
	error_two
	tx_packets_dropped
	n_error
)

const (
	next_error = iota
	next_punt
	next_ethernet_input
	next_ip4_input_valid_checksum
	next_ip6_input
	n_next
)

func init() {
	vnet.AddInit(func(v *vnet.Vnet) {
		MyNode.Errors = []string{
			error_one:          "error one",
			error_two:          "error two",
			tx_packets_dropped: "tx packets dropped",
		}
		MyNode.Next = []string{
			next_error:                    "error",
			next_punt:                     "punt",
			next_ethernet_input:           "ethernet-input",
			next_ip4_input_valid_checksum: "ip4-input-valid-checksum",
			next_ip6_input:                "ip6-input",
		}

		v.RegisterInterfaceNode(MyNode, MyNode.Hi(), "my-node")

		v.CliAdd(&cli.Command{
			Name:      "a",
			ShortHelp: "a short help",
			Action: func(c cli.Commander, w cli.Writer, in *cli.Input) error {
				n := MyNode
				n.nPackets = 1
				n.next = next_error
				for !in.End() {
					var next_name string
					if in.Parse("%d", &n.nPackets) {
						if n.nPackets == 0 { // no limit
							n.nPackets = ^uint(0)
						}
					} else if in.Parse("next punt") {
						n.next = next_punt
					} else if in.Parse("next error") {
						n.next = next_error
					} else if in.Parse("next ethernet-input") {
						n.next = next_ethernet_input
					} else if in.Parse("next ip4-input") {
						n.next = next_ip4_input_valid_checksum
					} else if in.Parse("next ip6-input") {
						n.next = next_ip6_input
					} else if in.Parse("next %s", &next_name) {
						n.next = v.AddNamedNext(n, next_name)
					} else {
						return cli.ParseError
					}
				}
				n.Activate(true)
				return nil
			},
		})
	})
}

func (n *myNode) ValidateSpeed(speed vnet.Bandwidth) (err error) { return }

const (
	s1_counter vnet.HwIfCounterKind = iota
	s2_counter
)

const (
	c1_counter vnet.HwIfCombinedCounterKind = iota
	c2_counter
)

func (n *myNode) GetHwInterfaceCounters(nm *vnet.InterfaceCounterNames, t *vnet.InterfaceThread) {
	nm.Single = []string{
		s1_counter: "s1",
		s2_counter: "s2",
	}
	nm.Combined = []string{
		c1_counter: "c1",
		c2_counter: "c2",
	}
}

func ip4Template(t *hw.BufferTemplate) {
	t.Data = vnet.MakePacket(
		&ethernet.Header{
			Type: ethernet.IP4.FromHost(),
			Src:  ethernet.Address{0xe0, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5},
			Dst:  ethernet.Address{0xea, 0xeb, 0xec, 0xed, 0xee, 0xef},
		},
		&ip4.Header{
			Protocol: ip.UDP,
			Src:      ip4.Address{0x1, 0x2, 0x3, 0x4},
			Dst:      ip4.Address{0x5, 0x6, 0x7, 0x8},
			Tos:      0,
			Ttl:      255,
			Ip_version_and_header_length: 0x45,
			Fragment_id:                  vnet.Uint16(0x1234).FromHost(),
			Flags_and_fragment_offset:    ip4.DontFragment.FromHost(),
		},
		&vnet.IncrementingPayload{Count: t.Size - ethernet.HeaderBytes - ip4.HeaderBytes},
	)
}

func arpTemplate(t *hw.BufferTemplate) {
	t.Data = vnet.MakePacket(
		&ethernet.Header{
			Type: ethernet.ARP.FromHost(),
			Src:  ethernet.Address{0xe0, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5},
			Dst:  ethernet.BroadcastAddr,
		},
		&arp.HeaderEthernetIp4{
			Header: arp.Header{
				Opcode:          arp.Request.FromHost(),
				L2Type:          arp.L2TypeEthernet.FromHost(),
				L3Type:          vnet.Uint16(ethernet.IP4.FromHost()),
				NL2AddressBytes: ethernet.AddressBytes,
				NL3AddressBytes: ip4.AddressBytes,
			},
			Addrs: [2]arp.EthernetIp4Addr{
				arp.EthernetIp4Addr{
					Ethernet: ethernet.Address{0xa0, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5},
					Ip4:      ip4.Address{10, 11, 12, 13},
				},
				arp.EthernetIp4Addr{
					Ethernet: ethernet.Address{0xb0, 0xb1, 0xb2, 0xb3, 0xb4, 0xb5},
					Ip4:      ip4.Address{20, 21, 22, 23},
				},
			},
		},
	)
}

func (n *myNode) Init() (err error) {
	v := n.Vnet
	config := &ethernet.InterfaceConfig{
		Address: ethernet.Address{0, 1, 2, 3, 4, 5},
	}
	ethernet.RegisterInterface(v, MyNode, config, "my-node")

	// Link is always up for packet generator.
	n.SetLinkUp(true)
	n.SetAdminUp(true)

	t := &n.pool.BufferTemplate
	*t = *hw.DefaultBufferTemplate
	t.Size = 64
	if true {
		ip4Template(t)
	} else {
		arpTemplate(t)
	}
	n.pool.Name = n.Name()
	v.AddBufferPool(&n.pool)
	return
}

func (n *myNode) Configure(in *parse.Input) {
	if in.Parse("tuntap") {
		n.isUnix = true
	} else {
		panic(parse.ErrInput)
	}
}

func (n *myNode) IsUnix() bool { return n.isUnix }

func (n *myNode) InterfaceInput(o *vnet.RefOut) {
	out := &o.Outs[n.next]
	out.BufferPool = &n.pool
	t := n.GetIfThread()
	nPackets := n.nPackets
	if l := out.Cap(); nPackets > l {
		nPackets = l
	}
	out.AllocPoolRefs(&n.pool, nPackets)
	rs := out.Refs[:]
	nBytes := uint(0)
	for i := uint(0); i < nPackets; i++ {
		r := &rs[i]
		n.SetError(r, uint(i%n_error))
		nBytes += r.DataLen()
	}
	vnet.IfRxCounter.Add(t, n.Si(), nPackets, nBytes)
	c1_counter.Add(t, n.Hi(), nPackets, nBytes)
	s1_counter.Add(t, n.Hi(), nPackets)
	out.SetLen(n.Vnet, nPackets)
	if n.nPackets != ^uint(0) {
		n.nPackets -= nPackets
	}
	n.Activate(n.nPackets > 0)
}

func (n *myNode) InterfaceOutput(i *vnet.RefVecIn, f chan *vnet.RefVecIn) {
	n.CountError(tx_packets_dropped, i.NPackets())
	f <- i
}

func main() {
	v := &vnet.Vnet{}

	// Select packages we want to run with.
	unix.Init(v)
	ethernet.Init(v)
	ip4.Init(v)
	ip6.Init(v)
	ixge.Init(v)
	myNodePackage = v.AddPackage("my-node", MyNode)

	var in parse.Input
	in.Add(os.Args[1:]...)
	err := v.Run(&in)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}
