package layer

import (
	"bytes"
)

type Layer interface {
	Len() int
	Fin([]Layer)
	Write(*bytes.Buffer)
}

func Make(args ...Layer) []byte {
	n := len(args)
	for i := 0; i < n; i++ {
		args[n-1-i].Fin(args[n-1-i : n-1])
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

func (i *Incrementing) Len() int      { return i.Count }
func (i *Incrementing) Fin(l []Layer) {}

func (i *Incrementing) Write(b *bytes.Buffer) {
	for j := 0; j < i.Count; j++ {
		b.WriteByte(byte(j % 256))
	}
}
