package evaluator

import (
	"fmt"

	"idiotic/internal/ast"
	"idiotic/internal/object"
)

var TRUE = &object.Boolean{Value: true}
var FALSE = &object.Boolean{Value: false}
var NULL = &object.Null{}

const maxFunctionCallDepth = 1024

type evaluationState struct {
	functionCallDepth int
}

func Eval(node ast.Node, env *object.Environment) object.Object {
	if node == nil {
		return newError("cannot evaluate a nil AST node")
	}
	if env == nil {
		return newError("cannot evaluate with a nil environment")
	}
	return eval(node, env, &evaluationState{})
}

func eval(node ast.Node, env *object.Environment, state *evaluationState) object.Object {
	switch node := node.(type) {
	case *ast.Program:
		return evalProgram(node, env, state)
	case *ast.ExpressionStatement:
		return eval(node.Expression, env, state)
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}
	case *ast.StringLiteral:
		return &object.String{Value: node.Value}
	case *ast.Boolean:
		if node.Value {
			return TRUE
		}
		return FALSE
	case *ast.Null:
		return NULL
	case *ast.PrefixExpression:
		right := eval(node.Right, env, state)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)
	case *ast.InfixExpression:
		left := eval(node.Left, env, state)
		if isError(left) {
			return left
		}
		if node.Operator == "&&" && !isTruthy(left) {
			return FALSE
		}
		if node.Operator == "||" && isTruthy(left) {
			return TRUE
		}
		right := eval(node.Right, env, state)
		if isError(right) {
			return right
		}
		return evalInfixExpression(node.Operator, left, right)
	case *ast.AssignExpression:
		value := eval(node.Value, env, state)
		if isError(value) {
			return value
		}
		if !env.Assign(node.Name.Value, value) {
			return newError("undefined variable: %s", node.Name.Value)
		}
		return value
	case *ast.LetStatement:
		value := eval(node.Value, env, state)
		if isError(value) {
			return value
		}
		env.Set(node.Name.Value, value)
		return NULL
	case *ast.BlockStatement:
		return evalBlockStatement(node, env, state)
	case *ast.IfExpression:
		return evalIfExpression(node, env, state)
	case *ast.WhileExpression:
		return evalWhileExpression(node, env, state)
	case *ast.ReturnStatement:
		value := eval(node.ReturnValue, env, state)
		if isError(value) {
			return value
		}
		return &object.ReturnValue{Value: value}
	case *ast.Identifier:
		return evalIdentifier(node, env)
	case *ast.FunctionLiteral:
		return &object.Function{Parameters: node.Parameters, Body: node.Body, Env: env}
	case *ast.FunctionStatement:
		fn := &object.Function{Parameters: node.Function.Parameters, Body: node.Function.Body, Env: env}
		env.Set(node.Name.Value, fn)
		return NULL
	case *ast.CallExpression:
		fn := eval(node.Function, env, state)
		if isError(fn) {
			return fn
		}
		args := evalExpressions(node.Arguments, env, state)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyFunction(fn, args, state)
	}
	return NULL
}

func evalProgram(program *ast.Program, env *object.Environment, state *evaluationState) object.Object {
	var result object.Object = NULL
	for _, statement := range program.Statements {
		result = eval(statement, env, state)
		if returnValue, ok := result.(*object.ReturnValue); ok {
			return returnValue.Value
		}
		if err, ok := result.(*object.Error); ok {
			return err
		}
	}
	return result
}
func evalBlockStatement(block *ast.BlockStatement, env *object.Environment, state *evaluationState) object.Object {
	var result object.Object = NULL
	for _, statement := range block.Statements {
		result = eval(statement, env, state)
		if result != nil {
			if result.Type() == objectTypeReturn || result.Type() == objectTypeError {
				return result
			}
		}
	}
	return result
}

const objectTypeReturn = "RETURN_VALUE"
const objectTypeError = "ERROR"

