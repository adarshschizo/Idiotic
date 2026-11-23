package compiler

import (
	"fmt"
	"math"

	"idiotic/internal/ast"
	"idiotic/internal/code"
	"idiotic/internal/object"
)

type Compiler struct {
	instructions     code.Instructions
	constants        []object.Object
	globals          map[string]int
	nextGlobal       int
	rootSymbols      *symbolTable
	symbols          *symbolTable
	instructionStack []code.Instructions
	symbolStack      []*symbolTable
	functionDepth    int
	temporaryIndex   int
}

func New() *Compiler {
	root := newSymbolTable()
	root.defineBuiltin("print", 0)
	root.defineBuiltin("len", 1)
	return &Compiler{
		globals:     make(map[string]int),
		rootSymbols: root,
		symbols:     root,
	}
}

func (c *Compiler) Compile(node ast.Node) error {
	if node == nil {
		return fmt.Errorf("cannot compile a nil AST node")
	}
	switch node := node.(type) {
	case *ast.Program:
		return c.compileProgram(node)
	case *ast.ExpressionStatement:
		if err := c.compileExpression(node.Expression); err != nil {
			return err
		}
		_, err := c.emit(code.OpPop)
		return err
	case *ast.LetStatement:
		return c.compileLetStatement(node)
	case *ast.ReturnStatement:
		if c.functionDepth == 0 {
			return fmt.Errorf("return statements are only valid inside functions")
		}
		if err := c.compileExpression(node.ReturnValue); err != nil {
			return err
		}
		_, err := c.emit(code.OpReturnValue)
		return err
	case *ast.BlockStatement:
		return c.compileBlock(node, false)
	case *ast.FunctionStatement:
		return c.compileFunctionStatement(node)
	case ast.Expression:
		return c.compileExpression(node)
	default:
		return fmt.Errorf("unsupported AST node %T", node)
	}
}

func (c *Compiler) Bytecode() *code.Bytecode {
	numGlobals := c.nextGlobal
	if numGlobals == 0 {
		numGlobals = 1
	}
	return &code.Bytecode{
		Instructions: append(code.Instructions(nil), c.instructions...),
		Constants:    append([]object.Object(nil), c.constants...),
		NumGlobals:   numGlobals,
	}
}

func (c *Compiler) compileProgram(program *ast.Program) error {
	for _, statement := range program.Statements {
		if err := c.Compile(statement); err != nil {
			return err
		}
	}
	if len(program.Statements) == 0 {
		if _, err := c.emit(code.OpNull); err != nil {
			return err
		}
		_, err := c.emit(code.OpPop)
		return err
	}
	if _, ok := program.Statements[len(program.Statements)-1].(*ast.ExpressionStatement); !ok {
		if _, err := c.emit(code.OpNull); err != nil {
			return err
		}
		_, err := c.emit(code.OpPop)
		return err
	}
	return nil
}

func (c *Compiler) compileLetStatement(statement *ast.LetStatement) error {
	if statement.Name == nil {
		return fmt.Errorf("let statement is missing a name")
	}
	symbol, err := c.defineName(statement.Name.Value)
	if err != nil {
		return err
	}
	if function, ok := statement.Value.(*ast.FunctionLiteral); ok {
		err = c.compileFunctionLiteral(function, statement.Name.Value)
	} else {
		err = c.compileExpression(statement.Value)
	}
	if err != nil {
		return err
	}
	return c.storeSymbol(symbol, true)
}

func (c *Compiler) compileBlock(block *ast.BlockStatement, keepLast bool) error {
	if block == nil {
		return fmt.Errorf("cannot compile a nil block")
	}
	for i, statement := range block.Statements {
		isLast := i == len(block.Statements)-1
		expression, isExpression := statement.(*ast.ExpressionStatement)
		if isExpression {
			if err := c.compileExpression(expression.Expression); err != nil {
				return err
			}
			if !keepLast || !isLast {
				if _, err := c.emit(code.OpPop); err != nil {
					return err
				}
			}
			continue
		}
		if err := c.Compile(statement); err != nil {
			return err
		}
		if keepLast && isLast {
			if _, isReturn := statement.(*ast.ReturnStatement); !isReturn {
				if _, err := c.emit(code.OpNull); err != nil {
					return err
				}
			}
		}
	}
	if keepLast && len(block.Statements) == 0 {
		_, err := c.emit(code.OpNull)
		return err
	}
	return nil
}

