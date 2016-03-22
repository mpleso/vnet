// Network counters + packet/byte counters optimized for large tables and cache locality.
package vnet

import (
	"github.com/platinasystems/elib"
	"math"
)

type CounterIndex uint32

// Array of single counters
type Counters struct {
	// Compact cache-friendly counters that (rarely) can overflow.
	mini elib.Uint16Vec
	// 64 bit counters to hold overflow from mini counters.
	maxi elib.Uint64Vec
	// For testing; nil for normal operation.
	debug elib.Uint64Vec
	// Counter values when last cleared or zero if counters have never been cleared.
	valuesLastClear elib.Uint64Vec
}

func (c *Counters) Validate(i CounterIndex) {
	c.mini.Validate(uint(i))
	c.maxi.Validate(uint(i))
	c.valuesLastClear.Validate(uint(i))
}

// Add x to counter with index i.
func (c *Counters) Add(i CounterIndex, x uint32) {
	mini := &c.mini[i]
	old := uint32(*mini)
	new := old + x
	*mini = uint16(new)
	if uint32(*mini) != new {
		c.maxi[i] += uint64(old + x)
		*mini = 0
	}
}

func (c *Counters) Value(i CounterIndex) (v uint64) {
	v = c.maxi[i] + uint64(c.mini[i]) - c.valuesLastClear[i]
	return
}

func (c *Counters) Clear(i uint32) {
	c.maxi[i] = 0
	c.mini[i] = 0
	c.valuesLastClear[i] = 0
}

type CombinedCounter struct{ packets, bytes uint64 }

func (c *CombinedCounter) Add(d *CombinedCounter) {
	c.packets += d.packets
	c.bytes += d.bytes
}

func (c *CombinedCounter) subNoValidate(d *CombinedCounter) {
	c.packets -= d.packets
	c.bytes -= d.bytes
}

func (c *CombinedCounter) Sub(d *CombinedCounter) {
	if d.packets > c.packets || d.bytes > c.bytes {
		panic("underflow")
	}
	c.subNoValidate(d)
}

func (c *CombinedCounter) Clear() {
	c.packets = 0
	c.bytes = 0
}

type miniCombinedCounter struct {
	// Packet count.
	packets uint16

	// Byte count = packets * ave-packet-size + byte_difference
	byteDiff int16
}

func (c *miniCombinedCounter) Clear() {
	c.packets = 0
	c.byteDiff = 0
}

//go:generate gentemplate -d Package=vnet -id miniCombinedCounter -d VecType=miniCombinedCounterVec -d Type=miniCombinedCounter github.com/platinasystems/elib/vec.tmpl
//go:generate gentemplate -d Package=vnet -id CombinedCounter -d VecType=CombinedCounterVec -d Type=CombinedCounter github.com/platinasystems/elib/vec.tmpl

type CombinedCounters struct {
	// Compact cache-friendly counters that may (rarely) overflow.
	mini miniCombinedCounterVec
	// 64 bit counters to hold overflow from mini counters.
	maxi CombinedCounterVec

	valuesLastClear CombinedCounterVec

	// Average packet size in bytes used to compute mini-counter byte differences.
	avePacketSize uint32

	// Current summed packets and bytes for average computation.
	sumPackets, sumBytes uint32

	avePacketSizeRecomputeInterval uint32
}

func (c *CombinedCounters) Validate(i CounterIndex) {
	c.mini.Validate(uint(i))
	c.maxi.Validate(uint(i))
	c.valuesLastClear.Validate(uint(i))
}

// Add packet and byte increment to counter.
func (c *CombinedCounters) Add(i CounterIndex, p, b uint32) {
	mini := &c.mini[i]

	op := uint32(mini.packets)
	od := int32(mini.byteDiff)

	np := op + p
	nd := od + int32(b-p*c.avePacketSize)

	mini.packets = uint16(np)
	mini.byteDiff = int16(nd)

	if uint32(mini.packets) != np || int32(mini.byteDiff) != nd {
		maxi := &c.maxi[i]

		nb := uint32(int32(op*c.avePacketSize)+od) + b

		maxi.packets += uint64(np)
		maxi.bytes += uint64(nb)
		mini.Clear()

		// Update average packet size.
		c.sumPackets += np
		c.sumBytes += nb
		if c.sumPackets >= c.avePacketSizeRecomputeInterval {
			c.recomputeAvePacketSize()
		}
	}
}

// Get counter value: 2 flavors.
func (c *CombinedCounters) Get(i CounterIndex, r *CombinedCounter) {
	*r = c.maxi[i]
	c.addMini(&c.mini[i], r)
	r.subNoValidate(&c.valuesLastClear[i])
}

func (c *CombinedCounters) Value(i CounterIndex) (v CombinedCounter) {
	c.Get(i, &v)
	return
}

func (c *CombinedCounters) Clear(i CounterIndex) {
	c.mini[i].Clear()
	c.maxi[i].Clear()
	c.valuesLastClear[i].Clear()
}

func (c *CombinedCounters) addMini(mini *miniCombinedCounter, maxi *CombinedCounter) {
	maxi.packets += uint64(mini.packets)
	maxi.bytes += uint64(int32(uint32(mini.packets)*c.avePacketSize) + int32(mini.byteDiff))
}

func (c *CombinedCounters) flushMini(mini *miniCombinedCounter, maxi *CombinedCounter) {
	c.addMini(mini, maxi)
	mini.Clear()
}

func (c *CombinedCounters) recomputeAvePacketSize() {
	if c.avePacketSizeRecomputeInterval == 0 {
		c.avePacketSizeRecomputeInterval = 4 << 10 // sane default
		return
	}

	newAve := uint32(math.Floor(.5 + float64(c.sumBytes)/float64(c.sumPackets)))
	if newAve != c.avePacketSize {
		// Flush counters since ave packet size is about to change.
		i, n_left := 0, len(c.mini)
		for n_left >= 4 {
			c.flushMini(&c.mini[i+0], &c.maxi[i+0])
			c.flushMini(&c.mini[i+1], &c.maxi[i+1])
			c.flushMini(&c.mini[i+2], &c.maxi[i+2])
			c.flushMini(&c.mini[i+3], &c.maxi[i+3])
			i += 4
			n_left -= 4
		}

		for n_left > 0 {
			c.flushMini(&c.mini[i+0], &c.maxi[i+0])
			i++
			n_left--
		}

		c.avePacketSize = newAve
		c.sumBytes, c.sumPackets = 0, 0
	}
}
