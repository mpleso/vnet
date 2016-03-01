package layer

import (
	"bytes"
)

type Layer interface {
	// Number of packet bytes in this layer's payload.
	Len() int

	// Finalize this layer given finalized inner layers.
	// This allows, for example, the IP4 layer to compute length and checksum based on payload.
	Finalize([]Layer)

	// Append this layer's packet data to given buffer.
	Write(*bytes.Buffer)
}

func Make(args ...Layer) []byte {
	n := len(args)
	if n == 0 {
		return nil
	}
	for i := 0; i < n; i++ {
		args[n-1-i].Finalize(args[n-1-i : n-1])
	}
	b := new(bytes.Buffer)
	for i := 0; i < n; i++ {
		b.Grow(args[i].Len())
		args[i].Write(b)
	}
	return b.Bytes()
}

// Packet layer with incrementing data of given byte count.
type Incrementing struct{ Count int }

func (i *Incrementing) Len() int           { return i.Count }
func (i *Incrementing) Finalize(l []Layer) {}

func (i *Incrementing) Write(b *bytes.Buffer) {
	for j := 0; j < i.Count; j++ {
		b.WriteByte(byte(j % 256))
	}
}
