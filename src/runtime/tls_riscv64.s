// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "go_asm.h"
#include "go_tls.h"
#include "funcdata.h"
#include "textflag.h"

// If !iscgo, this is a no-op.
//
// NOTE: mcall() assumes this clobbers only R23 (REGTMP).
TEXT runtime·save_g(SB),NOSPLIT,$-8-0
	MOVB	runtime·iscgo(SB), A0
	BEQ	$0, A0, nocgo

	MOV	g, runtime·tls_g(SB)

nocgo:
	RET

TEXT runtime·load_g(SB),NOSPLIT,$-8-0
	MOVB	runtime·iscgo(SB), A0
	BEQ	$0, A0, nocgo

	MOV	runtime·tls_g(SB), g

nocgo:

	RET
