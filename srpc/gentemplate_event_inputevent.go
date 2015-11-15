// autogenerated: do not edit!
// generated from gentemplate [gentemplate -d Package=srpc -id inputEvent -d Type=inputEvent github.com/platinasystems/elib/elog/event.tmpl]

package srpc

import (
	. "github.com/platinasystems/elib/elog"

	"fmt"
	"unsafe"
)

var inputEventType = &EventType{
	Name: "srpc.inputEvent",
}

func init() {
	t := inputEventType
	t.Stringer = stringer_inputEvent
	t.Encoder = encoder_inputEvent
	t.Decoder = decoder_inputEvent
	RegisterType(inputEventType)
}

func stringer_inputEvent(e *Event) string {
	x := (*inputEvent)(unsafe.Pointer(&e.Data[0]))
	return x.String()
}

func encoder_inputEvent(b []byte, e *Event) int {
	x := (interface{})((*inputEvent)(unsafe.Pointer(&e.Data[0])))
	if y, ok := x.(EventDataEncoder); ok {
		return y.Encode(b)
	} else {
		return copy(b, e.Data[:])
	}
}

func decoder_inputEvent(b []byte, e *Event) int {
	x := (interface{})((*inputEvent)(unsafe.Pointer(&e.Data[0])))
	if y, ok := x.(EventDataDecoder); ok {
		return y.Decode(b)
	} else {
		return copy(e.Data[:], b)
	}
}

func inputEventNew() (x *inputEvent) {
	e := Add(inputEventType)
	if unsafe.Sizeof(*x) > unsafe.Sizeof(e.Data) {
		panic(fmt.Sprintf("type too large: inputEvent %d bytes", unsafe.Sizeof(*x)))
	}
	x = (*inputEvent)(unsafe.Pointer(&e.Data[0]))
	return
}

func (x inputEvent) Log() {
	p := inputEventNew()
	*p = x
}

func (x *inputEvent) Event() (e *Event) {
	e = (*Event)(unsafe.Pointer(uintptr(unsafe.Pointer(x)) - unsafe.Offsetof(e.Data)))
	return
}

func (x *inputEvent) LogEventString(l *Log) string {
	e := x.Event()
	return fmt.Sprintf("%s: %s", e.EventString(l), x)
}

func (x *inputEvent) EventString() string { return x.LogEventString(DefaultLog) }
