/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

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

func parseBuiltinArgs(nextToken func() Token, src string) ([]string, error) {
	var args []string
	for {
		argTok := nextToken()
		if argTok.Type == RPAREN || argTok.Type == EOF {
			break
		}
		if argTok.Type != IDENT && argTok.Type != INT {
			return nil, errors.New("expected identifier or integer inside call")
		}
		args = append(args, argTok.Literal(src))
		next := nextToken()
		if next.Type == RPAREN {
			break
		}
		if next.Type == COMMA || next.Literal(src) == "," {
			continue
		}
		return nil, errors.New("expected ',' or ')' after argument")
	}
	return args, nil
}

func emitBuiltinCall(out []byte, name string, args []string, state *compilerState) ([]byte, error) {
	switch name {
	case "out8":
		if len(args) != 2 {
			return nil, errors.New("out8 expects exactly 2 arguments")
		}
		for i, arg := range args {
			if v, err := strconv.ParseUint(arg, 10, 64); err == nil {
				if i == 0 {
					if v > 0xFFFF {
						return nil, errors.New("out8 port immediate out of range")
					}
					out = emitMovImm16ToReg(out, 2, uint16(v))
				} else {
					if v > 0xFF {
						return nil, errors.New("out8 data immediate out of range")
					}
					out = emitMovImm8ToReg(out, 0, byte(v))
				}
			} else {
				offset, err := state.getStackOffset(arg)
				if err != nil {
					return nil, err
				}
				if i == 0 {
					out = emitMovStackToReg(out, 2, offset)
				} else {
					out = emitMovStackToReg(out, 0, offset)
				}
			}
		}
		out = append(out, 0xEE)
		return out, nil
	case "in8":
		if len(args) != 1 {
			return nil, errors.New("in8 expects exactly 1 argument")
		}
		if v, err := strconv.ParseUint(args[0], 10, 64); err == nil {
			if v > 0xFFFF {
				return nil, errors.New("in8 port immediate out of range")
			}
			out = emitMovImm16ToReg(out, 2, uint16(v))
		} else {
			offset, err := state.getStackOffset(args[0])
			if err != nil {
				return nil, err
			}
			out = emitMovStackToReg(out, 2, offset)
		}
		out = append(out, 0xEC)
		out = append(out, 0x48, 0x0F, 0xB6, 0xC0)
		return out, nil
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
			} else if rhs.Type == IDENT && rhs.Literal(src) == "peek" {
				if nextToken().Type != LPAREN {
					return nil, nil, errors.New("expected '(' after peek")
				}

				addrTok := nextToken()
				if addrTok.Type != IDENT && addrTok.Type != INT {
					return nil, nil, errors.New("expected address inside peek(...)")
				}

				if nextToken().Type != RPAREN {
					return nil, nil, errors.New("expected ')' after peek address")
				}

				if v, err := strconv.ParseUint(addrTok.Literal(src), 10, 64); err == nil {
					out = emitMovImm64ToReg(out, abiArgRegs[0], v)
				} else {
					addrOffset, err := state.getStackOffset(addrTok.Literal(src))
					if err != nil {
						return nil, nil, err
					}
					out = emitMovStackToReg(out, abiArgRegs[0], addrOffset)
				}

				out = append(out, 0x0F, 0xB7, 0x07)

				out = emitMovRegToStack(out, 0, offset)
			} else if rhs.Type == IDENT && rhs.Literal(src) == "in8" {
				if nextToken().Type != LPAREN {
					return nil, nil, errors.New("expected '(' after in8")
				}

				args, err := parseBuiltinArgs(nextToken, src)
				if err != nil {
					return nil, nil, err
				}

				out, err = emitBuiltinCall(out, "in8", args, state)
				if err != nil {
					return nil, nil, err
				}
				out = emitMovRegToStack(out, 0, offset)
			} else {
				return nil, nil, errors.New("let only supports immediate integers or peek() on RHS")
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
						return nil, nil, errors.New("expected '(' after print")
					}
					argTok := nextToken()
					if argTok.Type == RPAREN {
						return nil, nil, errors.New("print expects an argument")
					}

					if argTok.Type == STRING {
						strVal := argTok.Literal(src)
						if nextToken().Type != RPAREN {
							return nil, nil, errors.New("expected ')' after print string")
						}
						out = emitBareMetalPrint(out, strVal)
					} else if argTok.Type == INT {
						numVal := argTok.Literal(src)
						if nextToken().Type != RPAREN {
							return nil, nil, errors.New("expected ')' after print number")
						}
						var err error
						out, err = emitNumberPrint(out, numVal)
						if err != nil {
							return nil, nil, err
						}
					} else if argTok.Type == IDENT {
						regName := argTok.Literal(src)
						if nextToken().Type != RPAREN {
							return nil, nil, errors.New("expected ')' after print register")
						}
						var err error
						out, err = emitRegisterPrint(out, regName)
						if err != nil {
							return nil, nil, err
						}
					} else {
						return nil, nil, errors.New("print expects string, number, or register")
					}
					continue
				}

				if bodyTok.Type == IDENT {
					name := bodyTok.Literal(src)
					if name == "out8" || name == "in8" {
						if nextToken().Type != LPAREN {
							return nil, nil, errors.New("expected '(' after " + name)
						}
						args, err := parseBuiltinArgs(nextToken, src)
						if err != nil {
							return nil, nil, err
						}
						out, err = emitBuiltinCall(out, name, args, state)
						if err != nil {
							return nil, nil, err
						}
						continue
					}
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

		if t.Type == WHILE {
			condTok := nextToken()
			if condTok.Type != IDENT {
				return nil, nil, errors.New("expected variable name after while")
			}

			condOffset, err := state.getStackOffset(condTok.Literal(src))
			if err != nil {
				return nil, nil, err
			}

			if nextToken().Type != LBRACE {
				return nil, nil, errors.New("expected '{' after while condition")
			}

			loopStartOffset := len(out)

			out = emitMovStackToReg(out, 0, condOffset)
			out = emitMovImm64ToReg(out, 1, 0)
			out = emitCmpRegToReg(out, 1, 0)

			jzOffsetInCode := len(out)
			out = append(out, 0x0F, 0x84, 0x00, 0x00, 0x00, 0x00)

			for {
				bodyTok := nextToken()
				if bodyTok.Type == RBRACE || bodyTok.Type == EOF {
					if bodyTok.Type == EOF {
						return nil, nil, errors.New("unclosed '{' in while loop")
					}
					break
				}

				if bodyTok.Type == IDENT && bodyTok.Literal(src) == "print" {
					if nextToken().Type != LPAREN {
						return nil, nil, errors.New("expected '(' after print")
					}
					argTok := nextToken()
					if argTok.Type == RPAREN {
						return nil, nil, errors.New("print expects an argument")
					}

					if argTok.Type == STRING {
						strVal := argTok.Literal(src)
						if nextToken().Type != RPAREN {
							return nil, nil, errors.New("expected ')' after print string")
						}
						out = emitBareMetalPrint(out, strVal)
					} else if argTok.Type == INT {
						numVal := argTok.Literal(src)
						if nextToken().Type != RPAREN {
							return nil, nil, errors.New("expected ')' after print number")
						}
						var err error
						out, err = emitNumberPrint(out, numVal)
						if err != nil {
							return nil, nil, err
						}
					} else if argTok.Type == IDENT {
						regName := argTok.Literal(src)
						if nextToken().Type != RPAREN {
							return nil, nil, errors.New("expected ')' after print register")
						}
						var err error
						out, err = emitRegisterPrint(out, regName)
						if err != nil {
							return nil, nil, err
						}
					} else {
						return nil, nil, errors.New("print expects string, number, or register")
					}
					continue
				}

				if bodyTok.Type == IDENT {
					name := bodyTok.Literal(src)
					if name == "out8" || name == "in8" {
						if nextToken().Type != LPAREN {
							return nil, nil, errors.New("expected '(' after " + name)
						}
						args, err := parseBuiltinArgs(nextToken, src)
						if err != nil {
							return nil, nil, err
						}
						out, err = emitBuiltinCall(out, name, args, state)
						if err != nil {
							return nil, nil, err
						}
						continue
					}

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
						} else if rhs.Type == IDENT && rhs.Literal(src) == "in8" {
							if nextToken().Type != LPAREN {
								return nil, nil, errors.New("expected '(' after in8")
							}
							args, err := parseBuiltinArgs(nextToken, src)
							if err != nil {
								return nil, nil, err
							}
							out, err = emitBuiltinCall(out, "in8", args, state)
							if err != nil {
								return nil, nil, err
							}
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

			out = append(out, 0xE9)
			jmpAddrOffset := len(out)
			out = append(out, 0x00, 0x00, 0x00, 0x00)

			currentLen := len(out)
			dispBack := int32(loopStartOffset - currentLen)
			out[jmpAddrOffset] = byte(dispBack)
			out[jmpAddrOffset+1] = byte(dispBack >> 8)
			out[jmpAddrOffset+2] = byte(dispBack >> 16)
			out[jmpAddrOffset+3] = byte(dispBack >> 24)

			loopEndOffset := len(out)
			dispForward := int32(loopEndOffset - (jzOffsetInCode + 6))
			out[jzOffsetInCode+2] = byte(dispForward)
			out[jzOffsetInCode+3] = byte(dispForward >> 8)
			out[jzOffsetInCode+4] = byte(dispForward >> 16)
			out[jzOffsetInCode+5] = byte(dispForward >> 24)

			continue
		}

		if t.Type == IDENT && t.Literal(src) == "print" {
			if nextToken().Type != LPAREN {
				return nil, nil, errors.New("expected '(' after print")
			}
			argTok := nextToken()
			if argTok.Type == RPAREN {
				return nil, nil, errors.New("print expects an argument")
			}

			if argTok.Type == STRING {
				strVal := argTok.Literal(src)
				if nextToken().Type != RPAREN {
					return nil, nil, errors.New("expected ')' after print string")
				}
				out = emitBareMetalPrint(out, strVal)
			} else if argTok.Type == INT {
				numVal := argTok.Literal(src)
				if nextToken().Type != RPAREN {
					return nil, nil, errors.New("expected ')' after print number")
				}
				var err error
				out, err = emitNumberPrint(out, numVal)
				if err != nil {
					return nil, nil, err
				}
			} else if argTok.Type == IDENT {
				regName := argTok.Literal(src)
				if nextToken().Type != RPAREN {
					return nil, nil, errors.New("expected ')' after print register")
				}
				var err error
				out, err = emitRegisterPrint(out, regName)
				if err != nil {
					return nil, nil, err
				}
			} else {
				return nil, nil, errors.New("print expects string, number, or register")
			}
			continue
		}

		if t.Type == IDENT && (t.Literal(src) == "out8" || t.Literal(src) == "in8") {
			name := t.Literal(src)
			if nextToken().Type != LPAREN {
				return nil, nil, errors.New("expected '(' after " + name)
			}
			args, err := parseBuiltinArgs(nextToken, src)
			if err != nil {
				return nil, nil, err
			}
			out, err = emitBuiltinCall(out, name, args, state)
			if err != nil {
				return nil, nil, err
			}
			continue
		}

		if t.Type == IDENT && t.Literal(src) == "poke" {
			if nextToken().Type != LPAREN {
				return nil, nil, errors.New("expected '(' after poke")
			}

			var args []string
			for {
				tok := nextToken()
				if tok.Type == RPAREN || tok.Type == EOF {
					break
				}
				if tok.Type == IDENT || tok.Type == INT {
					args = append(args, tok.Literal(src))
				}
				next := nextToken()
				if next.Type == RPAREN {
					break
				}
				if next.Type == COMMA || next.Literal(src) == "," {
					continue
				}
			}

			if len(args) != 2 {
				return nil, nil, errors.New("poke expects exactly 2 arguments: poke(address, value)")
			}

			for i, arg := range args {
				if v, err := strconv.ParseUint(arg, 10, 64); err == nil {
					out = emitMovImm64ToReg(out, abiArgRegs[i], v)
				} else {
					offset, err := state.getStackOffset(arg)
					if err != nil {
						return nil, nil, err
					}
					out = emitMovStackToReg(out, abiArgRegs[i], offset)
				}
			}

			out = append(out, 0x66, 0x89, 0x37)
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
				} else if rhs.Type == IDENT && rhs.Literal(src) == "in8" {
					if nextToken().Type != LPAREN {
						return nil, nil, errors.New("expected '(' after in8")
					}
					args, err := parseBuiltinArgs(nextToken, src)
					if err != nil {
						return nil, nil, err
					}
					out, err = emitBuiltinCall(out, "in8", args, state)
					if err != nil {
						return nil, nil, err
					}

					out = emitMovRegToStack(out, 0, offset)

				} else if rhs.Type == IDENT && rhs.Literal(src) == "peek" {
					if nextToken().Type != LPAREN {
						return nil, nil, errors.New("expected '(' after peek")
					}

					addrTok := nextToken()

					if v, err := strconv.ParseUint(addrTok.Literal(src), 10, 64); err == nil {
						out = emitMovImm64ToReg(out, abiArgRegs[0], v)
					} else {
						addrOffset, err := state.getStackOffset(addrTok.Literal(src))
						if err != nil {
							return nil, nil, err
						}
						out = emitMovStackToReg(out, abiArgRegs[0], addrOffset)
					}
					out = append(out, 0x0F, 0xB7, 0x07)
					out = emitMovRegToStack(out, 0, offset)
				} else if rhs.Type == IDENT {
					r0ffset, err := state.getStackOffset(rhs.Literal(src))
					if err != nil {
						return nil, nil, err
					}
					out = emitMovRegToStack(out, 0, r0ffset)
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
				var callArgs []string

				if calledFuncName == "out8" || calledFuncName == "in8" {
					nextToken()
					args, err := parseBuiltinArgs(nextToken, src)
					if err != nil {
						return nil, nil, err
					}
					out, err = emitBuiltinCall(out, calledFuncName, args, state)
					if err != nil {
						return nil, nil, err
					}
					out = EmitEpilogue(out)
					return peephole(out), relocs, nil
				}

				if calledFuncName == "poke" {
					nextToken()

					var args []string
					for {
						tok := nextToken()

						if tok.Type == RPAREN || tok.Type == EOF {
							break
						}

						if tok.Type == IDENT || tok.Type == INT {
							args = append(args, tok.Literal(src))
						}
						next := nextToken()
						if next.Type == RPAREN {
							break
						}
						if next.Type == COMMA || next.Literal(src) == "," {
							continue
						}
					}

					if len(args) != 2 {
						return nil, nil, errors.New("poke expects exactly 2 arguments: poke(address, value)")
					}

					for i, arg := range args {
						if v, err := strconv.ParseUint(arg, 10, 64); err == nil {
							out = emitMovImm64ToReg(out, abiArgRegs[i], v)
						} else {
							offset, err := state.getStackOffset(arg)
							if err != nil {
								return nil, nil, err
							}
							out = emitMovStackToReg(out, abiArgRegs[i], offset)
						}
					}

					out = append(out, 0x66, 0x89, 0x37)
					out = EmitEpilogue(out)
					return peephole(out), relocs, nil

				}
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
