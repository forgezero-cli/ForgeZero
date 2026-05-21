package gloria

type TokenType string

const (
    ILLEGAL TokenType = "ILLEGAL"
    EOF     TokenType = "EOF"

    IDENT TokenType = "IDENT"
    INT   TokenType = "INT"
    ATREG TokenType = "ATREG"

    PLUS   TokenType = "+"
    MINUS  TokenType = "-"
    PLUS_ASSIGN TokenType = "+="
    MINUS_ASSIGN TokenType = "-="
    ASSIGN TokenType = "="
    LBRACE TokenType = "{"
    RBRACE TokenType = "}"
    LPAREN TokenType = "("
    RPAREN TokenType = ")"

    FN     TokenType = "FN"
    LET    TokenType = "LET"
    RETURN TokenType = "RETURN"
    REG    TokenType = "REG"
)

type Token struct {
    Type TokenType
    Lit  string
}

var keywords = map[string]TokenType{
    "fn":     FN,
    "let":    LET,
    "return": RETURN,
    "reg":    REG,
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

func (l *Lexer) skipWhitespace() {
    for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
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
    l.readChar() // consume '@'
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

func (l *Lexer) NextToken() Token {
    l.skipWhitespace()

    var tok Token

    switch l.ch {
    case '+':
        if l.peekChar() == '=' {
            l.readChar()
            tok = Token{Type: PLUS_ASSIGN, Lit: "+="}
        } else {
            tok = Token{Type: PLUS, Lit: "+"}
        }
    case '-':
        if l.peekChar() == '=' {
            l.readChar()
            tok = Token{Type: MINUS_ASSIGN, Lit: "-="}
        } else {
            tok = Token{Type: MINUS, Lit: "-"}
        }
    case '=':
        tok = Token{Type: ASSIGN, Lit: "="}
    case '{':
        tok = Token{Type: LBRACE, Lit: "{"}
    case '}':
        tok = Token{Type: RBRACE, Lit: "}"}
    case '(':
        tok = Token{Type: LPAREN, Lit: "("}
    case ')':
        tok = Token{Type: RPAREN, Lit: ")"}
    case '@':
        lit := l.readAtReg()
        tok = Token{Type: ATREG, Lit: lit}
        return tok
    case 0:
        tok = Token{Type: EOF, Lit: ""}
        return tok
    default:
        if isLetter(l.ch) {
            lit := l.readIdentifier()
            if typ, ok := keywords[lit]; ok {
                return Token{Type: typ, Lit: lit}
            }
            return Token{Type: IDENT, Lit: lit}
        } else if isDigit(l.ch) {
            lit := l.readNumber()
            return Token{Type: INT, Lit: lit}
        } else {
            tok = Token{Type: ILLEGAL, Lit: string(l.ch)}
        }
    }

    l.readChar()
    return tok
}
