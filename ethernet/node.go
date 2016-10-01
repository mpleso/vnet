package ethernet

import (
	"github.com/platinasystems/go/vnet"
)

type nodeMain struct {
	inputNode inputNode
}

type inputNode struct {
	vnet.InOutNode
}

const (
	input_next_drop = iota
	input_next_punt
)

func (m *Main) nodeInit(v *vnet.Vnet) {
	n := &m.inputNode
	n.Next = []string{
		input_next_drop: "error",
		input_next_punt: "punt",
	}
	v.RegisterInOutNode(n, "ethernet-input")
}

func (node *inputNode) NodeInput(in *vnet.RefIn, out *vnet.RefOut) {
	node.Redirect(in, out, input_next_punt)
}
