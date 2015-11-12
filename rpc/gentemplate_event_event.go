// autogenerated: do not edit!
// generated from gentemplate [gentemplate -d Package=rpc -id event -d Type=event github.com/platinasystems/elib/elog/event.tmpl]

package rpc

import (
	"fmt"
	. "github.com/platinasystems/elib/elog"
	"unsafe"
)

var eventType = &EventType{}

func init() {
	eventType.Stringer = func(e *Event) string {
		x := (*event)(unsafe.Pointer(&e.Data[0]))
		return x.String()
	}
}

func eventNew() (x *event) {
	e := Add(eventType)
	if unsafe.Sizeof(*x) > unsafe.Sizeof(e.Data) {
		panic(fmt.Sprintf("type too large: event %d bytes", unsafe.Sizeof(*x)))
	}
	x = (*event)(unsafe.Pointer(&e.Data[0]))
	return
}

func (x event) Log() {
	p := eventNew()
	*p = x
}

func (x *event) Event() (e *Event) {
	e = (*Event)(unsafe.Pointer(uintptr(unsafe.Pointer(x)) - unsafe.Offsetof(e.Data)))
	return
}

func (x *event) LogEventString(l *Log) string {
	e := x.Event()
	return fmt.Sprintf("%s: %s", e.EventString(l), x)
}

func (x *event) EventString() string { return x.LogEventString(DefaultLog) }
