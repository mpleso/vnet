package vnet

func (n *InterfaceNode) slowPath(rv *RefVec, rs []Ref, is, ivʹ, nBytesʹ uint) (iv, nBytes uint) {
	iv, nBytes = ivʹ, nBytesʹ
	s := rs[is]
	for {
		// Copy buffer reference.
		(*rv)[iv] = s
		iv++

		if h := s.NextRef(); h == nil {
			break
		} else {
			s.RefHeader = *h
		}
		nBytes += s.DataLen()
		rv.Validate(iv)
	}
	return
}

type interfaceNodeThread struct {
	freeChan chan *RefVecIn
	n_alloc  uint
}

//go:generate gentemplate -d Package=vnet -id interfaceNodeThreadVec -d VecType=interfaceNodeThreadVec -d Type=*interfaceNodeThread github.com/platinasystems/elib/vec.tmpl

func (t *interfaceNodeThread) getRefVecIn(n *InterfaceNode, in *RefIn) (i *RefVecIn) {
	for {
		select {
		case i = <-t.freeChan:
			i.FreeRefs()
		default:
			if t.n_alloc < 2 {
				i = &RefVecIn{}
				t.n_alloc++
			}
		}
		if i != nil {
			break
		}
		n.Vnet.loop.Suspend(&in.In)
	}
	return
}

func (n *InterfaceNode) InterfaceOutput(ri *RefIn) {
	id := ri.ThreadId()
	n.threads.Validate(id)
	if n.threads[id] == nil {
		n.threads[id] = &interfaceNodeThread{}
		n.threads[id].freeChan = make(chan *RefVecIn, 64)
	}
	nt := n.threads[id]
	rvi := nt.getRefVecIn(n, ri)

	// Copy common fields.
	rvi.refInCommon = ri.refInCommon

	nRef := ri.Len()
	rvi.Refs.Validate(nRef - 1)

	n_left := nRef
	rs := ri.Refs[:]
	rv := rvi.Refs
	is, iv := uint(0), uint(0)
	nBytes := uint(0)
	for n_left >= 4 {
		rv[iv+0] = rs[is+0]
		rv[iv+1] = rs[is+1]
		rv[iv+2] = rs[is+2]
		rv[iv+3] = rs[is+3]
		nBytes += rs[is+0].DataLen() + rs[is+1].DataLen() + rs[is+2].DataLen() + rs[is+3].DataLen()
		iv += 4
		is += 4
		n_left -= 4
		if RefFlag4(NextValid, rs, is-4) {
			iv -= 4
			iv, nBytes = n.slowPath(&rv, rs, is-4, iv, nBytes)
			iv, nBytes = n.slowPath(&rv, rs, is-3, iv, nBytes)
			iv, nBytes = n.slowPath(&rv, rs, is-2, iv, nBytes)
			iv, nBytes = n.slowPath(&rv, rs, is-1, iv, nBytes)
		}
	}
	for n_left > 0 {
		rv[iv+0] = rs[is+0]
		nBytes += rs[is+0].DataLen()
		is += 1
		iv += 1
		n_left -= 1
		if RefFlag1(NextValid, rs, is-1) {
			iv -= 1
			iv, nBytes = n.slowPath(&rv, rs, is-1, iv, nBytes)
		}
	}

	t := n.Vnet.GetIfThread(ri.ThreadId())
	hw := n.Vnet.HwIf(n.Hi)
	IfTxCounter.Add(t, hw.si, nRef, nBytes)

	n.i.InterfaceOutput(rvi, nt.freeChan)
}
