package gloria

import (
    "encoding/binary"
    "errors"
    "strconv"
)

func Emit(src string) ([]byte, error) {
    l := NewLexer(src)
    tok := l.NextToken()
    if tok.Type != FN {
        return nil, errors.New("expected fn")
    }
    nameTok := l.NextToken()
    if nameTok.Type != IDENT {
        return nil, errors.New("expected function name")
    }
    p := l.NextToken()
    if p.Type != LPAREN {
        return nil, errors.New("expected (")
    }
    paramTok := l.NextToken()
    if paramTok.Type == IDENT {
        if l.NextToken().Type != RPAREN {
            return nil, errors.New("expected )")
        }
    } else if paramTok.Type == RPAREN {
    } else {
        return nil, errors.New("bad param list")
    }

    if l.NextToken().Type != LBRACE {
        return nil, errors.New("expected {")
    }

    out := make([]byte, 0, 256)
    out = append(out, 0x55)
    out = append(out, 0x48, 0x89, 0xE5)

    for {
        t := l.NextToken()
        if t.Type == RBRACE || t.Type == EOF {
            break
        }
        if t.Type == ATREG {
            lhs := t.Lit
            op := l.NextToken()
            if op.Type == ASSIGN {
                rhs := l.NextToken()
                if rhs.Type == INT {
                    v, _ := strconv.ParseUint(rhs.Lit, 10, 64)
                    reg := regCode(lhs)
                    if reg < 8 {
                        out = emitMovImm64ToReg(out, reg, v)
                    }
                } else if rhs.Type == ATREG {
                    rreg := regCode(rhs.Lit)
                    lreg := regCode(lhs)
                    if rreg < 8 && lreg < 8 {
                        out = emitMovRegToReg(out, rreg, lreg)
                    }
                }
            } else if op.Type == PLUS_ASSIGN || op.Type == MINUS_ASSIGN {
                rhs := l.NextToken()
                if rhs.Type == INT {
                    v, _ := strconv.ParseUint(rhs.Lit, 10, 64)
                    reg := regCode(lhs)
                    if op.Type == PLUS_ASSIGN {
                        out = emitAddImm64ToReg(out, reg, v)
                    } else {
                        out = emitSubImm64ToReg(out, reg, v)
                    }
                } else if rhs.Type == ATREG {
                    rreg := regCode(rhs.Lit)
                    lreg := regCode(lhs)
                    if op.Type == PLUS_ASSIGN {
                        out = emitAddRegToReg(out, rreg, lreg)
                    } else {
                        out = emitSubRegToReg(out, rreg, lreg)
                    }
                }
            }
        } else if t.Type == RETURN {
            nxt := l.NextToken()
            if nxt.Type == ATREG {
                if nxt.Lit != "@rax" {
                    r := regCode(nxt.Lit)
                    if r < 8 {
                        out = emitMovRegToReg(out, r, 0)
                    }
                }
            } else if nxt.Type == INT {
                v, _ := strconv.ParseUint(nxt.Lit, 10, 64)
                out = emitMovImm64ToReg(out, 0, v)
            }
            out = append(out, 0x5D, 0xC3)
            return peephole(out), nil
        }
    }

    out = append(out, 0x5D, 0xC3)
    return peephole(out), nil
}

func emitMovImm64ToReg(out []byte, reg int, v uint64) []byte {
    b := make([]byte, 10)
    b[0] = 0x48
    b[1] = 0xB8 + byte(reg)
    binary.LittleEndian.PutUint64(b[2:], v)
    return append(out, b...)
}

func emitMovRegToReg(out []byte, src, dst int) []byte {
    modrm := byte(0xC0 | (src<<3) | dst)
    return append(out, 0x48, 0x89, modrm)
}

func emitAddRegToReg(out []byte, src, dst int) []byte {
    modrm := byte(0xC0 | (src<<3) | dst)
    return append(out, 0x48, 0x01, modrm)
}

func emitSubRegToReg(out []byte, src, dst int) []byte {
    modrm := byte(0xC0 | (src<<3) | dst)
    return append(out, 0x48, 0x29, modrm)
}

func emitAddImm64ToReg(out []byte, reg int, v uint64) []byte {
    return append(out, 0x48, 0x83, byte(0xC0|reg), byte(v&0xFF))
}

func emitSubImm64ToReg(out []byte, reg int, v uint64) []byte {
    return append(out, 0x48, 0x83, byte(0xE8|reg), byte(v&0xFF))
}

func regCode(lit string) int {
    switch lit {
    case "@rax":
        return 0
    case "@rcx":
        return 1
    case "@rdx":
        return 2
    case "@rbx":
        return 3
    case "@rsi":
        return 6
    case "@rdi":
        return 7
    }
    return 0
}

func peephole(ins []byte) []byte {
    out := make([]byte, 0, len(ins))
    i := 0
    for i < len(ins) {
        if i+3 <= len(ins) && ins[i] == 0x48 && ins[i+1] == 0x89 {
            mod := ins[i+2]
            src := (mod >> 3) & 7
            dst := mod & 7
            if src == dst {
                i += 3
                continue
            }
        }
        out = append(out, ins[i])
        i++
    }
    return out
}
