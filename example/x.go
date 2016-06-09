package main

import (
	"github.com/platinasystems/elib/cli"
	"github.com/platinasystems/elib/loop"
	"github.com/platinasystems/vnet"
	"github.com/platinasystems/vnet/arp"
	"github.com/platinasystems/vnet/ethernet"
	"github.com/platinasystems/vnet/ip"
	"github.com/platinasystems/vnet/ip4"
)

type myNode struct {
	vnet.InterfaceNode
	ethernet.Interface
	pool loop.BufferPool
}

var MyNode = &myNode{}

const (
	error_one = iota
	error_two
	n_error
)

const (
	next_error = iota
	next_self
	n_next
)

func init() {
	vnet.AddInit(func(v *vnet.Vnet) {
		MyNode.Errors = []string{
			error_one: "error one",
			error_two: "error two",
		}
		MyNode.Next = []string{
			next_error: "error",
			next_self:  "my-node",
		}

		v.RegisterInterfaceNode(MyNode, MyNode.Hi, "my-node")

		v.CliAdd(&cli.Command{
			Name:      "a",
			ShortHelp: "a short help",
			Action: func(c cli.Commander, w cli.Writer, s *cli.Scanner) (err error) {
				n := uint(1)
				if s.Peek() != cli.EOF {
					if err = s.Parse("%d", &n); err != nil {
						return
					}
				}
				if n == 0 {
					MyNode.Activate(true)
				} else {
					MyNode.ActivateCount(n)
				}
				return
			},
		})
	})
}

func (n *myNode) ValidateSpeed(speed vnet.Bandwidth) (err error) { return }

func ip4Template(t *loop.BufferTemplate) {
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

func arpTemplate(t *loop.BufferTemplate) {
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

func (n *myNode) LoopInit(l *loop.Loop) {
	v := n.Vnet
	config := &ethernet.InterfaceConfig{
		Address: ethernet.Address{1, 2, 3, 4, 5, 6},
	}
	ethernet.RegisterInterface(v, MyNode, config, "my-node")

	// Link is always up for packet generator.
	n.SetLinkUp(true)
	n.SetAdminUp(true)

	t := &n.pool.BufferTemplate
	*t = *loop.DefaultBufferTemplate
	t.Size = 2048
	if true {
		ip4Template(t)
	} else {
		arpTemplate(t)
	}
	n.pool.Init()
}

func (n *myNode) InterfaceInput(o *vnet.RefOut) {
	toErr := &o.Outs[next_error]
	toErr.BufferPool = &n.pool
	toErr.AllocPoolRefs(&n.pool)
	t := n.GetIfThread()
	rs := toErr.Refs[:]
	nBytes := uint(0)
	for i := range rs {
		r := &rs[i]
		n.SetError(r, uint(i%n_error))
		nBytes += r.DataLen()
	}
	vnet.IfRxCounter.Add(t, n.Si(), uint(len(rs)), nBytes)
	toErr.SetLen(n.Vnet, uint(len(toErr.Refs)))
}

func (n *myNode) InterfaceOutput(i *vnet.RefVecIn) {
	i.FreeRefs()
}

func main() { vnet.Run() }
