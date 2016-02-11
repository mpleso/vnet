package ip4

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/platinasystems/vnet/ip"
	"github.com/platinasystems/vnet/layer"
)

const (
	AddressBytes   = 4
	HeaderBytes    = 20
	MORE_FRAGMENTS = 1 << 13
	DONT_FRAGMENT  = 1 << 14
	CONGESTION     = 1 << 15
)

type Address [AddressBytes]byte

type Header struct {
	/* 4 bit packet length (in 32bit units) and version VVVVLLLL.
	   e.g. for packets w/ no options ip_version_and_header_length == 0x45. */
	Ip_version_and_header_length uint8

	/* Type of service. */
	Tos uint8

	/* Total layer 3 packet length including this header. */
	Length uint16

	/* Fragmentation ID. */
	Fragment_id uint16

	/* 3 bits of flags and 13 bits of fragment offset (in units
	   of 8 byte quantities). */
	Flags_and_fragment_offset uint16

	/* Time to live decremented by router at each hop. */
	Ttl uint8

	/* Next level protocol packet. */
	Protocol ip.Protocol

	/* Checksum. */
	Checksum uint16

	/* Source and destination address. */
	Src Address
	Dst Address
}

func (a *Address) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", a[0], a[1], a[2], a[3])
}

func (a *Address) Scan(ss fmt.ScanState, verb rune) (err error) {
	tok, err := ss.Token(false, func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z':
			return true
		case r >= 'A' && r <= 'Z':
			return true
		case r >= '0' && r <= '9':
			return true
		case r == '.':
			return true
		case r == '-':
			return true
		}
		return false
	})

	as, err := net.LookupHost(string(tok))
	if err != nil {
		return
	}
	for i := range as {
		_, err = fmt.Sscanf(as[i], "%d.%d.%d.%d", &a[0], &a[1], &a[2], &a[3])
		if err == nil {
			return
		}
	}
	return
}

func (a *Address) Uint32() uint32 {
	return uint32(a[3]) | uint32(a[2])<<8 | uint32(a[1])<<16 | uint32(a[0])<<24
}

func (a *Address) FromUint32(x uint32) {
	a[0] = byte(x >> 24)
	a[1] = byte(x >> 16)
	a[2] = byte(x >> 8)
	a[3] = byte(x)
}

// 16 bit port; ^uint32(0) means no port given.
type Port uint32

const (
	NilPort Port = 0xffffffff
)

func (p *Port) Scan(ss fmt.ScanState, verb rune) (err error) {
	*p = NilPort
	tok, err := ss.Token(false, func(r rune) bool {
		switch {
		case r >= '0' && r <= '9':
			return true
		}
		return false
	})
	if len(tok) > 0 {
		var x uint32
		_, err = fmt.Sscanf(string(tok), "%d", &x)
		if err != nil {
			return
		}
		if x>>16 != 0 {
			err = fmt.Errorf("out or range: %s", string(tok))
			return
		}
		*p = Port(x)
	}
	return
}

type Socket struct {
	Address Address
	Port    Port
}

func (s *Socket) String() string {
	return s.Address.String() + fmt.Sprintf(":%d", s.Port)
}

func (s *Socket) Scan(ss fmt.ScanState, verb rune) (err error) {
	r, _, err := ss.ReadRune()
	if err != nil {
		return
	}
	if r == ':' {
		_, err = fmt.Fscanf(ss, "%d", &s.Port)
	} else {
		err = ss.UnreadRune()
		if err != nil {
			return
		}
		_, err = fmt.Fscanf(ss, "%s:%d", &s.Address, &s.Port)
	}
	return
}

func sum_1x8(sum, x uint64) uint64 {
	t := sum + x
	if t < x {
		t += 1
	}
	return t
}

func sum_2x1(sum uint64, x0, x1 uint8) uint64 {
	return sum_1x8(sum, uint64(x0)<<8+uint64(x1))
}

func sum_1x2(sum uint64, x uint16) uint64 {
	return sum_1x8(sum, uint64((x>>8)&0xff)<<8+uint64((x>>0)&0xff))
}

func sum_addr(sum uint64, x Address) uint64 {
	sum = sum_1x8(sum, uint64(x[0])<<8+uint64(x[1]))
	sum = sum_1x8(sum, uint64(x[2])<<8+uint64(x[3]))
	return sum
}

/* Reduce to 16 bits. */
func checksum_reduce(c uint64) uint16 {
	c = (c & 0xffffffff) + (c >> 32)
	c = (c & 0xffff) + (c >> 16)
	c = (c & 0xffff) + (c >> 16)
	c = (c & 0xffff) + (c >> 16)
	return uint16(c)
}

func (h *Header) checksum() uint16 {
	c := uint64(0)
	c = sum_2x1(c, h.Ip_version_and_header_length, h.Tos)
	c = sum_1x2(c, h.Length)
	c = sum_1x2(c, h.Fragment_id)
	c = sum_1x2(c, h.Flags_and_fragment_offset)
	c = sum_2x1(c, h.Ttl, uint8(h.Protocol))
	c = sum_addr(c, h.Src)
	c = sum_addr(c, h.Dst)
	return ^checksum_reduce(c)
}

func (h *Header) Len() int {
	return HeaderBytes
}

func (h *Header) Fin(layers []layer.Layer) {
	var sum int
	for _, l := range layers {
		sum += l.Len()
	}
	h.Length = uint16(HeaderBytes + sum)
	h.Checksum = h.checksum()
}

func (h *Header) Write(b *bytes.Buffer) {
	binary.Write(b, binary.BigEndian, h)
}
