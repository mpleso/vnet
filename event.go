package vnet

import (
	"github.com/platinasystems/elib/event"
)

type eventNode struct{ Node }

func (n *eventNode) EventHandler() {}

type eventMain struct {
	eventNode eventNode
}

func (v *Vnet) eventInit() {
	v.loop.RegisterNode(&v.eventMain.eventNode, "event-handler")
}

type Event struct {
	v *Vnet
}

func (e *Event) GetEvent() *Event { return e }

type Eventer interface {
	GetEvent() *Event
	event.Actor
}

func (v *Vnet) SignalEvent(r Eventer) {
	e := r.GetEvent()
	e.v = v
	l := v.GetLoop()
	l.AddEvent(r, &v.eventMain.eventNode)
}
