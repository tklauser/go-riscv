// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package riscv64

import (
	"cmd/compile/internal/gc"
	"cmd/internal/obj"
	"cmd/internal/obj/riscv64"
)

// FIXME: This is incredibly inefficient, but nice and simple. Optimize.
// See other zerorange implementations for ideas
func zerorange(pp *gc.Progs, p *obj.Prog, off, cnt int64, _ *uint32) *obj.Prog {
	if cnt == 0 {
		return p
	}

	// Loop, zeroing one byte at a time.
	// ADD  $(off), SP, T0
	// ADD  $(cnt), T0, T1
	// loop:
	//      MOVB    ZERO, (T0)
	//      ADD     $1, T0
	//      BNE     T0, T1, loop

	// lo is an offset relative to the frame pointer, which we can't use from this function,
	// but adding the true frame size makes it into an offset from the stack pointer.  frame
	// is the local variable size, get the true frame pointer by adding the size of the saved
	// return address.
	p = appendpp(p, riscv64.AADD,
		obj.Addr{Type: obj.TYPE_CONST, Offset: int64(gc.Widthptr) + off},
		&obj.Addr{Type: obj.TYPE_REG, Reg: riscv64.REG_SP},
		obj.Addr{Type: obj.TYPE_REG, Reg: riscv64.REG_T0},
		0)
	p = appendpp(p, riscv64.AADD,
		obj.Addr{Type: obj.TYPE_CONST, Offset: cnt},
		&obj.Addr{Type: obj.TYPE_REG, Reg: riscv64.REG_T0},
		obj.Addr{Type: obj.TYPE_REG, Reg: riscv64.REG_T1},
		0)
	p = appendpp(p, riscv64.AMOVB,
		obj.Addr{Type: obj.TYPE_REG, Reg: riscv64.REG_ZERO},
		nil,
		obj.Addr{Type: obj.TYPE_MEM, Reg: riscv64.REG_T0},
		0)
	loop := p
	p = appendpp(p, riscv64.AADD,
		obj.Addr{Type: obj.TYPE_CONST, Offset: 1},
		nil,
		obj.Addr{Type: obj.TYPE_REG, Reg: riscv64.REG_T0},
		0)
	p = appendpp(p, riscv64.ABNE,
		obj.Addr{Type: obj.TYPE_REG, Reg: riscv64.REG_T0},
		nil,
		obj.Addr{Type: obj.TYPE_BRANCH},
		riscv64.REG_T1)
	gc.Patch(p, loop)

	return p
}

func appendpp(p *obj.Prog, as obj.As, from obj.Addr, from3 *obj.Addr, to obj.Addr, reg int16) *obj.Prog {
	q := gc.Ctxt.NewProg()
	q.As = as
	q.Pos = p.Pos
	q.From = from
	if from3 != nil {
		q.SetFrom3(*from3)
	}
	q.To = to
	q.Reg = reg
	q.Link = p.Link
	p.Link = q
	return q
}

func zeroAuto(pp *gc.Progs, n *gc.Node) {
	// TODO(tklauser)
}

func ginsnop(pp *gc.Progs) *obj.Prog {
	// Hardware nop is ADD $0, ZERO
	p := pp.Prog(riscv64.AADD)
	p.From.Type = obj.TYPE_CONST
	p.SetFrom3(obj.Addr{Type: obj.TYPE_REG, Reg: riscv64.REG_ZERO})
	p.To = *p.GetFrom3()
	return p
}
