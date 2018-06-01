// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package riscv64

import (
	"cmd/internal/obj"
	"cmd/internal/sys"
)

func buildop(ctxt *obj.Link) {
}

// All unary instructions which write to their arguments (as opposed to reading
// from them) go here.  The assembly parser uses this information to populate
// its AST in a semantically reasonable way.
//
// Any instructions not listed here is assumed to either be non-unary or to read
// from its argument.
var unaryDst = map[obj.As]bool{
	ARDCYCLE:    true,
	ARDCYCLEH:   true,
	ARDTIME:     true,
	ARDTIMEH:    true,
	ARDINSTRET:  true,
	ARDINSTRETH: true,
}

var Linkriscv64 = obj.LinkArch{
	Arch:           sys.ArchRISCV64,
	Init:           buildop,
	Preprocess:     preprocess,
	Assemble:       assemble,
	Progedit:       progedit,
	UnaryDst:       unaryDst,
	DWARFRegisters: RISCV64DWARFRegisters,
}
