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
)

func ParseFunctionHeader(l *Lexer, src string) ([]string, error) {
	if l.NextToken().Type != IDENT {
		return nil, errors.New("expected function name")
	}
	if l.NextToken().Type != LPAREN {
		return nil, errors.New("expected (")
	}

	var args []string
	for {
		tok := l.NextToken()
		if tok.Type == RPAREN {
			break
		}
		if tok.Type == IDENT {
			args = append(args, tok.Literal(src))
			next := l.NextToken()
			if next.Type == RPAREN {
				break
			}
			if next.Literal(src) == "," || next.Type == COMMA {
				continue
			}
			if next.Type == IDENT {
				args = append(args, next.Literal(src))
				continue
			}
		} else {
			if tok.Literal(src) == "," || tok.Type == COMMA {
				continue
			}
			return nil, errors.New("expected argument name or ')'")
		}
	}

	if len(args) > 6 {
		return nil, errors.New("gloria supports max 6 arguments for now (system V ABI limits)")
	}

	if l.NextToken().Type != LBRACE {
		return nil, errors.New("expected {")
	}

	return args, nil
}

func EmitPrologue(out []byte, state *compilerState, args []string) ([]byte, error) {
	// push rbp (0x55)
	out = append(out, 0x55)

	// mov rbp, rsp (0x48 0x89 0xE5)
	out = append(out, 0x48, 0x89, 0xE5)

	// sub rsp, 128 (allocate 128 bytes for locals: 0x48 0x81 0xEC 0x80 0x00 0x00 0x00)
	out = append(out, 0x48, 0x81, 0xEC, 0x80, 0x00, 0x00, 0x00)

	for i, argName := range args {
		offset, err := state.declareAndAlloc(argName)
		if err != nil {
			return nil, err
		}
		out = emitMovRegToStack(out, abiArgRegs[i], offset)
	}

	return out, nil
}

func EmitEpilogue(out []byte) []byte {
	// leave (0xC9): equivalent to mov rsp, rbp; pop rbp
	out = append(out, 0xC9)

	// ret (0xC3)
	out = append(out, 0xC3)

	return out
}
