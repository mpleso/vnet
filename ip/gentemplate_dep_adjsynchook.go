// autogenerated: do not edit!
// generated from gentemplate [gentemplate -id adjSyncHook -d Package=ip -d DepsType=adjSyncHookVec -d Type=adjSyncHook -d Data=adjSyncHooks github.com/platinasystems/elib/dep/dep.tmpl]

package ip

import (
	"github.com/platinasystems/elib/dep"
)

type adjSyncHookVec struct {
	deps         dep.Deps
	adjSyncHooks []adjSyncHook
}

func (t *adjSyncHookVec) Len() int {
	return t.deps.Len()
}

func (t *adjSyncHookVec) Get(i int) adjSyncHook {
	return t.adjSyncHooks[t.deps.Index(i)]
}

func (t *adjSyncHookVec) Add(x adjSyncHook, ds ...*dep.Dep) {
	if len(ds) == 0 {
		t.deps.Add(&dep.Dep{})
	} else {
		t.deps.Add(ds[0])
	}
	t.adjSyncHooks = append(t.adjSyncHooks, x)
}