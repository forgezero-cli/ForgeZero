#include "textflag.h"

TEXT ·buildCacheKey(SB), NOSPLIT, $0-56
    RET

TEXT ·joinPath(SB), NOSPLIT, $0-48
    RET

TEXT ·cacheEntryPath(SB), NOSPLIT, $0-56
    RET

TEXT ·pathBuffer_appendString(SB), NOSPLIT, $0-48
    RET

TEXT ·pathBuffer_appendByte(SB), NOSPLIT, $0-24
    RET

TEXT ·pathBuffer_appendBytes(SB), NOSPLIT, $0-48
    RET