func (c *Compiler) compileExpression(expression ast.Expression) error {
	if expression == nil {
		return fmt.Errorf("cannot compile a nil expression")
	}
	switch expression := expression.(type) {
	case *ast.IntegerLiteral:
		return c.compileConstant(&object.Integer{Value: expression.Value})
	case *ast.StringLiteral:
		return c.compileConstant(&object.String{Value: expression.Value})
	case *ast.Boolean:
		if expression.Value {
			_, err := c.emit(code.OpTrue)
			return err
		}
		_, err := c.emit(code.OpFalse)
		return err
	case *ast.Null:
		_, err := c.emit(code.OpNull)
		return err
	case *ast.Identifier:
		symbol, err := c.resolveOrDefineGlobal(expression.Value)
		if err != nil {
			return err
		}
		return c.loadSymbol(symbol)
	case *ast.PrefixExpression:
		if err := c.compileExpression(expression.Right); err != nil {
			return err
		}
		switch expression.Operator {
		case "!":
			_, err := c.emit(code.OpBang)
			return err
		case "-":
			_, err := c.emit(code.OpMinus)
			return err
		default:
			return fmt.Errorf("unsupported prefix operator %q", expression.Operator)
		}
	case *ast.InfixExpression:
		return c.compileInfixExpression(expression)
	case *ast.AssignExpression:
		symbol, ok := c.symbols.resolve(expression.Name.Value)
		if !ok {
			var err error
			_, err = c.defineGlobal(expression.Name.Value)
			if err != nil {
				return err
			}
			symbol = c.rootSymbols.store[expression.Name.Value]
		}
		if err := c.compileExpression(expression.Value); err != nil {
			return err
		}
		if err := c.storeSymbol(symbol, false); err != nil {
			return err
		}
		return c.loadSymbol(symbol)
	case *ast.IfExpression:
		return c.compileIfExpression(expression)
	case *ast.WhileExpression:
		return c.compileWhileExpression(expression)
	case *ast.FunctionLiteral:
		return c.compileFunctionLiteral(expression, "")
	case *ast.CallExpression:
		return c.compileCallExpression(expression)
	default:
		return fmt.Errorf("unsupported expression %T", expression)
	}
}

func (c *Compiler) compileConstant(value object.Object) error {
	index, err := c.addConstant(value)
	if err != nil {
		return err
	}
	_, err = c.emit(code.OpConstant, index)
	return err
}

func (c *Compiler) compileInfixExpression(expression *ast.InfixExpression) error {
	if expression.Operator == "&&" || expression.Operator == "||" {
		return c.compileLogicalExpression(expression)
	}
	if err := c.compileExpression(expression.Left); err != nil {
		return err
	}
	if err := c.compileExpression(expression.Right); err != nil {
		return err
	}

	var op code.Opcode
	negate := false
	switch expression.Operator {
	case "+":
		op = code.OpAdd
	case "-":
		op = code.OpSub
	case "*":
		op = code.OpMul
	case "/":
		op = code.OpDiv
	case "==":
		op = code.OpEqual
	case "!=":
		op = code.OpNotEqual
	case ">":
		op = code.OpGreaterThan
	case "<":
		op = code.OpLessThan
	case ">=":
		op, negate = code.OpLessThan, true
	case "<=":
		op, negate = code.OpGreaterThan, true
	default:
		return fmt.Errorf("unsupported infix operator %q", expression.Operator)
	}
	if _, err := c.emit(op); err != nil {
		return err
	}
	if negate {
		_, err := c.emit(code.OpBang)
		return err
	}
	return nil
}

