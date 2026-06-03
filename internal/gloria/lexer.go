package gloria

var keywords = map[string]TokenType{
	"fn":     FN,
	"let":    LET,
	"return": RETURN,
	"reg":    REG,
	"if":     IF,
	"while":  WHILE,
}

type Lexer struct {
	input   string
	pos     int
	readPos int
	ch      byte
}

func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) scanIdentifier() (int, int) {
	start := l.pos
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}

	return start, l.pos
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
}

func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *Lexer) scanNumber() (int, int) {
	start := l.pos

	for isDigit(l.ch) {
		l.readChar()
	}
	return start, l.pos
}

func (l *Lexer) scanAtReg() (int, int) {
	start := l.pos
	l.readChar()
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}

	return start, l.pos
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != '\r' && l.ch != 0 {
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	start := l.pos
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[start:l.pos]
}

func (l *Lexer) readNumber() string {
	start := l.pos
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[start:l.pos]
}

func (l *Lexer) readAtReg() string {
	start := l.pos
	l.readChar()
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[start:l.pos]
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func lookupIdent(lit string) TokenType {
	switch lit {
	case "fn":
		return FN
	case "let":
		return LET
	case "return":
		return RETURN
	case "reg":
		return REG
	case "if":
		return IF
	case "while":
		return WHILE
	}

	return IDENT
}

func (l *Lexer) NextToken() Token {
	for {
		l.skipWhitespace()

		if l.ch == '/' && l.peekChar() == '/' {
			l.skipComment()
			continue
		}
		break
	}

	var tok Token
	tok.Start = l.pos

	switch l.ch {
	case '+':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = PLUS_ASSIGN
			l.readChar()
		} else {
			tok.Type = PLUS
			l.readChar()
		}
	case '"':
		tok.Type = STRING
		tok.Start = l.pos + 1
		l.readChar()
		for l.ch != '"' && l.ch != 0 {
			l.readChar()
		}
		tok.End = l.pos
		if l.ch == '"' {
			l.readChar()
		} else {
			tok.Type = ILLEGAL
		}
		return tok
	case '-':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = MINUS_ASSIGN
			l.readChar()
		} else {
			tok.Type = MINUS
			l.readChar()
		}
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = EQ
			l.readChar()
		} else {
			tok.Type = ASSIGN
			l.readChar()
		}
	case '/':
		tok.Type = SLASH
		l.readChar()
	case '{':
		tok.Type = LBRACE
		l.readChar()
	case '}':
		tok.Type = RBRACE
		l.readChar()
	case '(':
		tok.Type = LPAREN
		l.readChar()
	case ')':
		tok.Type = RPAREN
		l.readChar()
	case '<':
		tok.Type = LT
		l.readChar()
	case '>':
		tok.Type = GT
		l.readChar()
	case '@':
		start, end := l.scanAtReg()
		return Token{Type: ATREG, Start: start, End: end}
	case '*':
		tok.Type = STAR 
		l.readChar()
	case '%':
		tok.Type = PERCENT
		l.readChar()
	case '|':
		tok.Type = PIPE 
		l.readChar()
	case '^':
		tok.Type = CARET 
		l.readChar()
	case 0:
		tok.Type = EOF
		tok.Start = l.pos
		tok.End = l.pos
	default:
		if isLetter(l.ch) {
			start, end := l.scanIdentifier()
			lit := l.input[start:end]
			tok.Type = lookupIdent(lit)
			tok.Start = start
			tok.End = end
			return tok
		} else if isDigit(l.ch) {
			start, end := l.scanNumber()
			tok.Type = INT
			tok.Start = start
			tok.End = end
			return tok
		} else {
			tok.Type = ILLEGAL
			l.readChar()
		}
	}

	tok.End = l.pos
	return tok
}
