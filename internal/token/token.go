package token

type Type string

const (
	ILLEGAL Type = "ILLEGAL"
	EOF     Type = "EOF"

	IDENT  Type = "IDENT"
	INT    Type = "INT"
	STRING Type = "STRING"

	ASSIGN   Type = "="
	PLUS     Type = "+"
	MINUS    Type = "-"
	BANG     Type = "!"
	ASTERISK Type = "*"
	SLASH    Type = "/"
	LT       Type = "<"
	GT       Type = ">"
	EQ       Type = "=="
	NOT_EQ   Type = "!="
	LTE      Type = "<="
	GTE      Type = ">="
	AND      Type = "&&"
	OR       Type = "||"

	COMMA     Type = ","
	SEMICOLON Type = ";"
	COLON     Type = ":"
	LPAREN    Type = "("
	RPAREN    Type = ")"
	LBRACE    Type = "{"
	RBRACE    Type = "}"

	FUNCTION Type = "FUNCTION"
	LET      Type = "LET"
	TRUE     Type = "TRUE"
	FALSE    Type = "FALSE"
	NULL     Type = "NULL"
	IF       Type = "IF"
	ELSE     Type = "ELSE"
	RETURN   Type = "RETURN"
	WHILE    Type = "WHILE"
	FOR      Type = "FOR"
)

type Token struct {
	Type    Type
	Literal string
	Line    int
	Column  int
}

var keywords = map[string]Type{
	"fn": FUNCTION, "let": LET, "true": TRUE, "false": FALSE,
	"null": NULL, "if": IF, "else": ELSE, "return": RETURN,
	"while": WHILE, "for": FOR,
}

func LookupIdent(ident string) Type {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