func (c *Compiler) compileLogicalExpression(expression *ast.InfixExpression) error {
	if err := c.compileExpression(expression.Left); err != nil {
		return err
	}
	jumpNotTruthy, err := c.emit(code.OpJumpNotTruthy, 0)
	if err != nil {
		return err
	}

	if expression.Operator == "&&" {
		if err := c.compileExpression(expression.Right); err != nil {
			return err
		}
		if _, err := c.emit(code.OpToBoolean); err != nil {
			return err
		}
		jumpEnd, err := c.emit(code.OpJump, 0)
		if err != nil {
			return err
		}
		if err := c.changeOperand(jumpNotTruthy, len(c.instructions)); err != nil {
			return err
		}
		if _, err := c.emit(code.OpFalse); err != nil {
			return err
		}
		return c.changeOperand(jumpEnd, len(c.instructions))
	}

	if _, err := c.emit(code.OpTrue); err != nil {
		return err
	}
	jumpEnd, err := c.emit(code.OpJump, 0)
	if err != nil {
		return err
	}
	if err := c.changeOperand(jumpNotTruthy, len(c.instructions)); err != nil {
		return err
	}
	if err := c.compileExpression(expression.Right); err != nil {
		return err
	}
	if _, err := c.emit(code.OpToBoolean); err != nil {
		return err
	}
	return c.changeOperand(jumpEnd, len(c.instructions))
}

func (c *Compiler) compileIfExpression(expression *ast.IfExpression) error {
	if err := c.compileExpression(expression.Condition); err != nil {
		return err
	}
	jumpNotTruthy, err := c.emit(code.OpJumpNotTruthy, 0)
	if err != nil {
		return err
	}
	if err := c.compileBlock(expression.Consequence, true); err != nil {
		return err
	}
	jumpEnd, err := c.emit(code.OpJump, 0)
	if err != nil {
		return err
	}
	if err := c.changeOperand(jumpNotTruthy, len(c.instructions)); err != nil {
		return err
	}
	if expression.Alternative == nil {
		if _, err := c.emit(code.OpNull); err != nil {
			return err
		}
	} else if err := c.compileBlock(expression.Alternative, true); err != nil {
		return err
	}
	return c.changeOperand(jumpEnd, len(c.instructions))
}

func (c *Compiler) compileWhileExpression(expression *ast.WhileExpression) error {
	c.temporaryIndex++
	resultSymbol, err := c.defineName(fmt.Sprintf("\x00loop-result-%d", c.temporaryIndex))
	if err != nil {
		return err
	}
	if _, err := c.emit(code.OpNull); err != nil {
		return err
	}
	if err := c.storeSymbol(resultSymbol, true); err != nil {
		return err
	}

	loopStart := len(c.instructions)
	if err := c.compileExpression(expression.Condition); err != nil {
		return err
	}
	exitJump, err := c.emit(code.OpJumpNotTruthy, 0)
	if err != nil {
		return err
	}
	if err := c.compileBlock(expression.Body, true); err != nil {
		return err
	}
	if err := c.storeSymbol(resultSymbol, false); err != nil {
		return err
	}
	if _, err := c.emit(code.OpJump, loopStart); err != nil {
		return err
	}
	if err := c.changeOperand(exitJump, len(c.instructions)); err != nil {
		return err
	}
	return c.loadSymbol(resultSymbol)
}

func (c *Compiler) defineGlobal(name string) (int, error) {
	if index, exists := c.globals[name]; exists {
		return index, nil
	}
	index, err := c.reserveGlobal()
	if err != nil {
		return 0, err
	}
	c.globals[name] = index
	c.rootSymbols.defineGlobal(name, index)
	return index, nil
}

func (c *Compiler) reserveGlobal() (int, error) {
	if c.nextGlobal > math.MaxUint16 {
		return 0, fmt.Errorf("global variable table exceeds 16-bit index limit")
	}
	index := c.nextGlobal
	c.nextGlobal++
	return index, nil
}

func (c *Compiler) emit(op code.Opcode, operands ...int) (int, error) {
	instruction, err := code.Make(op, operands...)
	if err != nil {
		return 0, err
	}
	position := len(c.instructions)
	c.instructions = append(c.instructions, instruction...)
	return position, nil
}

func (c *Compiler) changeOperand(position, operand int) error {
	if position < 0 || position >= len(c.instructions) {
		return fmt.Errorf("instruction position %d is out of range", position)
	}
	op := code.Opcode(c.instructions[position])
	definition, err := code.Lookup(op)
	if err != nil {
		return err
	}
	if len(definition.OperandWidths) != 1 || definition.OperandWidths[0] != 2 {
		return fmt.Errorf("cannot patch instruction %s: expected one 2-byte operand", definition.Name)
	}
	instruction, err := code.Make(op, operand)
	if err != nil {
		return err
	}
	if position+len(instruction) > len(c.instructions) {
		return fmt.Errorf("instruction at %d is truncated", position)
	}
	if len(instruction) != 1+definition.OperandWidths[0] {
		return fmt.Errorf("cannot patch instruction %s with a different width", definition.Name)
	}
	copy(c.instructions[position:], instruction)
	return nil
}

