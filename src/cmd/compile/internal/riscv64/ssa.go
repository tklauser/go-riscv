// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package riscv64

import (
	"math"

	"cmd/compile/internal/gc"
	"cmd/compile/internal/ssa"
	"cmd/compile/internal/types"
	"cmd/internal/obj"
	"cmd/internal/obj/riscv64"
)

// ssaRegToReg maps ssa register numbers to obj register numbers.
var ssaRegToReg = []int16{
	riscv64.REG_X0,
	// X1 (RA): unused
	riscv64.REG_X2,
	riscv64.REG_X3,
	riscv64.REG_X4,
	riscv64.REG_X5,
	riscv64.REG_X6,
	riscv64.REG_X7,
	riscv64.REG_X8,
	riscv64.REG_X9,
	riscv64.REG_X10,
	riscv64.REG_X11,
	riscv64.REG_X12,
	riscv64.REG_X13,
	riscv64.REG_X14,
	riscv64.REG_X15,
	riscv64.REG_X16,
	riscv64.REG_X17,
	riscv64.REG_X18,
	riscv64.REG_X19,
	riscv64.REG_X20,
	riscv64.REG_X21,
	riscv64.REG_X22,
	riscv64.REG_X23,
	riscv64.REG_X24,
	riscv64.REG_X25,
	riscv64.REG_X26,
	riscv64.REG_X27,
	riscv64.REG_X28,
	riscv64.REG_X29,
	riscv64.REG_X30,
	riscv64.REG_X31,
	riscv64.REG_F0,
	riscv64.REG_F1,
	riscv64.REG_F2,
	riscv64.REG_F3,
	riscv64.REG_F4,
	riscv64.REG_F5,
	riscv64.REG_F6,
	riscv64.REG_F7,
	riscv64.REG_F8,
	riscv64.REG_F9,
	riscv64.REG_F10,
	riscv64.REG_F11,
	riscv64.REG_F12,
	riscv64.REG_F13,
	riscv64.REG_F14,
	riscv64.REG_F15,
	riscv64.REG_F16,
	riscv64.REG_F17,
	riscv64.REG_F18,
	riscv64.REG_F19,
	riscv64.REG_F20,
	riscv64.REG_F21,
	riscv64.REG_F22,
	riscv64.REG_F23,
	riscv64.REG_F24,
	riscv64.REG_F25,
	riscv64.REG_F26,
	riscv64.REG_F27,
	riscv64.REG_F28,
	riscv64.REG_F29,
	riscv64.REG_F30,
	riscv64.REG_F31,
	0, // SB isn't a real register.  We fill an Addr.Reg field with 0 in this case.
}

func loadByType(t *types.Type) obj.As {
	width := t.Size()

	if t.IsFloat() {
		switch width {
		case 4:
			return riscv64.AMOVF
		case 8:
			return riscv64.AMOVD
		default:
			gc.Fatalf("unknown float width for load %d in type %v", width, t)
			return 0
		}
	}

	switch width {
	case 1:
		if t.IsSigned() {
			return riscv64.AMOVB
		} else {
			return riscv64.AMOVBU
		}
	case 2:
		if t.IsSigned() {
			return riscv64.AMOVH
		} else {
			return riscv64.AMOVHU
		}
	case 4:
		if t.IsSigned() {
			return riscv64.AMOVW
		} else {
			return riscv64.AMOVWU
		}
	case 8:
		return riscv64.AMOV
	default:
		gc.Fatalf("unknown width for load %d in type %v", width, t)
		return 0
	}
}

// storeByType returns the store instruction of the given type.
func storeByType(t *types.Type) obj.As {
	width := t.Size()

	if t.IsFloat() {
		switch width {
		case 4:
			return riscv64.AMOVF
		case 8:
			return riscv64.AMOVD
		default:
			gc.Fatalf("unknown float width for store %d in type %v", width, t)
			return 0
		}
	}

	switch width {
	case 1:
		return riscv64.AMOVB
	case 2:
		return riscv64.AMOVH
	case 4:
		return riscv64.AMOVW
	case 8:
		return riscv64.AMOV
	default:
		gc.Fatalf("unknown width for store %d in type %v", width, t)
		return 0
	}
}

