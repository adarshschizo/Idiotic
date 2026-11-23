package parser

import (
	"fmt"
	"strconv"

	"idiotic/internal/ast"
	"idiotic/internal/lexer"
	"idiotic/internal/token"
)

type precedence int

const (
	LOWEST precedence = iota
	ASSIGNMENT
	OR
	AND
	EQUALS
	LESSGREATER
	SUM
	PRODUCT
	PREFIX
	CALL
)

type Parser struct {
	l              *lexer.Lexer
	cur            token.Token
	peek           token.Token
	errors         []string
	prefixParseFns map[token.Type]func() ast.Expression
	infixParseFns  map[token.Type]func(ast.Expression) ast.Expression
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l, errors: []string{}, prefixParseFns: map[token.Type]func() ast.Expression{}, infixParseFns: map[token.Type]func(ast.Expression) ast.Expression{}}
	p.prefixParseFns[token.IDENT] = p.parseIdentifier
	p.prefixParseFns[token.INT] = p.parseIntegerLiteral
	p.prefixParseFns[token.STRING] = p.parseStringLiteral
	p.prefixParseFns[token.TRUE] = p.parseBoolean
	p.prefixParseFns[token.FALSE] = p.parseBoolean
	p.prefixParseFns[token.NULL] = p.parseNull
	p.prefixParseFns[token.BANG] = p.parsePrefixExpression
	p.prefixParseFns[token.MINUS] = p.parsePrefixExpression
	p.prefixParseFns[token.LPAREN] = p.parseGroupedExpression
	p.prefixParseFns[token.IF] = p.parseIfExpression
	p.prefixParseFns[token.WHILE] = p.parseWhileExpression
	p.prefixParseFns[token.FUNCTION] = p.parseFunctionLiteral
	p.infixParseFns[token.ASSIGN] = p.parseAssignExpression
	p.infixParseFns[token.PLUS] = p.parseInfixExpression
	p.infixParseFns[token.MINUS] = p.parseInfixExpression
	p.infixParseFns[token.ASTERISK] = p.parseInfixExpression
	p.infixParseFns[token.SLASH] = p.parseInfixExpression
	p.infixParseFns[token.EQ] = p.parseInfixExpression
	p.infixParseFns[token.NOT_EQ] = p.parseInfixExpression
	p.infixParseFns[token.LT] = p.parseInfixExpression
	p.infixParseFns[token.GT] = p.parseInfixExpression
	p.infixParseFns[token.LTE] = p.parseInfixExpression
	p.infixParseFns[token.GTE] = p.parseInfixExpression
	p.infixParseFns[token.AND] = p.parseInfixExpression
	p.infixParseFns[token.OR] = p.parseInfixExpression
	p.infixParseFns[token.LPAREN] = p.parseCallExpression
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) Errors() []string              { return p.errors }
func (p *Parser) nextToken()                    { p.cur = p.peek; p.peek = p.l.NextToken() }
func (p *Parser) curTokenIs(t token.Type) bool  { return p.cur.Type == t }
func (p *Parser) peekTokenIs(t token.Type) bool { return p.peek.Type == t }
func (p *Parser) expectPeek(t token.Type) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}
func (p *Parser) peekError(t token.Type) {
	p.errors = append(p.errors, fmt.Sprintf("line %d, column %d: expected next token %s, got %s", p.peek.Line, p.peek.Column, t, p.peek.Type))
}
func (p *Parser) noPrefixParseFnError(t token.Type) {
	p.errors = append(p.errors, fmt.Sprintf("line %d, column %d: no prefix parse function for %s", p.cur.Line, p.cur.Column, t))
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	for p.cur.Type != token.EOF {
		if stmt := p.parseStatement(); stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}
	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.cur.Type {
	case token.LET:
		return p.parseLetStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.FUNCTION:
		return p.parseFunctionStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Value: p.cur.Literal}
	if !p.expectPeek(token.ASSIGN) {
		return nil
	}
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{}
	p.nextToken()
	stmt.ReturnValue = p.parseExpression(LOWEST)
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseFunctionStatement() ast.Statement {
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := &ast.Identifier{Value: p.cur.Literal}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	params := p.parseFunctionParameters()
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	body := p.parseBlockStatement()
	return &ast.FunctionStatement{Name: name, Function: &ast.FunctionLiteral{Parameters: params, Body: body}}
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Expression: p.parseExpression(LOWEST)}
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseExpression(precedence precedence) ast.Expression {
	prefix := p.prefixParseFns[p.cur.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.cur.Type)
		return nil
	}
	left := prefix()
	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peek.Type]
		if infix == nil {
			return left
		}
		p.nextToken()
		left = infix(left)
	}
	return left
}