func (c *Compiler) defineName(name string) (symbol, error) {
	if c.symbols.outer == nil {
		_, err := c.defineGlobal(name)
		if err != nil {
			return symbol{}, err
		}
		return c.rootSymbols.store[name], nil
	}
	return c.symbols.defineLocal(name)
}

func (c *Compiler) resolveOrDefineGlobal(name string) (symbol, error) {
	if value, ok := c.symbols.resolve(name); ok {
		return value, nil
	}
	_, err := c.defineGlobal(name)
	if err != nil {
		return symbol{}, err
	}
	return c.rootSymbols.store[name], nil
}

func (c *Compiler) loadSymbol(value symbol) error {
	switch value.scope {
	case globalScope:
		_, err := c.emit(code.OpGetGlobal, value.index)
		return err
	case localScope:
		_, err := c.emit(code.OpGetLocal, value.index)
		return err
	case builtinScope:
		_, err := c.emit(code.OpGetBuiltin, value.index)
		return err
	case freeScope:
		_, err := c.emit(code.OpGetFree, value.index)
		return err
	case functionScope:
		_, err := c.emit(code.OpCurrentClosure)
		return err
	default:
		return fmt.Errorf("cannot load symbol %q with scope %q", value.name, value.scope)
	}
}

func (c *Compiler) loadSymbolReference(value symbol) error {
	switch value.scope {
	case localScope:
		_, err := c.emit(code.OpGetLocalRef, value.index)
		return err
	case freeScope:
		_, err := c.emit(code.OpGetFreeRef, value.index)
		return err
	case functionScope:
		_, err := c.emit(code.OpCurrentClosure)
		return err
	default:
		return fmt.Errorf("cannot capture symbol %q with scope %q", value.name, value.scope)
	}
}

func (c *Compiler) storeSymbol(value symbol, define bool) error {
	switch value.scope {
	case globalScope:
		op := code.OpSetGlobal
		if define {
			op = code.OpDefineGlobal
		}
		_, err := c.emit(op, value.index)
		return err
	case localScope:
		_, err := c.emit(code.OpSetLocal, value.index)
		return err
	case freeScope:
		if define {
			return fmt.Errorf("cannot define captured symbol %q", value.name)
		}
		_, err := c.emit(code.OpSetFree, value.index)
		return err
	default:
		return fmt.Errorf("cannot assign to symbol %q with scope %q", value.name, value.scope)
	}
}

func (c *Compiler) compileFunctionStatement(statement *ast.FunctionStatement) error {
	if statement.Name == nil || statement.Function == nil {
		return fmt.Errorf("function declaration is incomplete")
	}
	symbol, err := c.defineName(statement.Name.Value)
	if err != nil {
		return err
	}
	if err := c.compileFunctionLiteral(statement.Function, statement.Name.Value); err != nil {
		return err
	}
	return c.storeSymbol(symbol, true)
}

func (c *Compiler) compileFunctionLiteral(function *ast.FunctionLiteral, name string) error {
	if function == nil || function.Body == nil {
		return fmt.Errorf("function literal is incomplete")
	}
	c.instructionStack = append(c.instructionStack, c.instructions)
	c.symbolStack = append(c.symbolStack, c.symbols)
	c.instructions = nil
	c.symbols = newEnclosedSymbolTable(c.symbols)
	c.functionDepth++

	if name != "" {
		c.symbols.defineFunctionName(name)
	}
	for _, parameter := range function.Parameters {
		if parameter == nil {
			return c.abortFunctionScope(fmt.Errorf("function has a nil parameter"))
		}
		if _, err := c.symbols.defineLocal(parameter.Value); err != nil {
			return c.abortFunctionScope(err)
		}
	}
	if err := c.predeclareBlockLocals(function.Body); err != nil {
		return c.abortFunctionScope(err)
	}
	if err := c.compileBlock(function.Body, true); err != nil {
		return c.abortFunctionScope(err)
	}
	lastIsReturn := len(function.Body.Statements) > 0
	if lastIsReturn {
		_, lastIsReturn = function.Body.Statements[len(function.Body.Statements)-1].(*ast.ReturnStatement)
	}
	if !lastIsReturn {
		if _, err := c.emit(code.OpReturnValue); err != nil {
			return c.abortFunctionScope(err)
		}
	}

	functionInstructions := append(code.Instructions(nil), c.instructions...)
	numLocals := c.symbols.numDefinitions
	freeSymbols := append([]symbol(nil), c.symbols.freeSymbols...)
	c.leaveFunctionScope()

	for _, free := range freeSymbols {
		if err := c.loadSymbolReference(free); err != nil {
			return err
		}
	}
	functionConstant, err := c.addConstant(&object.CompiledFunction{
		Instructions:  functionInstructions,
		NumLocals:     numLocals,
		NumParameters: len(function.Parameters),
	})
	if err != nil {
		return err
	}
	_, err = c.emit(code.OpClosure, functionConstant, len(freeSymbols))
	return err
}