// largestMove returns the largest move instruction possible and its size,
// given the alignment of the total size of the move.
//
// e.g., a 16-byte move may use MOV, but an 11-byte move must use MOVB.
//
// Note that the moves may not be on naturally aligned addresses depending on
// the source and destination.
//
// This matches the calculation in ssa.moveSize.
func largestMove(alignment int64) (obj.As, int64) {
	switch {
	case alignment%8 == 0:
		return riscv64.AMOV, 8
	case alignment%4 == 0:
		return riscv64.AMOVW, 4
	case alignment%2 == 0:
		return riscv64.AMOVH, 2
	default:
		return riscv64.AMOVB, 1
	}
}

// markMoves marks any MOVXconst ops that need to avoid clobbering flags.
// RISC-V has no flags, so this is a no-op.
func ssaMarkMoves(s *gc.SSAGenState, b *ssa.Block) {}

func ssaGenValue(s *gc.SSAGenState, v *ssa.Value) {
	switch v.Op {
	case ssa.OpCopy, ssa.OpRISCV64MOVconvert:
		if v.Type.IsMemory() {
			return
		}
		rs := v.Args[0].Reg()
		rd := v.Reg()
		if rs == rd {
			return
		}
		as := riscv64.AMOV
		if v.Type.IsFloat() {
			as = riscv64.AMOVD
		}
		p := s.Prog(as)
		p.From.Type = obj.TYPE_REG
		p.From.Reg = rs
		p.To.Type = obj.TYPE_REG
		p.To.Reg = rd
	case ssa.OpLoadReg:
		if v.Type.IsFlags() {
			v.Fatalf("load flags not implemented: %v", v.LongString())
			return
		}
		p := s.Prog(loadByType(v.Type))
		gc.AddrAuto(&p.From, v.Args[0])
		p.To.Type = obj.TYPE_REG
		p.To.Reg = v.Reg()
	case ssa.OpStoreReg:
		if v.Type.IsFlags() {
			v.Fatalf("store flags not implemented: %v", v.LongString())
			return
		}
		p := s.Prog(storeByType(v.Type))
		p.From.Type = obj.TYPE_REG
		p.From.Reg = v.Args[0].Reg()
		gc.AddrAuto(&p.To, v)
	case ssa.OpRISCV64ADD, ssa.OpRISCV64SUB, ssa.OpRISCV64XOR, ssa.OpRISCV64OR, ssa.OpRISCV64AND,
		ssa.OpRISCV64SLL, ssa.OpRISCV64SRA, ssa.OpRISCV64SRL,
		ssa.OpRISCV64SLT, ssa.OpRISCV64SLTU, ssa.OpRISCV64MUL, ssa.OpRISCV64MULW, ssa.OpRISCV64MULH,
		ssa.OpRISCV64MULHU, ssa.OpRISCV64DIV, ssa.OpRISCV64DIVU, ssa.OpRISCV64DIVW,
		ssa.OpRISCV64DIVUW, ssa.OpRISCV64REM, ssa.OpRISCV64REMU, ssa.OpRISCV64REMW,
		ssa.OpRISCV64REMUW,
		ssa.OpRISCV64FADDS, ssa.OpRISCV64FSUBS, ssa.OpRISCV64FMULS, ssa.OpRISCV64FDIVS,
		ssa.OpRISCV64FEQS, ssa.OpRISCV64FNES, ssa.OpRISCV64FLTS, ssa.OpRISCV64FLES,
		ssa.OpRISCV64FADDD, ssa.OpRISCV64FSUBD, ssa.OpRISCV64FMULD, ssa.OpRISCV64FDIVD,
		ssa.OpRISCV64FEQD, ssa.OpRISCV64FNED, ssa.OpRISCV64FLTD, ssa.OpRISCV64FLED:
		r := v.Reg()
		r1 := v.Args[0].Reg()
		r2 := v.Args[1].Reg()
		p := s.Prog(v.Op.Asm())
		p.From.Type = obj.TYPE_REG
		p.From.Reg = r2
		p.SetFrom3(obj.Addr{
			Type: obj.TYPE_REG,
			Reg:  r1,
		})
		p.To.Type = obj.TYPE_REG
		p.To.Reg = r
	case ssa.OpRISCV64FSQRTS, ssa.OpRISCV64FNEGS, ssa.OpRISCV64FSQRTD, ssa.OpRISCV64FNEGD,
		ssa.OpRISCV64FMVSX, ssa.OpRISCV64FMVDX,
		ssa.OpRISCV64FCVTSW, ssa.OpRISCV64FCVTSL, ssa.OpRISCV64FCVTWS, ssa.OpRISCV64FCVTLS,
		ssa.OpRISCV64FCVTDW, ssa.OpRISCV64FCVTDL, ssa.OpRISCV64FCVTWD, ssa.OpRISCV64FCVTLD, ssa.OpRISCV64FCVTDS, ssa.OpRISCV64FCVTSD:
		p := s.Prog(v.Op.Asm())
		p.From.Type = obj.TYPE_REG
		p.From.Reg = v.Args[0].Reg()
		p.To.Type = obj.TYPE_REG
		p.To.Reg = v.Reg()
	case ssa.OpRISCV64ADDI, ssa.OpRISCV64XORI, ssa.OpRISCV64ORI, ssa.OpRISCV64ANDI,
		ssa.OpRISCV64SLLI, ssa.OpRISCV64SRAI, ssa.OpRISCV64SRLI, ssa.OpRISCV64SLTI,
		ssa.OpRISCV64SLTIU:
		p := s.Prog(v.Op.Asm())
		p.From.Type = obj.TYPE_CONST
		p.From.Offset = v.AuxInt
		p.SetFrom3(obj.Addr{Type: obj.TYPE_REG, Reg: v.Args[0].Reg()})
		p.To.Type = obj.TYPE_REG
		p.To.Reg = v.Reg()
	case ssa.OpRISCV64MOVBconst, ssa.OpRISCV64MOVHconst, ssa.OpRISCV64MOVWconst, ssa.OpRISCV64MOVDconst:
		p := s.Prog(v.Op.Asm())
		p.From.Type = obj.TYPE_CONST
		p.From.Offset = v.AuxInt
		p.To.Type = obj.TYPE_REG
		p.To.Reg = v.Reg()
	case ssa.OpRISCV64MOVSconst:
		p := s.Prog(v.Op.Asm())
		// Convert the float to the equivalent integer literal so we can
		// move it using existing infrastructure.
		p.From.Type = obj.TYPE_CONST
		p.From.Offset = int64(int32(math.Float32bits(float32(math.Float64frombits(uint64(v.AuxInt))))))
		p.To.Type = obj.TYPE_REG
		p.To.Reg = v.Reg()
	case ssa.OpRISCV64MOVaddr:
		p := s.Prog(v.Op.Asm())
		p.From.Type = obj.TYPE_ADDR
		p.To.Type = obj.TYPE_REG
		p.To.Reg = v.Reg()

		var wantreg string
		// MOVW $sym+off(base), R
		switch v.Aux.(type) {
		default:
			v.Fatalf("aux is of unknown type %T", v.Aux)
		case *obj.LSym:
			wantreg = "SB"
			gc.AddAux(&p.From, v)
		case *gc.Node:
			wantreg = "SP"
			gc.AddAux(&p.From, v)
		case nil:
			// No sym, just MOVW $off(SP), R
			wantreg = "SP"
			p.From.Reg = riscv64.REG_SP
			p.From.Offset = v.AuxInt
		}
		if reg := v.Args[0].RegName(); reg != wantreg {
			v.Fatalf("bad reg %s for symbol type %T, want %s", reg, v.Aux, wantreg)
		}
	case ssa.OpRISCV64MOVBload, ssa.OpRISCV64MOVHload, ssa.OpRISCV64MOVWload, ssa.OpRISCV64MOVDload,
		ssa.OpRISCV64MOVBUload, ssa.OpRISCV64MOVHUload, ssa.OpRISCV64MOVWUload,
		ssa.OpRISCV64FMOVWload, ssa.OpRISCV64FMOVDload:
		p := s.Prog(v.Op.Asm())
		p.From.Type = obj.TYPE_MEM
		p.From.Reg = v.Args[0].Reg()
		gc.AddAux(&p.From, v)
		p.To.Type = obj.TYPE_REG
		p.To.Reg = v.Reg()
	case ssa.OpRISCV64MOVBstore, ssa.OpRISCV64MOVHstore, ssa.OpRISCV64MOVWstore, ssa.OpRISCV64MOVDstore,
		ssa.OpRISCV64FMOVWstore, ssa.OpRISCV64FMOVDstore:
		p := s.Prog(v.Op.Asm())
		p.From.Type = obj.TYPE_REG
		p.From.Reg = v.Args[1].Reg()
		p.To.Type = obj.TYPE_MEM
		p.To.Reg = v.Args[0].Reg()
		gc.AddAux(&p.To, v)
	case ssa.OpRISCV64SEQZ, ssa.OpRISCV64SNEZ:
		p := s.Prog(v.Op.Asm())
		p.From.Type = obj.TYPE_REG
		p.From.Reg = v.Args[0].Reg()
		p.To.Type = obj.TYPE_REG
		p.To.Reg = v.Reg()
	case ssa.OpRISCV64CALLstatic, ssa.OpRISCV64CALLclosure, ssa.OpRISCV64CALLinter:
		s.Call(v)
	case ssa.OpRISCV64LoweredZero:
		mov, sz := largestMove(v.AuxInt)

		//	mov	ZERO, (Rarg0)
		//	ADD	$sz, Rarg0
		//	BGEU	Rarg1, Rarg0, -2(PC)

		p := s.Prog(mov)
		p.From.Type = obj.TYPE_REG
		p.From.Reg = riscv64.REG_ZERO
		p.To.Type = obj.TYPE_MEM
		p.To.Reg = v.Args[0].Reg()

		p2 := s.Prog(riscv64.AADD)
		p2.From.Type = obj.TYPE_CONST
		p2.From.Offset = sz
		p2.To.Type = obj.TYPE_REG
		p2.To.Reg = v.Args[0].Reg()

		p3 := s.Prog(riscv64.ABGEU)
		p3.To.Type = obj.TYPE_BRANCH
		p3.Reg = v.Args[0].Reg()
		p3.From.Type = obj.TYPE_REG
		p3.From.Reg = v.Args[1].Reg()
		gc.Patch(p3, p)

	case ssa.OpRISCV64LoweredMove:
		mov, sz := largestMove(v.AuxInt)

		//	mov	(Rarg1), T2
		//	mov	T2, (Rarg0)
		//	ADD	$sz, Rarg0
		//	ADD	$sz, Rarg1
		//	BGEU	Rarg2, Rarg0, -4(PC)

		p := s.Prog(mov)
		p.From.Type = obj.TYPE_MEM
		p.From.Reg = v.Args[1].Reg()
		p.To.Type = obj.TYPE_REG
		p.To.Reg = riscv64.REG_T2

		p2 := s.Prog(mov)
		p2.From.Type = obj.TYPE_REG
		p2.From.Reg = riscv64.REG_T2
		p2.To.Type = obj.TYPE_MEM
		p2.To.Reg = v.Args[0].Reg()

		p3 := s.Prog(riscv64.AADD)
		p3.From.Type = obj.TYPE_CONST
		p3.From.Offset = sz
		p3.To.Type = obj.TYPE_REG
		p3.To.Reg = v.Args[0].Reg()

		p4 := s.Prog(riscv64.AADD)
		p4.From.Type = obj.TYPE_CONST
		p4.From.Offset = sz
		p4.To.Type = obj.TYPE_REG
		p4.To.Reg = v.Args[1].Reg()

		p5 := s.Prog(riscv64.ABGEU)
		p5.To.Type = obj.TYPE_BRANCH
		p5.Reg = v.Args[1].Reg()
		p5.From.Type = obj.TYPE_REG
		p5.From.Reg = v.Args[2].Reg()
		gc.Patch(p5, p)

	case ssa.OpRISCV64LoweredNilCheck:
		// Issue a load which will fault if arg is nil.
		// TODO: optimizations. See arm and amd64 LoweredNilCheck.
		p := s.Prog(riscv64.AMOVB)
		p.From.Type = obj.TYPE_MEM
		p.From.Reg = v.Args[0].Reg()
		gc.AddAux(&p.From, v)
		p.To.Type = obj.TYPE_REG
		p.To.Reg = riscv64.REG_ZERO
		if gc.Debug_checknil != 0 && v.Pos.Line() > 1 { // v.Pos == 1 in generated wrappers
			gc.Warnl(v.Pos, "generated nil check")
		}
	case ssa.OpRISCV64LoweredGetClosurePtr:
		// Closure pointer is S4 (riscv64.REG_CTXT).
		gc.CheckLoweredGetClosurePtr(v)
	case ssa.OpRISCV64LoweredGetCallerSP:
		// caller's SP is FixedFrameSize below the address of the first arg
		p := s.Prog(riscv64.AMOV)
		p.From.Type = obj.TYPE_ADDR
		p.From.Offset = -gc.Ctxt.FixedFrameSize()
		p.From.Name = obj.NAME_PARAM
		p.To.Type = obj.TYPE_REG
		p.To.Reg = v.Reg()
	case ssa.OpRISCV64LoweredGetCallerPC:
		p := s.Prog(obj.AGETCALLERPC)
		p.To.Type = obj.TYPE_REG
		p.To.Reg = v.Reg()
	default:
		v.Fatalf("Unhandled op %v", v.Op)
	}
}

