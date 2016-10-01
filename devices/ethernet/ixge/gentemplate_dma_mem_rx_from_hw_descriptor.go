// autogenerated: do not edit!
// generated from gentemplate [gentemplate -d Package=ixge -id rx_from_hw_descriptor -d Type=rx_from_hw_descriptor -d VecType=rx_from_hw_descriptor_vec github.com/platinasystems/go/elib/hw/dma_mem.tmpl]

package ixge

import (
	"github.com/platinasystems/go/elib"
	"github.com/platinasystems/go/elib/hw"

	"reflect"
	"unsafe"
)

type rx_from_hw_descriptor_vec []rx_from_hw_descriptor

func fromByteSlice_rx_from_hw_descriptor(b []byte, l, c uint) (x rx_from_hw_descriptor_vec) {
	s := uint(unsafe.Sizeof(x[0]))
	if l == 0 {
		l = uint(len(b)) / s
		c = uint(cap(b))
	}
	return *(*rx_from_hw_descriptor_vec)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&b[0])),
		Len:  int(l),
		Cap:  int(c / s),
	}))
}

func (x rx_from_hw_descriptor_vec) toByteSlice() []byte {
	l := uint(len(x))
	l *= uint(unsafe.Sizeof(x[0]))
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&x[0])),
		Len:  int(l),
		Cap:  int(l)}))
}

func rx_from_hw_descriptorAllocAligned(n, a uint) (x rx_from_hw_descriptor_vec, id elib.Index) {
	var b []byte
	var c uint
	b, id, _, c = hw.DmaAllocAligned(n*uint(unsafe.Sizeof(x[0])), a)
	x = fromByteSlice_rx_from_hw_descriptor(b, n, c)
	return
}

func rx_from_hw_descriptorAlloc(n uint) (x rx_from_hw_descriptor_vec, id elib.Index) {
	return rx_from_hw_descriptorAllocAligned(n, 0)
}

func rx_from_hw_descriptorNew() (x rx_from_hw_descriptor_vec, id elib.Index) {
	return rx_from_hw_descriptorAlloc(1)
}

func (x *rx_from_hw_descriptor_vec) Free(id elib.Index) {
	hw.DmaFree(id)
	*x = nil
}

func (x *rx_from_hw_descriptor_vec) Get(id elib.Index) {
	*x = fromByteSlice_rx_from_hw_descriptor(hw.DmaGetData(id), 0, 0)
}

func (x *rx_from_hw_descriptor) PhysAddress() uintptr {
	return hw.DmaPhysAddress(uintptr(unsafe.Pointer(x)))
}
