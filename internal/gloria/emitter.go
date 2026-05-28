package gloria

import (
	"errors"
	"strconv"
)

type FunctionProto struct {
	Name   string
	Offset int
	Args   []string
}

type Relocation struct {
	CallInstructionOffset int
	TargetFuncName        string
}

type FuncAST struct {
	name       string
	args       []string
	bodyTokens []Token
}

func Emit(src string) ([]byte, error) {
	l := NewLexer(src)

	funcTable := make(map[string]FunctionProto)
	var funcDecls []FuncAST

	for {
		tok := l.NextToken()
		if tok.Type == EOF {
			break
		}

		if tok.Type == FN {
			nameTok := l.NextToken()
			if nameTok.Type != IDENT {
				return nil, errors.New("expected function name")
			}
			name := nameTok.Literal(src)

			if l.NextToken().Type != LPAREN {
				return nil, errors.New("expected (")
			}

			var args []string
			for {
				t := l.NextToken()
				if t.Type == RPAREN {
					break
				}
				if t.Type == IDENT {
					args = append(args, t.Literal(src))
					next := l.NextToken()
					if next.Type == RPAREN {
						break
					}
					if next.Type == COMMA || next.Literal(src) == "," {
						continue
					}
				}
			}

			if l.NextToken().Type != LBRACE {
				return nil, errors.New("expected {")
			}

			var body []Token
			braceCount := 1
			for {
				t := l.NextToken()
				if t.Type == EOF {
					return nil, errors.New("unexpected EOF inside function body")
				}
				if t.Type == LBRACE {
					braceCount++
				}
				if t.Type == RBRACE {
					braceCount--
					if braceCount == 0 {
						break
					}
				}
				body = append(body, t)
			}

			funcDecls = append(funcDecls, FuncAST{
				name:       name,
				args:       args,
				bodyTokens: body,
			})
		}
	}

	out := make([]byte, 0, 512)

	out = append(out, 0xE9, 0x00, 0x00, 0x00, 0x00)

	var relocations []Relocation

	for _, decl := range funcDecls {
		proto := FunctionProto{
			Name:   decl.name,
			Offset: len(out),
			Args:   decl.args,
		}
		funcTable[decl.name] = proto

		funcBytes, funcRelocs, err := CompileFunc(decl, src, funcTable)
		if err != nil {
			return nil, err
		}

		for _, r := range funcRelocs {
			r.CallInstructionOffset += proto.Offset
			relocations = append(relocations, r)
		}

		out = append(out, funcBytes...)
	}

	mainFunc, ok := funcTable["main"]
	if !ok {
		return nil, errors.New("gloria: main function not found")
	}
	jmpTarget := int32(mainFunc.Offset - 5)
	out[1] = byte(jmpTarget & 0xFF)
	out[2] = byte((jmpTarget >> 8) & 0xFF)
	out[3] = byte((jmpTarget >> 16) & 0xFF)
	out[4] = byte((jmpTarget >> 24) & 0xFF)

	for _, reloc := range relocations {
		target, ok := funcTable[reloc.TargetFuncName]
		if !ok {
			return nil, errors.New("gloria: undefined function call: " + reloc.TargetFuncName)
		}

		callTarget := int32(target.Offset - (reloc.CallInstructionOffset + 5))

		out[reloc.CallInstructionOffset+1] = byte(callTarget & 0xFF)
		out[reloc.CallInstructionOffset+2] = byte((callTarget >> 8) & 0xFF)
		out[reloc.CallInstructionOffset+3] = byte((callTarget >> 16) & 0xFF)
		out[reloc.CallInstructionOffset+4] = byte((callTarget >> 24) & 0xFF)
	}

	return out, nil
}

