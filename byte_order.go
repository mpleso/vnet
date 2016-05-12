package vnet

// Network byte order helpers.
type Uint16 uint16
type Uint32 uint32
type Uint64 uint64

func (x Uint16) ToHost() uint16   { return swap16(uint16(x)) }
func (x Uint16) FromHost() Uint16 { return Uint16(swap16(uint16(x))) }
func (x *Uint16) Set(v uint)      { *x = Uint16(swap16(uint16(v))) }
