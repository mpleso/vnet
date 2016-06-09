// autogenerated: do not edit!
// generated from gentemplate [gentemplate -id IfAddrAddDelHook -d Package=ip4 -d DepsType=IfAddrAddDelHookVec -d Type=IfAddrAddDelHook -d Data=hooks github.com/platinasystems/elib/dep/dep.tmpl]

package ip4

import (
	"github.com/platinasystems/elib/dep"
)

type IfAddrAddDelHookVec struct {
	deps  dep.Deps
	hooks []IfAddrAddDelHook
}

func (t *IfAddrAddDelHookVec) Len() int {
	return t.deps.Len()
}

func (t *IfAddrAddDelHookVec) Get(i int) IfAddrAddDelHook {
	return t.hooks[t.deps.Index(i)]
}

func (t *IfAddrAddDelHookVec) Add(x IfAddrAddDelHook, ds ...*dep.Dep) {
	if len(ds) == 0 {
		t.deps.Add(&dep.Dep{})
	} else {
		t.deps.Add(ds[0])
	}
	t.hooks = append(t.hooks, x)
}
