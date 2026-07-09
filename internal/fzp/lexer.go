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

package fzp

import (
	"strconv"
	"strings"
)

const (
	tokNumber = iota
	tokIdent
	tokOp
	tokEOF
)

type token struct {
	typeID int
	value  string
}

type tokenizer struct {
	input string
	pos   int
}

func newTokenizer(input string) *tokenizer {
	return &tokenizer{input: input}
}

func (t *tokenizer) next() token {
	for t.pos < len(t.input) {
		ch := t.input[t.pos]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			t.pos++
			continue
		}
		if ch >= '0' && ch <= '9' {
			start := t.pos
			for t.pos < len(t.input) && t.input[t.pos] >= '0' && t.input[t.pos] <= '9' {
				t.pos++
			}
			return token{typeID: tokNumber, value: t.input[start:t.pos]}
		}
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' {
			start := t.pos
			for t.pos < len(t.input) {
				next := t.input[t.pos]
				if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') || (next >= '0' && next <= '9') || next == '_' {
					t.pos++
					continue
				}
				break
			}
			return token{typeID: tokIdent, value: t.input[start:t.pos]}
		}
		if ch == '(' || ch == ')' || ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '%' {
			val := string(ch)
			t.pos++
			return token{typeID: tokOp, value: val}
		}
		if ch == '!' || ch == '=' || ch == '<' || ch == '>' {
			start := t.pos
			ch2 := byte(0)
			if t.pos+1 < len(t.input) {
				ch2 = t.input[t.pos+1]
			}
			if ch == '!' && ch2 == '=' {
				t.pos += 2
				return token{typeID: tokOp, value: "!="}
			}
			if ch == '=' && ch2 == '=' {
				t.pos += 2
				return token{typeID: tokOp, value: "=="}
			}
			if ch == '<' && ch2 == '=' {
				t.pos += 2
				return token{typeID: tokOp, value: "<="}
			}
			if ch == '>' && ch2 == '=' {
				t.pos += 2
				return token{typeID: tokOp, value: ">="}
			}
			if ch == '&' && ch2 == '&' {
				t.pos += 2
				return token{typeID: tokOp, value: "&&"}
			}
			if ch == '|' && ch2 == '|' {
				t.pos += 2
				return token{typeID: tokOp, value: "||"}
			}
			if ch == '!' {
				t.pos++
				return token{typeID: tokOp, value: "!"}
			}
			if ch == '=' {
				t.pos++
				return token{typeID: tokOp, value: "="}
			}
			if ch == '<' {
				t.pos++
				return token{typeID: tokOp, value: "<"}
			}
			if ch == '>' {
				t.pos++
				return token{typeID: tokOp, value: ">"}
			}
			return token{typeID: tokOp, value: t.input[start:t.pos]}
		}
		t.pos++
	}
	return token{typeID: tokEOF}
}

type parser struct {
	tok   []token
	pos   int
	macros map[string]macro
}

func newParser(input string, macros map[string]macro) *parser {
	tok := make([]token, 0, len(input)/2)
	t := newTokenizer(input)
	for {
		tokValue := t.next()
		if tokValue.typeID == tokEOF {
			break
		}
		tok = append(tok, tokValue)
	}
	return &parser{tok: tok, macros: macros}
}

func (p *parser) parse() int {
	if len(p.tok) == 0 {
		return 0
	}
	value := p.parseOr()
	return value
}

func (p *parser) parseOr() int {
	left := p.parseAnd()
	for p.peekValue("||") {
		p.pos++
		left = boolValue(left != 0 || p.parseAnd() != 0)
	}
	return left
}

func (p *parser) parseAnd() int {
	left := p.parseCompare()
	for p.peekValue("&&") {
		p.pos++
		left = boolValue(left != 0 && p.parseCompare() != 0)
	}
	return left
}

func (p *parser) parseCompare() int {
	left := p.parseAdd()
	for {
		if p.peekValue("==") {
			p.pos++
			left = boolValue(left == p.parseAdd())
			continue
		}
		if p.peekValue("!=") {
			p.pos++
			left = boolValue(left != p.parseAdd())
			continue
		}
		if p.peekValue("<=") {
			p.pos++
			left = boolValue(left <= p.parseAdd())
			continue
		}
		if p.peekValue(">=") {
			p.pos++
			left = boolValue(left >= p.parseAdd())
			continue
		}
		if p.peekValue("<") {
			p.pos++
			left = boolValue(left < p.parseAdd())
			continue
		}
		if p.peekValue(">") {
			p.pos++
			left = boolValue(left > p.parseAdd())
			continue
		}
		break
	}
	return left
}

func (p *parser) parseAdd() int {
	left := p.parseMul()
	for p.peekValue("+") || p.peekValue("-") {
		if p.peekValue("+") {
			p.pos++
			left += p.parseMul()
			continue
		}
		p.pos++
		left -= p.parseMul()
	}
	return left
}

func (p *parser) parseMul() int {
	left := p.parseUnary()
	for p.peekValue("*") || p.peekValue("/") || p.peekValue("%") {
		if p.peekValue("*") {
			p.pos++
			left *= p.parseUnary()
			continue
		}
		if p.peekValue("/") {
			p.pos++
			left /= p.parseUnary()
			continue
		}
		p.pos++
		left %= p.parseUnary()
	}
	return left
}

func (p *parser) parseUnary() int {
	if p.peekValue("!") {
		p.pos++
		return boolValue(p.parseUnary() == 0)
	}
	if p.peekValue("+") {
		p.pos++
		return p.parseUnary()
	}
	if p.peekValue("-") {
		p.pos++
		return -p.parseUnary()
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() int {
	if p.pos >= len(p.tok) {
		return 0
	}
	curr := p.tok[p.pos]
	if curr.typeID == tokNumber {
		p.pos++
		value, _ := strconv.Atoi(curr.value)
		return value
	}
	if curr.typeID == tokIdent {
		p.pos++
		name := curr.value
		if name == "defined" {
			if p.pos < len(p.tok) && p.tok[p.pos].value == "(" {
				p.pos++
				if p.pos < len(p.tok) {
					arg := p.tok[p.pos]
					if arg.typeID == tokIdent {
						p.pos++
						if p.pos < len(p.tok) && p.tok[p.pos].value == ")" {
							p.pos++
							_, ok := p.macros[arg.value]
							return boolValue(ok)
						}
					}
				}
			}
			return 0
		}
		if macro, ok := p.macros[name]; ok {
			if macro.hasArgs {
				return 0
			}
			value, err := strconv.Atoi(strings.TrimSpace(macro.value))
			if err == nil {
				return value
			}
			return boolValue(macro.value != "")
		}
		return 0
	}
	if curr.value == "(" {
		p.pos++
		value := p.parseOr()
		if p.pos < len(p.tok) && p.tok[p.pos].value == ")" {
			p.pos++
		}
		return value
	}
	return 0
}

func (p *parser) peekValue(value string) bool {
	return p.pos < len(p.tok) && p.tok[p.pos].value == value
}

func boolValue(value bool) int {
	if value {
		return 1
	}
	return 0
}
