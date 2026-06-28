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

type TokenType byte

const (
	ILLEGAL TokenType = iota
	EOF

	IDENT
	INT
	STRING
	ATREG
	COMMA
	NUMBER // int

	LIT_INT
	CONST_INT

	PLUS         // +
	MINUS        // -
	PLUS_ASSIGN  // +=
	MINUS_ASSIGN // -=
	ASSIGN       // =
	SLASH        // /
	LBRACE       // {
	RBRACE       // }
	LPAREN       // (
	RPAREN       // )
	EQ
	LT
	GT

	AMP
	STAR
	PERCENT

	PIPE
	CARET

	FN
	LET
	RETURN
	REG
	IF
	WHILE
)

func (t TokenType) String() string {
	names := [...]string{
		"ILLEGAL", "EOF", "IDENT", "INT", "STRING", "ATREG", ",",
		"+", "-", "+=", "-=", "=", "/", "{", "}", "(", ")",
		"==", "<", ">",
		"FN", "LET", "RETURN", "REG", "IF", "WHILE",
	}

	if int(t) < len(names) {
		return names[t]
	}

	return "UNKNOWN"
}

type Token struct {
	Type  TokenType
	Start int
	End   int
}

func (t Token) Literal(src string) string {
	if t.Start < 0 || t.End > len(src) || t.Start > t.End {
		return ""
	}
	return src[t.Start:t.End]
}
