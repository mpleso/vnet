package ip4

import (
	"github.com/platinasystems/elib/parse"
	"github.com/platinasystems/vnet"
	"github.com/platinasystems/vnet/ip"
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
	Protocol: ip.UDP,
	Src:      Address{0x1, 0x2, 0x3, 0x4},
	Dst:      Address{0x5, 0x6, 0x7, 0x8},
	Tos:      0,
	Ttl:      255,
	Ip_version_and_header_length: 0x45,
	Fragment_id:                  vnet.Uint16(0x1234).FromHost(),
	Flags_and_fragment_offset:    DontFragment.FromHost(),
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
	pg.AddStreamType(v, "ip4", m)
}
