// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package riscv64

const (
	MaxAlign  = 32 // max data alignment
	MinAlign  = 1
	FuncAlign = 8
)

// Used by ../internal/ld/dwarf.go
const (
	// From GNU GCC REGISTER_NAMES
	// https://github.com/riscv/riscv-gnu-toolchain/blob/master/gcc/gcc/config/riscv/riscv.h#L858
	DWARFREGSP = 2
	DWARFREGLR = 1
)
