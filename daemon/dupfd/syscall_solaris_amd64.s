#include "textflag.h"

TEXT ·fcntl1(SB),NOSPLIT,$0
	JMP	runtime·syscall_fcntl(SB)