func (p *Parser) parseIdentifier() ast.Expression { return &ast.Identifier{Value: p.cur.Literal} }
func (p *Parser) parseIntegerLiteral() ast.Expression {
	value, err := strconv.ParseInt(p.cur.Literal, 10, 64)
	if err != nil {
		p.errors = append(p.errors, "could not parse "+p.cur.Literal+" as integer")
		return nil
	}
	return &ast.IntegerLiteral{Value: value}
}
func (p *Parser) parseStringLiteral() ast.Expression { return &ast.StringLiteral{Value: p.cur.Literal} }
func (p *Parser) parseBoolean() ast.Expression       { return &ast.Boolean{Value: p.curTokenIs(token.TRUE)} }
func (p *Parser) parseNull() ast.Expression          { return &ast.Null{} }

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{Operator: p.cur.Literal}
	p.nextToken()
	expression.Right = p.parseExpression(PREFIX)
	return expression
}
func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{Left: left, Operator: p.cur.Literal}
	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)
	return expression
}
func (p *Parser) parseAssignExpression(left ast.Expression) ast.Expression {
	name, ok := left.(*ast.Identifier)
	if !ok {
		p.errors = append(p.errors, "left side of assignment must be identifier")
		return nil
	}
	p.nextToken()
	value := p.parseExpression(LOWEST)
	return &ast.AssignExpression{Name: name, Value: value}
}
func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return exp
}

func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	expression.Consequence = p.parseBlockStatement()
	if p.peekTokenIs(token.ELSE) {
		p.nextToken()
		if !p.expectPeek(token.LBRACE) {
			return nil
		}
		expression.Alternative = p.parseBlockStatement()
	}
	return expression
}

func (p *Parser) parseWhileExpression() ast.Expression {
	expression := &ast.WhileExpression{}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	expression.Body = p.parseBlockStatement()
	return expression
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{}
	p.nextToken()
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if stmt := p.parseStatement(); stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}
	if p.curTokenIs(token.EOF) {
		p.errors = append(p.errors, fmt.Sprintf("line %d, column %d: expected } before end of input", p.cur.Line, p.cur.Column))
	}
	return block
}

func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{}
	if p.peekTokenIs(token.IDENT) {
		p.nextToken()
	}
	if !p.peekTokenIs(token.LPAREN) {
		p.errors = append(p.errors, "expected ( after function name")
		return nil
	}
	p.nextToken()
	lit.Parameters = p.parseFunctionParameters()
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	lit.Body = p.parseBlockStatement()
	return lit
}
func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	params := []*ast.Identifier{}
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return params
	}
	p.nextToken()
	for {
		if !p.curTokenIs(token.IDENT) {
			p.errors = append(p.errors, fmt.Sprintf("line %d, column %d: expected parameter name, got %s", p.cur.Line, p.cur.Column, p.cur.Type))
			return nil
		}
		params = append(params, &ast.Identifier{Value: p.cur.Literal})
		if !p.peekTokenIs(token.COMMA) {
			break
		}
		p.nextToken()
		p.nextToken()
	}
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return params
}

func (p *Parser) parseCallExpression(fn ast.Expression) ast.Expression {
	return &ast.CallExpression{Function: fn, Arguments: p.parseExpressionList(token.RPAREN)}
}
func (p *Parser) parseExpressionList(end token.Type) []ast.Expression {
	list := []ast.Expression{}
	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}
	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))
	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}
	if !p.expectPeek(end) {
		return nil
	}
	return list
}

func (p *Parser) peekPrecedence() precedence {
	if v, ok := precedences[p.peek.Type]; ok {
		return v
	}
	return LOWEST
}
func (p *Parser) curPrecedence() precedence {
	if v, ok := precedences[p.cur.Type]; ok {
		return v
	}
	return LOWEST
}

var precedences = map[token.Type]precedence{
	token.ASSIGN: ASSIGNMENT,
	token.OR:     OR, token.AND: AND, token.EQ: EQUALS, token.NOT_EQ: EQUALS,
	token.LT: LESSGREATER, token.GT: LESSGREATER, token.LTE: LESSGREATER, token.GTE: LESSGREATER,
	token.PLUS: SUM, token.MINUS: SUM, token.ASTERISK: PRODUCT, token.SLASH: PRODUCT,
	token.LPAREN: CALL,
}
