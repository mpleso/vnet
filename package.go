package vnet

import (
	"github.com/platinasystems/elib/parse"

	"fmt"
)

type Packager interface {
	GetPackage() *Package
	Configure(in *parse.Input)
}

type Package struct {
	name string
}

func (p *Package) GetPackage() *Package { return p }

type packageMain struct {
	packageByName parse.StringMap
	packages      []Packager
}

func (m *packageMain) AddPackage(name string, r Packager) uint {
	i := uint(len(m.packages))
	m.packageByName.Set(name, i)
	m.packages = append(m.packages, r)
	p := r.GetPackage()
	p.name = name
	return i
}

func (m *packageMain) GetPackage(i uint) Packager { return m.packages[i] }

func (p *Package) configure(r Packager, in *parse.Input) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("configure %s: %s: %s", p.name, e, in)
		}
	}()
	r.Configure(in)
	return
}

func (m *packageMain) Configure(in *parse.Input) (err error) {
	for !in.End() {
		var i uint
		if in.Parse("%v", m.packageByName, &i) {
			r := m.packages[i]
			p := r.GetPackage()
			err = p.configure(r, in)
			if err != nil {
				return
			}
		} else {
			return in.Error()
		}
	}
	return
}
