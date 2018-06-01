// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package riscv64

import (
	"cmd/internal/obj/riscv64"
	"cmd/internal/objabi"
	"cmd/internal/sys"
	"cmd/link/internal/ld"
	"cmd/link/internal/sym"
	"fmt"
	"log"
)

func adddynrel(ctxt *ld.Link, s *sym.Symbol, r *sym.Reloc) bool {
	log.Fatalf("adddynrel not implemented")
	return false
}

func archreloc(ctxt *ld.Link, r *sym.Reloc, s *sym.Symbol, val *int64) bool {
	switch r.Type {
	case objabi.R_CALLRISCV:
		// Nothing to do.
		return true
	case objabi.R_RISCV_PCREL_ITYPE, objabi.R_RISCV_PCREL_STYPE:
		pc := s.Value + int64(r.Off)
		off := ld.Symaddr(r.Sym) + r.Add - pc

		// Generate
		// AUIPC
		// and
		// second
		// instruction
		// immediates.
		low, high, err := riscv64.Split32BitImmediate(off)
		if err != nil {
			ld.Errorf(s, "R_RISCV_PCREL_ relocation does not fit in 32-bits: %d", off)
		}

		auipcImm, err := riscv64.EncodeUImmediate(high)
		if err != nil {
			ld.Errorf(s, "cannot encode R_RISCV_PCREL_ AUIPC relocation offset for %s: %v", r.Sym.Name, err)
		}

		var secondImm, secondImmMask int64
		switch r.Type {
		case objabi.R_RISCV_PCREL_ITYPE:
			secondImmMask = riscv64.ITypeImmMask
			secondImm, err = riscv64.EncodeIImmediate(low)
			if err != nil {
				ld.Errorf(s, "cannot encode R_RISCV_PCREL_ITYPE I-type instruction relocation offset for %s: %v", r.Sym.Name, err)
			}
		case objabi.R_RISCV_PCREL_STYPE:
			secondImmMask = riscv64.STypeImmMask
			secondImm, err = riscv64.EncodeSImmediate(low)
			if err != nil {
				ld.Errorf(s, "cannot encode R_RISCV_PCREL_STYPE S-type instruction relocation offset for %s: %v", r.Sym.Name, err)
			}
		default:
			log.Fatalf("Unknown relocation type: %v", r.Type)
		}

		auipc := int64(uint32(*val))
		second := int64(uint32(*val >> 32))

		auipc = (auipc &^ riscv64.UTypeImmMask) | int64(uint32(auipcImm))
		second = (second &^ secondImmMask) | int64(uint32(secondImm))

		*val = second<<32 | auipc
		return true
	}

	return false
}

func archrelocvariant(ctxt *ld.Link, r *sym.Reloc, s *sym.Symbol, t int64) int64 {
	log.Fatalf("unexpected relocation variant")
	return -1
}

// TODO(mpratt): Refactor asmb for other archs. This worked literally copied
// and pasted from mips64.
func asmb(ctxt *ld.Link) {
	if ctxt.Debugvlog != 0 {
		ctxt.Logf("%5.2f asmb\n", ld.Cputime())
	}

	if ctxt.IsELF {
		ld.Asmbelfsetup()
	}

	sect := ld.Segtext.Sections[0]
	ctxt.Out.SeekSet(int64(sect.Vaddr - ld.Segtext.Vaddr + ld.Segtext.Fileoff))
	ld.Codeblk(ctxt, int64(sect.Vaddr), int64(sect.Length))
	for _, sect = range ld.Segtext.Sections[1:] {
		ctxt.Out.SeekSet(int64(sect.Vaddr - ld.Segtext.Vaddr + ld.Segtext.Fileoff))
		ld.Datblk(ctxt, int64(sect.Vaddr), int64(sect.Length))
	}

	if ld.Segrodata.Filelen > 0 {
		if ctxt.Debugvlog != 0 {
			ctxt.Logf("%5.2f rodatblk\n", ld.Cputime())
		}

		ctxt.Out.SeekSet(int64(ld.Segrodata.Fileoff))
		ld.Datblk(ctxt, int64(ld.Segrodata.Vaddr), int64(ld.Segrodata.Filelen))
	}

	if ctxt.Debugvlog != 0 {
		ctxt.Logf("%5.2f datblk\n", ld.Cputime())
	}

	ctxt.Out.SeekSet(int64(ld.Segdata.Fileoff))
	ld.Datblk(ctxt, int64(ld.Segdata.Vaddr), int64(ld.Segdata.Filelen))

	ctxt.Out.SeekSet(int64(ld.Segdwarf.Fileoff))
	ld.Dwarfblk(ctxt, int64(ld.Segdwarf.Vaddr), int64(ld.Segdwarf.Filelen))

	/* output symbol table */
	ld.Symsize = 0

	ld.Lcsize = 0
	symo := uint32(0)
	if !*ld.FlagS {
		if !ctxt.IsELF {
			ld.Errorf(nil, "unsupported executable format")
		}
		if ctxt.Debugvlog != 0 {
			ctxt.Logf("%5.2f sym\n", ld.Cputime())
		}
		symo = uint32(ld.Segdwarf.Fileoff + ld.Segdwarf.Filelen)
		symo = uint32(ld.Rnd(int64(symo), int64(*ld.FlagRound)))

		ctxt.Out.SeekSet(int64(symo))
		if ctxt.Debugvlog != 0 {
			ctxt.Logf("%5.2f elfsym\n", ld.Cputime())
		}
		ld.Asmelfsym(ctxt)
		ctxt.Out.Flush()
		ctxt.Out.Write(ld.Elfstrdat)

		if ctxt.Debugvlog != 0 {
			ctxt.Logf("%5.2f dwarf\n", ld.Cputime())
		}

		if ctxt.LinkMode == ld.LinkExternal {
			ld.Elfemitreloc(ctxt)
		}
	}

	if ctxt.Debugvlog != 0 {
		ctxt.Logf("%5.2f header\n", ld.Cputime())
	}

	ctxt.Out.SeekSet(0)
	switch ctxt.HeadType {
	default:
		ld.Errorf(nil, "unsupported operating system")
	case objabi.Hlinux:
		ld.Asmbelf(ctxt, int64(symo))
	}

	ctxt.Out.Flush()
	if *ld.FlagC {
		fmt.Printf("textsize=%d\n", ld.Segtext.Filelen)
		fmt.Printf("datsize=%d\n", ld.Segdata.Filelen)
		fmt.Printf("bsssize=%d\n", ld.Segdata.Length-ld.Segdata.Filelen)
		fmt.Printf("symsize=%d\n", ld.Symsize)
		fmt.Printf("lcsize=%d\n", ld.Lcsize)
		fmt.Printf("total=%d\n", ld.Segtext.Filelen+ld.Segdata.Length+uint64(ld.Symsize)+uint64(ld.Lcsize))
	}
}

func elfreloc1(ctxt *ld.Link, r *sym.Reloc, sectoff int64) bool {
	log.Fatalf("elfreloc1 not implemented")
	return false
}

func elfsetupplt(ctxt *ld.Link) {
	return
}

func gentext(ctxt *ld.Link) {
	return
}

func machoreloc1(arch *sys.Arch, out *ld.OutBuf, s *sym.Symbol, r *sym.Reloc, sectoff int64) bool {
	return false
}
