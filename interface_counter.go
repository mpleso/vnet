package vnet

type ifCounterKind int
type ifCombinedCounterKind int

const (
	IfDrops ifCounterKind = iota
	IfPunts
	nBuiltinSingleIfCounters
)
const (
	IfRxCounter ifCombinedCounterKind = iota
	IfTxCounter
	nBuiltinCombinedIfCounters
)

var builtinSingleIfCounterNames = [...]string{
	IfDrops: "drops", IfPunts: "punts",
}
var builtinCombinedIfCounterNames = [...]string{
	IfRxCounter: "rx", IfTxCounter: "tx",
}

func (v *interfaceMain) addCounterName(name string) {
	v.swIfCounterNames = append(v.swIfCounterNames, name)
}
func (v *interfaceMain) addCombinedCounterName(name string) {
	v.swIfCombinedCounterNames = append(v.swIfCombinedCounterNames, name)
}
func (v *interfaceMain) counterName(k ifCounterKind) string { return v.swIfCounterNames[k] }
func (v *interfaceMain) combinedCounterName(k ifCombinedCounterKind) string {
	return v.swIfCombinedCounterNames[k]
}

// Allocate given number of single counters.
func (v *interfaceMain) NewCounters(names []string) (i ifCounterKind) {
	n := uint(len(names))
	nSwIfs := v.swInterfaces.Len()
	for _, t := range v.ifThreads {
		i = ifCounterKind(t.singleCounters.Len())
		t.singleCounters.Resize(n)
		for j := uint(0); j < n; j++ {
			t.singleCounters[uint(i)+j].Validate(nSwIfs)
		}
	}
	for i := range names {
		v.addCounterName(names[i])
	}
	return
}

// Allocate given number of combined packet and byte counters.
func (v *Vnet) NewCombinedCounters(names []string) (i ifCounterKind) {
	n := uint(len(names))
	nSwIfs := v.swInterfaces.Len()
	for _, t := range v.ifThreads {
		i = ifCounterKind(t.combinedCounters.Len())
		t.combinedCounters.Resize(n)
		for j := uint(0); j < n; j++ {
			t.combinedCounters[uint(i)+j].Validate(nSwIfs)
		}
	}
	for i := range names {
		v.addCombinedCounterName(names[i])
	}
	return
}

// Add to given interface counters value.
func (c ifCounterKind) Add(t *interfaceThread, swIfIndex SwIfIndex, value uint) {
	t.singleCounters[c].Add(uint(swIfIndex), value)
}

// Add to given interface counters packets and bytes values.
func (c ifCombinedCounterKind) Add(t *interfaceThread, swIfIndex SwIfIndex, packets, bytes uint) {
	t.combinedCounters[c].Add(uint(swIfIndex), packets, bytes)
}

func (m *interfaceMain) doSingle(t *interfaceThread,
	kind ifCounterKind,
	enableZeroCounters bool,
	si SwIfIndex,
	f func(name string, value uint64)) {
	v := t.singleCounters[kind].Value(uint(si))
	if v != 0 || enableZeroCounters {
		f(m.swIfCounterNames[kind], v)
	}
}

func (m *interfaceMain) doCombined(t *interfaceThread,
	kind ifCombinedCounterKind,
	enableZeroCounters bool,
	si SwIfIndex,
	f func(name string, value uint64)) {
	v := t.combinedCounters[kind].Value(uint(si))
	if v.packets != 0 || enableZeroCounters {
		f(m.swIfCombinedCounterNames[kind]+" packets", v.packets)
		f(m.swIfCombinedCounterNames[kind]+" bytes", v.bytes)
	}
}

func (m *interfaceMain) foreachCounter(enableZeroCounters bool, si SwIfIndex, f func(name string, value uint64)) {
	for _, t := range m.ifThreads {
		// First builtin counters.
		for i := 0; i < len(builtinCombinedIfCounterNames); i++ {
			m.doCombined(t, ifCombinedCounterKind(i), enableZeroCounters, si, f)
		}
		for i := 0; i < len(builtinSingleIfCounterNames); i++ {
			m.doSingle(t, ifCounterKind(i), enableZeroCounters, si, f)
		}

		// Next user-defined counters.
		for i := len(builtinCombinedIfCounterNames); i < len(t.combinedCounters); i++ {
			m.doCombined(t, ifCombinedCounterKind(i), enableZeroCounters, si, f)
		}
	}
}

func (m *interfaceMain) clearIfCounters() {
	for _, t := range m.ifThreads {
		t.combinedCounters.Clear()
		t.singleCounters.Clear()
	}
}

func (m *interfaceMain) counterValidate(si SwIfIndex) {
	i := uint(si)
	for _, t := range m.ifThreads {
		for k := range t.combinedCounters {
			t.combinedCounters[k].Validate(i)
		}
		for k := range t.singleCounters {
			t.singleCounters[k].Validate(i)
		}
	}
}

func (v *Vnet) counterInit(t *interfaceThread) {
	t.singleCounters.Validate(uint(nBuiltinSingleIfCounters))
	t.combinedCounters.Validate(uint(nBuiltinCombinedIfCounters))

	if len(v.swIfCounterNames) < len(builtinSingleIfCounterNames) {
		for i := range builtinSingleIfCounterNames {
			v.addCounterName(builtinSingleIfCounterNames[i])
		}
	}
	if len(v.swIfCombinedCounterNames) < len(builtinCombinedIfCounterNames) {
		for i := range builtinCombinedIfCounterNames {
			v.addCombinedCounterName(builtinCombinedIfCounterNames[i])
		}
	}

	nSwIfs := v.swInterfaces.Len()
	for i := range t.singleCounters {
		t.singleCounters[i].Validate(nSwIfs)
	}
	for i := range t.combinedCounters {
		t.combinedCounters[i].Validate(nSwIfs)
	}
}
