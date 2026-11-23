package ast

import (
	"bytes"
	"strconv"
	"strings"
)

type Node interface {
	TokenLiteral() string
	String() string
}
type Statement interface {
	Node
	statementNode()
}
type Expression interface {
	Node
	expressionNode()
}

type Program struct{ Statements []Statement }

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}
func (p *Program) String() string {
	var b bytes.Buffer
	for _, s := range p.Statements {
		b.WriteString(s.String())
	}
	return b.String()
}

type LetStatement struct {
	Name  *Identifier
	Value Expression
}

func (s *LetStatement) statementNode()       {}
func (s *LetStatement) TokenLiteral() string { return "let" }
func (s *LetStatement) String() string {
	var b bytes.Buffer
	b.WriteString("let " + s.Name.String() + " = ")
	if s.Value != nil {
		b.WriteString(s.Value.String())
	}
	b.WriteString(";")
	return b.String()
}

type ReturnStatement struct{ ReturnValue Expression }

func (s *ReturnStatement) statementNode()       {}
func (s *ReturnStatement) TokenLiteral() string { return "return" }
func (s *ReturnStatement) String() string {
	var b bytes.Buffer
	b.WriteString("return ")
	if s.ReturnValue != nil {
		b.WriteString(s.ReturnValue.String())
	}
	b.WriteString(";")
	return b.String()
}

type ExpressionStatement struct{ Expression Expression }

func (s *ExpressionStatement) statementNode() {}
func (s *ExpressionStatement) TokenLiteral() string {
	if s.Expression != nil {
		return s.Expression.TokenLiteral()
	}
	return ""
}
func (s *ExpressionStatement) String() string {
	if s.Expression != nil {
		return s.Expression.String()
	}
	return ""
}

type BlockStatement struct{ Statements []Statement }

func (s *BlockStatement) statementNode()       {}
func (s *BlockStatement) TokenLiteral() string { return "{" }
func (s *BlockStatement) String() string {
	var b bytes.Buffer
	b.WriteString("{")
	for _, st := range s.Statements {
		b.WriteString(st.String())
	}
	b.WriteString("}")
	return b.String()
}

type Identifier struct{ Value string }

func (e *Identifier) expressionNode()      {}
func (e *Identifier) TokenLiteral() string { return e.Value }
func (e *Identifier) String() string       { return e.Value }

type IntegerLiteral struct{ Value int64 }

func (e *IntegerLiteral) expressionNode()      {}
func (e *IntegerLiteral) TokenLiteral() string { return e.String() }
func (e *IntegerLiteral) String() string       { return strconv.FormatInt(e.Value, 10) }

type StringLiteral struct{ Value string }

func (e *StringLiteral) expressionNode()      {}
func (e *StringLiteral) TokenLiteral() string { return e.String() }
func (e *StringLiteral) String() string       { return strconv.Quote(e.Value) }

type Boolean struct{ Value bool }

func (e *Boolean) expressionNode() {}
func (e *Boolean) TokenLiteral() string {
	if e.Value {
		return "true"
	}
	return "false"
}
func (e *Boolean) String() string { return e.TokenLiteral() }

type Null struct{}

func (e *Null) expressionNode()      {}
func (e *Null) TokenLiteral() string { return "null" }
func (e *Null) String() string       { return "null" }

type PrefixExpression struct {
	Operator string
	Right    Expression
}

func (e *PrefixExpression) expressionNode()      {}
func (e *PrefixExpression) TokenLiteral() string { return e.Operator }
func (e *PrefixExpression) String() string       { return "(" + e.Operator + e.Right.String() + ")" }

type InfixExpression struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (e *InfixExpression) expressionNode()      {}
func (e *InfixExpression) TokenLiteral() string { return e.Left.TokenLiteral() }
func (e *InfixExpression) String() string {
	return "(" + e.Left.String() + " " + e.Operator + " " + e.Right.String() + ")"
}

type AssignExpression struct {
	Name  *Identifier
	Value Expression
}

func (e *AssignExpression) expressionNode()      {}
func (e *AssignExpression) TokenLiteral() string { return e.Name.Value }
func (e *AssignExpression) String() string       { return e.Name.String() + " = " + e.Value.String() }

type IfExpression struct {
	Condition                Expression
	Consequence, Alternative *BlockStatement
}

func (e *IfExpression) expressionNode()      {}
func (e *IfExpression) TokenLiteral() string { return "if" }
func (e *IfExpression) String() string {
	var b bytes.Buffer
	b.WriteString("if (")
	b.WriteString(e.Condition.String())
	b.WriteString(") ")
	b.WriteString(e.Consequence.String())
	if e.Alternative != nil {
		b.WriteString(" else ")
		b.WriteString(e.Alternative.String())
	}
	return b.String()
}

type WhileExpression struct {
	Condition Expression
	Body      *BlockStatement
}

func (e *WhileExpression) expressionNode()      {}
func (e *WhileExpression) TokenLiteral() string { return "while" }
func (e *WhileExpression) String() string {
	return "while (" + e.Condition.String() + ") " + e.Body.String()
}

type FunctionLiteral struct {
	Parameters []*Identifier
	Body       *BlockStatement
}

func (e *FunctionLiteral) expressionNode()      {}
func (e *FunctionLiteral) TokenLiteral() string { return "fn" }
func (e *FunctionLiteral) String() string {
	params := make([]string, len(e.Parameters))
	for i, p := range e.Parameters {
		params[i] = p.String()
	}
	return "fn(" + strings.Join(params, ", ") + ") " + e.Body.String()
}

type FunctionStatement struct {
	Name     *Identifier
	Function *FunctionLiteral
}

func (s *FunctionStatement) statementNode()       {}
func (s *FunctionStatement) TokenLiteral() string { return "fn" }
func (s *FunctionStatement) String() string       { return "fn " + s.Name.String() + s.Function.String()[2:] }

type CallExpression struct {
	Function  Expression
	Arguments []Expression
}

func (e *CallExpression) expressionNode()      {}
func (e *CallExpression) TokenLiteral() string { return e.Function.TokenLiteral() }
func (e *CallExpression) String() string {
	args := make([]string, len(e.Arguments))
	for i, a := range e.Arguments {
		args[i] = a.String()
	}
	return e.Function.String() + "(" + strings.Join(args, ", ") + ")"
}