func CompileFunc(f FuncAST, src string, funcTable map[string]FunctionProto) ([]byte, []Relocation, error) {
	state := newCompilerState()
	out := make([]byte, 0, 128)
	var relocs []Relocation

	var err error
	out, err = EmitPrologue(out, state, f.args)
	if err != nil {
		return nil, nil, err
	}

	tokIdx := 0
	nextToken := func() Token {
		if tokIdx >= len(f.bodyTokens) {
			return Token{Type: EOF}
		}
		t := f.bodyTokens[tokIdx]
		tokIdx++
		return t
	}

	peekToken := func() Token {
		if tokIdx >= len(f.bodyTokens) {
			return Token{Type: EOF}
		}
		return f.bodyTokens[tokIdx]
	}

	for {
		t := nextToken()
		if t.Type == EOF {
			break
		}

		if t.Type == LET {
			nameTok := nextToken()
			if nameTok.Type != IDENT {
				return nil, nil, errors.New("expected variable name after let")
			}
			name := nameTok.Literal(src)

			if nextToken().Type != ASSIGN {
				return nil, nil, errors.New("expected '=' after variable name")
			}

			offset, err := state.declareAndAlloc(name)
			if err != nil {
				return nil, nil, err
			}

			rhs := nextToken()
			if rhs.Type == INT {
				v, _ := strconv.ParseUint(rhs.Literal(src), 10, 64)
				out = emitMovImm64ToReg(out, 0, v)
				out = emitMovRegToStack(out, 0, offset)
			} else {
				return nil, nil, errors.New("let only supports immediate integers on RHS for now")
			}
			continue
		}

		if t.Type == IF {
			lhsTok := nextToken()
			if lhsTok.Type != IDENT {
				return nil, nil, errors.New("expected variable on LHS of 'if'")
			}
			lOffset, err := state.getStackOffset(lhsTok.Literal(src))
			if err != nil {
				return nil, nil, err
			}

			opTok := nextToken()
			if opTok.Type != LT && opTok.Type != GT && opTok.Type != EQ {
				return nil, nil, errors.New("expected comparison operator (<, >, ==) in 'if'")
			}

			rhsTok := nextToken()
			if rhsTok.Type != IDENT {
				return nil, nil, errors.New("expected variable on RHS of 'if'")
			}
			rOffset, err := state.getStackOffset(rhsTok.Literal(src))
			if err != nil {
				return nil, nil, err
			}

			if nextToken().Type != LBRACE {
				return nil, nil, errors.New("expected '{' after 'if' condition")
			}

			out = emitMovStackToReg(out, 0, lOffset)
			out = emitMovStackToReg(out, 1, rOffset)
			out = emitCmpRegToReg(out, 1, 0)

			var jmpOp byte
			switch opTok.Type {
			case LT:
				jmpOp = 0x7D
			case GT:
				jmpOp = 0x7E
			case EQ:
				jmpOp = 0x75
			}

			patchIdx, updatedOut := emitCondJmp(out, jmpOp)
			out = updatedOut

			bodyStart := len(out)

			for {
				bodyTok := nextToken()
				if bodyTok.Type == RBRACE || bodyTok.Type == EOF {
					break
				}

				if bodyTok.Type == IDENT && bodyTok.Literal(src) == "print" {
					if nextToken().Type != LPAREN {
						return nil, nil, errors.New("expected '('")
					}
					strTok := nextToken()
					if strTok.Type != STRING {
						return nil, nil, errors.New("print expects string")
					}
					if nextToken().Type != RPAREN {
						return nil, nil, errors.New("expected ')'")
					}
					out = emitLowLevelPrint(out, strTok.Literal(src))
					continue
				}

				if bodyTok.Type == IDENT {
					name := bodyTok.Literal(src)
					offset, err := state.getStackOffset(name)
					if err != nil {
						return nil, nil, err
					}
					assignOp := nextToken()
					if assignOp.Type == ASSIGN {
						rhs := nextToken()
						if rhs.Type == INT {
							v, _ := strconv.ParseUint(rhs.Literal(src), 10, 64)
							out = emitMovImm64ToReg(out, 0, v)
							out = emitMovRegToStack(out, 0, offset)
						}
					} else if assignOp.Type == PLUS_ASSIGN || assignOp.Type == MINUS_ASSIGN {
						rhs := nextToken()
						if rhs.Type == INT {
							v, _ := strconv.ParseUint(rhs.Literal(src), 10, 64)
							out = emitMovStackToReg(out, 0, offset)
							if assignOp.Type == PLUS_ASSIGN {
								out = emitAddImm64ToReg(out, 0, v)
							} else {
								out = emitSubImm64ToReg(out, 0, v)
							}
							out = emitMovRegToStack(out, 0, offset)
						}
					}
					continue
				}
			}

			bodyLen := len(out) - bodyStart
			if bodyLen > 127 {
				return nil, nil, errors.New("body of 'if' is too large for short jump (max 127 bytes)")
			}

			out[patchIdx] = byte(bodyLen)
			continue
		}

		if t.Type == IDENT && t.Literal(src) == "print" {
			if nextToken().Type != LPAREN {
				return nil, nil, errors.New("expected '(' after print")
			}
			strTok := nextToken()
			if strTok.Type != STRING {
				return nil, nil, errors.New("print only supports raw strings for now")
			}
			if nextToken().Type != RPAREN {
				return nil, nil, errors.New("expected ')' after print string")
			}

			strVal := strTok.Literal(src)
			out = emitLowLevelPrint(out, strVal)
			continue
		}

		if t.Type == IDENT {
			name := t.Literal(src)
			offset, err := state.getStackOffset(name)
			if err != nil {
				return nil, nil, err
			}

			op := nextToken()
			if op.Type == ASSIGN {
				rhs := nextToken()
				if rhs.Type == INT {
					v, _ := strconv.ParseUint(rhs.Literal(src), 10, 64)
					out = emitMovImm64ToReg(out, 0, v)
					out = emitMovRegToStack(out, 0, offset)
				} else if rhs.Type == IDENT {
					rOffset, err := state.getStackOffset(rhs.Literal(src))
					if err != nil {
						return nil, nil, err
					}
					out = emitMovStackToReg(out, 0, rOffset)
					out = emitMovRegToStack(out, 0, offset)
				}
			} else if op.Type == PLUS_ASSIGN || op.Type == MINUS_ASSIGN {
				rhs := nextToken()
				if rhs.Type == INT {
					v, _ := strconv.ParseUint(rhs.Literal(src), 10, 64)
					out = emitMovStackToReg(out, 0, offset)
					if op.Type == PLUS_ASSIGN {
						out = emitAddImm64ToReg(out, 0, v)
					} else {
						out = emitSubImm64ToReg(out, 0, v)
					}
					out = emitMovRegToStack(out, 0, offset)
				} else if rhs.Type == IDENT {
					rOffset, err := state.getStackOffset(rhs.Literal(src))
					if err != nil {
						return nil, nil, err
					}
					out = emitMovStackToReg(out, 0, offset)
					out = emitMovStackToReg(out, 1, rOffset)
					if op.Type == PLUS_ASSIGN {
						out = emitAddRegToReg(out, 1, 0)
					} else {
						out = emitSubRegToReg(out, 1, 0)
					}
					out = emitMovRegToStack(out, 0, offset)
				}
			}
			continue
		}

		if t.Type == RETURN {
			nxt := nextToken()

			if nxt.Type == IDENT && peekToken().Type == LPAREN {
				calledFuncName := nxt.Literal(src)

				nextToken()

				var callArgs []string
				for {
					argTok := nextToken()
					if argTok.Type == RPAREN {
						break
					}
					if argTok.Type == IDENT {
						callArgs = append(callArgs, argTok.Literal(src))
						next := nextToken()
						if next.Type == RPAREN {
							break
						}
						if next.Type == COMMA || next.Literal(src) == "," {
							continue
						}
					}
				}

				for i, argName := range callArgs {
					offset, err := state.getStackOffset(argName)
					if err != nil {
						return nil, nil, err
					}
					out = emitMovStackToReg(out, abiArgRegs[i], offset)
				}

				callInstOffset := len(out)
				out = append(out, 0xE8, 0x00, 0x00, 0x00, 0x00)

				relocs = append(relocs, Relocation{
					CallInstructionOffset: callInstOffset,
					TargetFuncName:        calledFuncName,
				})

				out = EmitEpilogue(out)
				return peephole(out), relocs, nil
			}

			if nxt.Type == IDENT && (peekToken().Type == PLUS || peekToken().Type == MINUS) {
				lhsName := nxt.Literal(src)
				opTok := nextToken()

				rhsTok := nextToken()
				if rhsTok.Type != IDENT && rhsTok.Type != INT {
					return nil, nil, errors.New("expected variable or integer after operator in return")
				}

				lOffset, err := state.getStackOffset(lhsName)
				if err != nil {
					return nil, nil, err
				}
				out = emitMovStackToReg(out, 0, lOffset)

				if rhsTok.Type == IDENT {
					rOffset, err := state.getStackOffset(rhsTok.Literal(src))
					if err != nil {
						return nil, nil, err
					}
					out = emitMovStackToReg(out, 1, rOffset)

					if opTok.Type == PLUS {
						out = emitAddRegToReg(out, 1, 0)
					} else {
						out = emitSubRegToReg(out, 1, 0)
					}
				} else if rhsTok.Type == INT {
					v, _ := strconv.ParseUint(rhsTok.Literal(src), 10, 64)
					if opTok.Type == PLUS {
						out = emitAddImm64ToReg(out, 0, v)
					} else {
						out = emitSubImm64ToReg(out, 0, v)
					}
				}

				out = EmitEpilogue(out)
				return peephole(out), relocs, nil
			}

			if nxt.Type == IDENT {
				offset, err := state.getStackOffset(nxt.Literal(src))
				if err != nil {
					return nil, nil, err
				}
				out = emitMovStackToReg(out, 0, offset)
			} else if nxt.Type == INT {
				v, _ := strconv.ParseUint(nxt.Literal(src), 10, 64)
				out = emitMovImm64ToReg(out, 0, v)
			}

			out = EmitEpilogue(out)
			return peephole(out), relocs, nil
		}
	}

	out = EmitEpilogue(out)
	return peephole(out), relocs, nil
}