func (c *Compiler) compileCallExpression(expression *ast.CallExpression) error {
	if err := c.compileExpression(expression.Function); err != nil {
		return err
	}
	for _, argument := range expression.Arguments {
		if err := c.compileExpression(argument); err != nil {
			return err
		}
	}
	_, err := c.emit(code.OpCall, len(expression.Arguments))
	return err
}

func (c *Compiler) addConstant(value object.Object) (int, error) {
	if len(c.constants) > math.MaxUint16 {
		return 0, fmt.Errorf("constant pool exceeds 16-bit index limit")
	}
	index := len(c.constants)
	c.constants = append(c.constants, value)
	return index, nil
}

func (c *Compiler) leaveFunctionScope() {
	last := len(c.instructionStack) - 1
	c.instructions = c.instructionStack[last]
	c.instructionStack = c.instructionStack[:last]
	c.symbols = c.symbolStack[len(c.symbolStack)-1]
	c.symbolStack = c.symbolStack[:len(c.symbolStack)-1]
	c.functionDepth--
}

func (c *Compiler) abortFunctionScope(err error) error {
	c.leaveFunctionScope()
	return err
}

func (c *Compiler) predeclareBlockLocals(block *ast.BlockStatement) error {
	if block == nil {
		return nil
	}
	for _, statement := range block.Statements {
		switch statement := statement.(type) {
		case *ast.LetStatement:
			if statement.Name != nil {
				if _, err := c.symbols.defineLocal(statement.Name.Value); err != nil {
					return err
				}
			}
			if err := c.predeclareExpressionLocals(statement.Value); err != nil {
				return err
			}
		case *ast.FunctionStatement:
			if statement.Name != nil {
				if _, err := c.symbols.defineLocal(statement.Name.Value); err != nil {
					return err
				}
			}
		case *ast.ExpressionStatement:
			if err := c.predeclareExpressionLocals(statement.Expression); err != nil {
				return err
			}
		case *ast.ReturnStatement:
			if err := c.predeclareExpressionLocals(statement.ReturnValue); err != nil {
				return err
			}
		case *ast.BlockStatement:
			if err := c.predeclareBlockLocals(statement); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Compiler) predeclareExpressionLocals(expression ast.Expression) error {
	switch expression := expression.(type) {
	case *ast.PrefixExpression:
		return c.predeclareExpressionLocals(expression.Right)
	case *ast.InfixExpression:
		if err := c.predeclareExpressionLocals(expression.Left); err != nil {
			return err
		}
		return c.predeclareExpressionLocals(expression.Right)
	case *ast.AssignExpression:
		return c.predeclareExpressionLocals(expression.Value)
	case *ast.IfExpression:
		if err := c.predeclareExpressionLocals(expression.Condition); err != nil {
			return err
		}
		if err := c.predeclareBlockLocals(expression.Consequence); err != nil {
			return err
		}
		return c.predeclareBlockLocals(expression.Alternative)
	case *ast.WhileExpression:
		if err := c.predeclareExpressionLocals(expression.Condition); err != nil {
			return err
		}
		return c.predeclareBlockLocals(expression.Body)
	case *ast.CallExpression:
		if err := c.predeclareExpressionLocals(expression.Function); err != nil {
			return err
		}
		for _, argument := range expression.Arguments {
			if err := c.predeclareExpressionLocals(argument); err != nil {
				return err
			}
		}
	}
	return nil
}
