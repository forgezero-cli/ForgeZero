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