func ssaGenBlock(s *gc.SSAGenState, b, next *ssa.Block) {
	s.SetPos(b.Pos)

	switch b.Kind {
	case ssa.BlockDefer:
		// defer returns in A0:
		// 0 if we should continue executing
		// 1 if we should jump to deferreturn call
		p := s.Prog(riscv64.ABNE)
		p.To.Type = obj.TYPE_BRANCH
		p.From.Type = obj.TYPE_REG
		p.From.Reg = riscv64.REG_ZERO
		p.Reg = riscv64.REG_A0
		s.Branches = append(s.Branches, gc.Branch{P: p, B: b.Succs[1].Block()})
		if b.Succs[0].Block() != next {
			p := s.Prog(obj.AJMP)
			p.To.Type = obj.TYPE_BRANCH
			s.Branches = append(s.Branches, gc.Branch{P: p, B: b.Succs[0].Block()})
		}
	case ssa.BlockPlain:
		if b.Succs[0].Block() != next {
			p := s.Prog(obj.AJMP)
			p.To.Type = obj.TYPE_BRANCH
			s.Branches = append(s.Branches, gc.Branch{P: p, B: b.Succs[0].Block()})
		}
	case ssa.BlockExit:
		s.Prog(obj.AUNDEF)
	case ssa.BlockRet:
		s.Prog(obj.ARET)
	case ssa.BlockRetJmp:
		p := s.Prog(obj.AJMP)
		p.To.Type = obj.TYPE_MEM
		p.To.Name = obj.NAME_EXTERN
		p.To.Sym = b.Aux.(*obj.LSym)
	case ssa.BlockRISCV64BNE:
		// Conditional branch if Control != 0.
		p := s.Prog(riscv64.ABNE)
		p.To.Type = obj.TYPE_BRANCH
		p.Reg = b.Control.Reg()
		p.From.Type = obj.TYPE_REG
		p.From.Reg = riscv64.REG_ZERO
		switch next {
		case b.Succs[0].Block():
			p.As = riscv64.InvertBranch(p.As)
			s.Branches = append(s.Branches, gc.Branch{P: p, B: b.Succs[1].Block()})
		case b.Succs[1].Block():
			s.Branches = append(s.Branches, gc.Branch{P: p, B: b.Succs[0].Block()})
		default:
			s.Branches = append(s.Branches, gc.Branch{P: p, B: b.Succs[0].Block()})
			q := s.Prog(obj.AJMP)
			q.To.Type = obj.TYPE_BRANCH
			s.Branches = append(s.Branches, gc.Branch{P: q, B: b.Succs[1].Block()})
		}

	default:
		b.Fatalf("Unhandled kind %v", b.Kind)
	}
}
