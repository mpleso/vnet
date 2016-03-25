package sfp

import (
	"github.com/platinasystems/i2c"
	"unsafe"
)

type QsfpModule struct {
	// Read in when module is inserted and taken out of reset.
	sfpRegs SfpRegs

	signals [QsfpNSignal]QsfpSignal

	BusIndex   int
	BusAddress int
}

var dummy byte

func qsfpRegs() *QsfpRegs { return (*QsfpRegs)(unsafe.Pointer(&dummy)) }

func (r *reg8) offset() uint8 {
	return uint8(uintptr(unsafe.Pointer(r)) - uintptr(unsafe.Pointer(&dummy)))
}

func (r *reg16) offset() uint8 {
	return uint8(uintptr(unsafe.Pointer(r)) - uintptr(unsafe.Pointer(&dummy)))
}

func (m *QsfpModule) i2cDo(rw i2c.RW, regOffset uint8, size i2c.SMBusSize, data *i2c.SMBusData) (err error) {
	var bus i2c.Bus

	err = bus.Open(m.BusIndex)
	if err != nil {
		return
	}
	defer bus.Close()

	err = bus.ForceSlaveAddress(m.BusAddress)
	if err != nil {
		return
	}

	err = bus.Do(rw, regOffset, size, data)
	return
}

func (r *reg8) get(m *QsfpModule) byte {
	var data i2c.SMBusData
	err := m.i2cDo(i2c.Read, r.offset(), i2c.ByteData, &data)
	if err != nil {
		panic(err)
	}
	return data[0]
}

func (r *reg8) set(m *QsfpModule, v uint8) {
	var data i2c.SMBusData
	data[0] = v
	err := m.i2cDo(i2c.Write, r.offset(), i2c.ByteData, &data)
	if err != nil {
		panic(err)
	}
}

func (r *reg16) get(m *QsfpModule) (v uint16) {
	var data i2c.SMBusData
	err := m.i2cDo(i2c.Read, r.offset(), i2c.WordData, &data)
	if err != nil {
		panic(err)
	}
	return uint16(data[0])<<8 | uint16(data[1])
}

func (r *reg16) set(m *QsfpModule, v uint16) {
	var data i2c.SMBusData
	data[0] = uint8(v >> 8)
	data[1] = uint8(v)
	err := m.i2cDo(i2c.Write, r.offset(), i2c.WordData, &data)
	if err != nil {
		panic(err)
	}
}

func (r *regi16) get(m *QsfpModule) (v int16) { v = int16((*reg16)(r).get(m)); return }
func (r *regi16) set(m *QsfpModule, v int16)  { (*reg16)(r).set(m, uint16(v)) }

func (r *QsfpSignal) get() (v bool) {
	// GPIO
	return
}

func (r *QsfpSignal) set(v bool) {
	// GPIO
}
