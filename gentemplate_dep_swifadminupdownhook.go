// autogenerated: do not edit!
// generated from gentemplate [gentemplate -id SwIfAdminUpDownHook -d Package=vnet -d DepsType=SwIfAdminUpDownHookVec -d Type=SwIfAdminUpDownHook -d Data=hooks github.com/platinasystems/elib/dep/dep.tmpl]

package vnet

import (
	"github.com/platinasystems/elib/dep"
)

type SwIfAdminUpDownHookVec struct {
	deps  dep.Deps
	hooks []SwIfAdminUpDownHook
}

func (t *SwIfAdminUpDownHookVec) Len() int {
	return t.deps.Len()
}

func (t *SwIfAdminUpDownHookVec) Get(i int) SwIfAdminUpDownHook {
	return t.hooks[t.deps.Index(i)]
}

func (t *SwIfAdminUpDownHookVec) Add(x SwIfAdminUpDownHook, ds ...*dep.Dep) {
	for i := range ds {
		t.deps.Add(ds[i])
	}
	t.hooks = append(t.hooks, x)
}