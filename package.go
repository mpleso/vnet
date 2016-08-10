package vnet

import (
	"github.com/platinasystems/elib/cli"
	"github.com/platinasystems/elib/parse"

	"fmt"
)

type Packager interface {
	GetPackage() *Package
	Configure(in *parse.Input)
	Init() (err error)
	Exit() (err error)
}

type Package struct {
	Vnet *Vnet
	name string
}

func (p *Package) GetPackage() *Package { return p }
func (p *Package) Init() (err error)    { return } // likely overridden
func (p *Package) Exit() (err error)    { return } // likely overridden
func (p *Package) Configure(in *parse.Input) {
	panic(cli.ParseError)
}

type packageMain struct {
	packageByName parse.StringMap
	packages      []Packager
}

func (v *Vnet) AddPackage(name string, r Packager) uint {
	m := &v.packageMain
	if len(m.packages) == 0 { // 0th package is always empty.
		m.packages = append(m.packages, nil)
	}
	i := uint(len(m.packages))
	m.packageByName.Set(name, i)
	m.packages = append(m.packages, r)
	p := r.GetPackage()
	p.name = name
	p.Vnet = v
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

func (m *packageMain) ConfigurePackages(in *parse.Input) (err error) {
	// Parse package configuration.
	for !in.End() {
		var (
			i     uint
			subIn parse.Input
		)
		if in.Parse("%v %v", m.packageByName, &i, &subIn) {
			r := m.packages[i]
			p := r.GetPackage()
			err = p.configure(r, &subIn)
			if err != nil {
				return
			}
		} else {
			return in.Error()
		}
	}
	return
}

func (m *packageMain) InitPackages() (err error) {
	// Call package init functions.
	for i, p := range m.packages {
		if i == 0 {
			continue
		}
		err = p.Init()
		if err != nil {
			return
		}
	}
	return
}

func (m *packageMain) ExitPackages() (err error) {
	l := len(m.packages)
	for i := l - 1; i > 0; i-- {
		p := m.packages[i]
		err = p.Exit()
		if err != nil {
			return
		}
	}
	return
}
