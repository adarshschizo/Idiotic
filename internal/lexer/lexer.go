package lexer

import "idiotic/internal/token"

type Lexer struct {
	input        []rune
	position     int
	readPosition int
	ch           rune
	line         int
	column       int
}

func New(input string) *Lexer {
	l := &Lexer{input: []rune(input), line: 1, column: 0}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition > len(l.input) {
		l.ch = 0
		return
	}
	l.position = l.readPosition
	l.readPosition++
	l.column++
	if l.position >= len(l.input) {
		l.ch = 0
		return
	}
	l.ch = l.input[l.position]
	if l.ch == '\n' {
		l.line++
		l.column = 0
	}
}

func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() token.Token {
	l.skipWhitespaceAndComments()
	line, column := l.line, l.column
	var tok token.Token

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.EQ, Literal: string(ch) + string(l.ch), Line: line, Column: column}
		} else {
			tok = newToken(token.ASSIGN, l.ch, line, column)
		}
	case '+':
		tok = newToken(token.PLUS, l.ch, line, column)
	case '-':
		tok = newToken(token.MINUS, l.ch, line, column)
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.NOT_EQ, Literal: string(ch) + string(l.ch), Line: line, Column: column}
		} else {
			tok = newToken(token.BANG, l.ch, line, column)
		}
	case '*':
		tok = newToken(token.ASTERISK, l.ch, line, column)
	case '/':
		tok = newToken(token.SLASH, l.ch, line, column)
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.LTE, Literal: string(ch) + string(l.ch), Line: line, Column: column}
		} else {
			tok = newToken(token.LT, l.ch, line, column)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.GTE, Literal: string(ch) + string(l.ch), Line: line, Column: column}
		} else {
			tok = newToken(token.GT, l.ch, line, column)
		}
	case '&':
		if l.peekChar() == '&' {
			l.readChar()
			tok = token.Token{Type: token.AND, Literal: "&&", Line: line, Column: column}
		} else {
			tok = token.Token{Type: token.ILLEGAL, Literal: "&", Line: line, Column: column}
		}
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			tok = token.Token{Type: token.OR, Literal: "||", Line: line, Column: column}
		} else {
			tok = token.Token{Type: token.ILLEGAL, Literal: "|", Line: line, Column: column}
		}
	case ',':
		tok = newToken(token.COMMA, l.ch, line, column)
	case ';':
		tok = newToken(token.SEMICOLON, l.ch, line, column)
	case ':':
		tok = newToken(token.COLON, l.ch, line, column)
	case '(':
		tok = newToken(token.LPAREN, l.ch, line, column)
	case ')':
		tok = newToken(token.RPAREN, l.ch, line, column)
	case '{':
		tok = newToken(token.LBRACE, l.ch, line, column)
	case '}':
		tok = newToken(token.RBRACE, l.ch, line, column)
	case '"':
		text, ok := l.readString()
		if ok {
			tok = token.Token{Type: token.STRING, Literal: text, Line: line, Column: column}
		} else {
			tok = token.Token{Type: token.ILLEGAL, Literal: "unterminated string", Line: line, Column: column}
		}
	case 0:
		tok = token.Token{Type: token.EOF, Literal: "", Line: line, Column: column}
	default:
		if isLetter(l.ch) {
			literal := l.readIdentifier()
			return token.Token{Type: token.LookupIdent(literal), Literal: literal, Line: line, Column: column}
		}
		if isDigit(l.ch) {
			return token.Token{Type: token.INT, Literal: l.readNumber(), Line: line, Column: column}
		}
		tok = token.Token{Type: token.ILLEGAL, Literal: string(l.ch), Line: line, Column: column}
	}
	l.readChar()
	return tok
}

func newToken(t token.Type, ch rune, line, column int) token.Token {
	return token.Token{Type: t, Literal: string(ch), Line: line, Column: column}
}

func (l *Lexer) skipWhitespaceAndComments() {
	for {
		for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
			l.readChar()
		}
		if l.ch == '/' && l.peekChar() == '/' {
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
			continue
		}
		return
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return string(l.input[position:l.position])
}

func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return string(l.input[position:l.position])
}

func (l *Lexer) readString() (string, bool) {
	var out []rune
	l.readChar()
	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case '"':
				out = append(out, '"')
			case '\\':
				out = append(out, '\\')
			default:
				out = append(out, l.ch)
			}
		} else {
			out = append(out, l.ch)
		}
		l.readChar()
	}
	return string(out), l.ch == '"'
}

func isLetter(ch rune) bool { return ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' }
func isDigit(ch rune) bool  { return ch >= '0' && ch <= '9' }