func evalIfExpression(ie *ast.IfExpression, env *object.Environment, state *evaluationState) object.Object {
	condition := eval(ie.Condition, env, state)
	if isError(condition) {
		return condition
	}
	if isTruthy(condition) {
		return eval(ie.Consequence, env, state)
	}
	if ie.Alternative != nil {
		return eval(ie.Alternative, env, state)
	}
	return NULL
}
func evalWhileExpression(w *ast.WhileExpression, env *object.Environment, state *evaluationState) object.Object {
	var result object.Object = NULL
	for {
		condition := eval(w.Condition, env, state)
		if isError(condition) {
			return condition
		}
		if !isTruthy(condition) {
			return result
		}
		result = eval(w.Body, env, state)
		if isError(result) {
			return result
		}
		if result != nil && result.Type() == objectTypeReturn {
			return result
		}
	}
}
func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!":
		return nativeBoolToBooleanObject(!isTruthy(right))
	case "-":
		if value, ok := right.(*object.Integer); ok {
			return &object.Integer{Value: -value.Value}
		}
		return newError("unknown operator: -%s", right.Type())
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}
func evalInfixExpression(operator string, left, right object.Object) object.Object {
	switch {
	case left.Type() == "INTEGER" && right.Type() == "INTEGER":
		l, r := left.(*object.Integer), right.(*object.Integer)
		switch operator {
		case "+":
			return &object.Integer{Value: l.Value + r.Value}
		case "-":
			return &object.Integer{Value: l.Value - r.Value}
		case "*":
			return &object.Integer{Value: l.Value * r.Value}
		case "/":
			if r.Value == 0 {
				return newError("division by zero")
			}
			return &object.Integer{Value: l.Value / r.Value}
		case "<":
			return nativeBoolToBooleanObject(l.Value < r.Value)
		case ">":
			return nativeBoolToBooleanObject(l.Value > r.Value)
		case "<=":
			return nativeBoolToBooleanObject(l.Value <= r.Value)
		case ">=":
			return nativeBoolToBooleanObject(l.Value >= r.Value)
		case "==":
			return nativeBoolToBooleanObject(l.Value == r.Value)
		case "!=":
			return nativeBoolToBooleanObject(l.Value != r.Value)
		}
	case left.Type() == "STRING" && right.Type() == "STRING":
		l, r := left.(*object.String), right.(*object.String)
		switch operator {
		case "+":
			return &object.String{Value: l.Value + r.Value}
		case "==":
			return nativeBoolToBooleanObject(l.Value == r.Value)
		case "!=":
			return nativeBoolToBooleanObject(l.Value != r.Value)
		}
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	case operator == "&&":
		return nativeBoolToBooleanObject(isTruthy(left) && isTruthy(right))
	case operator == "||":
		return nativeBoolToBooleanObject(isTruthy(left) || isTruthy(right))
	}
	return newError("type mismatch: %s %s %s", left.Type(), operator, right.Type())
}
func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}
	switch node.Value {
	case "print":
		return Builtins["print"]
	case "len":
		return Builtins["len"]
	}
	return newError("identifier not found: " + node.Value)
}
func evalExpressions(expressions []ast.Expression, env *object.Environment, state *evaluationState) []object.Object {
	var result []object.Object
	for _, e := range expressions {
		evaluated := eval(e, env, state)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}
	return result
}
func applyFunction(fn object.Object, args []object.Object, state *evaluationState) object.Object {
	switch fn := fn.(type) {
	case *object.Function:
		if len(fn.Parameters) != len(args) {
			return newError("wrong number of arguments: got=%d, want=%d", len(args), len(fn.Parameters))
		}
		if state.functionCallDepth >= maxFunctionCallDepth {
			return newError("maximum function call depth exceeded (limit %d)", maxFunctionCallDepth)
		}
		state.functionCallDepth++
		defer func() {
			state.functionCallDepth--
		}()
		extended := object.NewEnclosedEnvironment(fn.Env)
		for i, param := range fn.Parameters {
			extended.Set(param.Value, args[i])
		}
		evaluated := eval(fn.Body, extended, state)
		if returnValue, ok := evaluated.(*object.ReturnValue); ok {
			return returnValue.Value
		}
		return evaluated
	case *object.Builtin:
		return fn.Fn(args...)
	default:
		return newError("not a function: %s", fn.Type())
	}
}
func isTruthy(obj object.Object) bool {
	switch obj := obj.(type) {
	case *object.Null:
		return false
	case *object.Boolean:
		return obj.Value
	default:
		return true
	}
}
func nativeBoolToBooleanObject(v bool) *object.Boolean {
	if v {
		return TRUE
	}
	return FALSE
}
func isError(obj object.Object) bool { return obj != nil && obj.Type() == objectTypeError }
func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}
