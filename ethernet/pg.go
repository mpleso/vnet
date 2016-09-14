package ethernet

import (
	"github.com/platinasystems/elib/parse"
	"github.com/platinasystems/vnet"
	"github.com/platinasystems/vnet/pg"
)

type pgStream struct {
	pg.Stream
	h []vnet.PacketHeader
}

func (s *pgStream) PacketHeaders() []vnet.PacketHeader {
	if len(s.h) == 1 {
		s.h = append(s.h,
			&vnet.IncrementingPayload{Count: s.MaxSize() - HeaderBytes})
	}
	return s.h
}

type pgMain struct{}

var defaultHeader = Header{
	Type: IP4.FromHost(),
	Src:  Address{0xe0, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5},
	Dst:  Address{0xea, 0xeb, 0xec, 0xed, 0xee, 0xef},
}

func (m *pgMain) ParseStream(in *parse.Input) (r pg.Streamer, err error) {
	var s pgStream
	h := defaultHeader
	for !in.End() {
		switch {
		case in.Parse("%v", &h):
		default:
			err = parse.ErrInput
			return
		}
	}
	if err == nil {
		s.h = append(s.h, &h)
		r = &s
	}
	return
}

func (m *pgMain) pgInit(v *vnet.Vnet) {
	pg.AddStreamType(v, "ethernet", m)
}
