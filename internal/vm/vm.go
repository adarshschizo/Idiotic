package vm

import (
	"fmt"
	"io"
	"os"

	"idiotic/internal/code"
	"idiotic/internal/object"
)

const (
	maxStackSize = 2048
	globalSize   = 1 << 16
	maxFrames    = 1025
)

var builtinNames = []string{"print", "len"}

type frame struct {
	closure     *object.Closure
	ip          int
	basePointer int
	locals      []*object.Cell
}

func (frame *frame) instructions(main code.Instructions) code.Instructions {
	if frame.closure == nil {
		return main
	}
	return code.Instructions(frame.closure.Fn.Instructions)
}

type VM struct {
	instructions code.Instructions
	constants    []object.Object
	stack        []object.Object
	sp           int
	globals      []object.Object
	lastPopped   object.Object
	frames       []*frame
	output       io.Writer
}

func New(bytecode *code.Bytecode) *VM {
	return NewWithOutput(bytecode, os.Stdout)
}

func NewWithOutput(bytecode *code.Bytecode, output io.Writer) *VM {
	if output == nil {
		output = os.Stdout
	}
	numGlobals := globalSize
	if bytecode != nil && bytecode.NumGlobals > 0 && bytecode.NumGlobals <= globalSize {
		numGlobals = bytecode.NumGlobals
	}
	vm := &VM{
		stack:   make([]object.Object, maxStackSize),
		globals: make([]object.Object, numGlobals),
		output:  output,
	}
	if bytecode != nil {
		vm.instructions = bytecode.Instructions
		vm.constants = bytecode.Constants
	}
	vm.frames = []*frame{{}}
	return vm
}

func (vm *VM) Run() error {
	for len(vm.frames) > 0 {
		frame := vm.frames[len(vm.frames)-1]
		instructions := frame.instructions(vm.instructions)
		if frame.ip >= len(instructions) {
			if len(vm.frames) == 1 {
				return nil
			}
			return fmt.Errorf("function instruction pointer %d is outside bytecode", frame.ip)
		}
		ip := frame.ip
		op := code.Opcode(instructions[ip])
		definition, err := code.Lookup(op)
		if err != nil {
			return fmt.Errorf("instruction at %d: %w", ip, err)
		}
		operands, bytesRead, err := code.ReadOperands(definition, instructions[ip+1:])
		if err != nil {
			return fmt.Errorf("instruction at %d: %w", ip, err)
		}
		nextIP := ip + 1 + bytesRead
		frame.ip = nextIP

		switch op {
		case code.OpConstant:
			if err := vm.pushConstant(operands[0]); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv:
			if err := vm.executeBinaryOperation(op); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpEqual, code.OpNotEqual, code.OpGreaterThan, code.OpLessThan:
			if err := vm.executeComparison(op); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpMinus:
			if err := vm.executeMinusOperator(); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpBang:
			if err := vm.executeBangOperator(); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpToBoolean:
			value, err := vm.pop()
			if err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			if err := vm.push(&object.Boolean{Value: isTruthy(value)}); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpTrue:
			if err := vm.push(&object.Boolean{Value: true}); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpFalse:
			if err := vm.push(&object.Boolean{Value: false}); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpNull:
			if err := vm.push(&object.Null{}); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpPop:
			popped, err := vm.pop()
			if err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			vm.lastPopped = popped
		case code.OpJump:
			frame.ip = operands[0]
		case code.OpJumpNotTruthy:
			condition, err := vm.pop()
			if err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			if !isTruthy(condition) {
				frame.ip = operands[0]
			}
		case code.OpGetGlobal:
			if err := vm.checkGlobalIndex(operands[0]); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			value := vm.globals[operands[0]]
			if value == nil {
				return fmt.Errorf("instruction at %d: global index %d is uninitialized", ip, operands[0])
			}
			if err := vm.push(value); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpSetGlobal:
			if err := vm.checkGlobalIndex(operands[0]); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			value, err := vm.pop()
			if err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			if vm.globals[operands[0]] == nil {
				return fmt.Errorf("instruction at %d: global index %d is uninitialized", ip, operands[0])
			}
			vm.globals[operands[0]] = value
		case code.OpDefineGlobal:
			if err := vm.checkGlobalIndex(operands[0]); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			value, err := vm.pop()
			if err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			vm.globals[operands[0]] = value
		case code.OpGetLocal:
			cell, err := vm.localCell(frame, operands[0])
			if err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			if cell.Value == nil {
				return fmt.Errorf("instruction at %d: local index %d is uninitialized", ip, operands[0])
			}
			if err := vm.push(cell.Value); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpSetLocal:
			value, err := vm.pop()
			if err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			cell, err := vm.localCell(frame, operands[0])
			if err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			cell.Value = value
		case code.OpGetLocalRef:
			cell, err := vm.localCell(frame, operands[0])
			if err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			if err := vm.push(&object.CellReference{Cell: cell}); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpGetFree:
			cell, err := vm.freeCell(frame, operands[0])
			if err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			if cell.Value == nil {
				return fmt.Errorf("instruction at %d: free index %d is uninitialized", ip, operands[0])
			}
			if err := vm.push(cell.Value); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpGetFreeRef:
			cell, err := vm.freeCell(frame, operands[0])
			if err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			if err := vm.push(&object.CellReference{Cell: cell}); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpSetFree:
			value, err := vm.pop()
			if err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			cell, err := vm.freeCell(frame, operands[0])
			if err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			cell.Value = value
		case code.OpGetBuiltin:
			if operands[0] >= len(builtinNames) {
				return fmt.Errorf("instruction at %d: builtin index %d is out of range", ip, operands[0])
			}
			builtin := builtinFor(builtinNames[operands[0]], vm.output)
			if err := vm.push(builtin); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpClosure:
			if err := vm.pushClosure(operands[0], operands[1]); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpCurrentClosure:
			if frame.closure == nil {
				return fmt.Errorf("instruction at %d: no current closure", ip)
			}
			if err := vm.push(frame.closure); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpCall:
			if err := vm.executeCall(operands[0]); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpReturnValue:
			value, err := vm.pop()
			if err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
			if err := vm.returnFromFrame(value); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		case code.OpReturn:
			if err := vm.returnFromFrame(&object.Null{}); err != nil {
				return fmt.Errorf("instruction at %d: %w", ip, err)
			}
		default:
			return fmt.Errorf("instruction at %d: opcode %s is not implemented by the core VM", ip, definition.Name)
		}

		if frame.ip < 0 || frame.ip > len(instructions) {
			return fmt.Errorf("instruction at %d: jump target %d is outside bytecode", ip, frame.ip)
		}
	}
	return nil
}

func (vm *VM) LastPoppedStackElem() object.Object {
	return vm.lastPopped
}

func (vm *VM) checkGlobalIndex(index int) error {
	if index < 0 || index >= len(vm.globals) {
		return fmt.Errorf("global index %d is out of range", index)
	}
	return nil
}

func (vm *VM) localCell(frame *frame, index int) (*object.Cell, error) {
	if frame.closure == nil {
		return nil, fmt.Errorf("local access outside a function")
	}
	if index < 0 || index >= len(frame.locals) {
		return nil, fmt.Errorf("local index %d is out of range", index)
	}
	return frame.locals[index], nil
}

func (vm *VM) freeCell(frame *frame, index int) (*object.Cell, error) {
	if frame.closure == nil {
		return nil, fmt.Errorf("free variable access outside a function")
	}
	if index < 0 || index >= len(frame.closure.Free) {
		return nil, fmt.Errorf("free index %d is out of range", index)
	}
	if frame.closure.Free[index] == nil {
		return nil, fmt.Errorf("free index %d has a nil cell", index)
	}
	return frame.closure.Free[index], nil
}

func (vm *VM) pushClosure(constantIndex, freeCount int) error {
	if constantIndex < 0 || constantIndex >= len(vm.constants) {
		return fmt.Errorf("constant index %d is out of range", constantIndex)
	}
	function, ok := vm.constants[constantIndex].(*object.CompiledFunction)
	if !ok {
		return fmt.Errorf("constant index %d is not a compiled function", constantIndex)
	}
	if freeCount < 0 || freeCount > vm.sp {
		return fmt.Errorf("closure expects %d free value(s), stack has %d", freeCount, vm.sp)
	}

	free := make([]*object.Cell, freeCount)
	start := vm.sp - freeCount
	for i := 0; i < freeCount; i++ {
		value := vm.stack[start+i]
		if reference, ok := value.(*object.CellReference); ok {
			if reference.Cell == nil {
				return fmt.Errorf("free value %d has a nil cell", i)
			}
			free[i] = reference.Cell
		} else if value != nil {
			free[i] = &object.Cell{Value: value}
		} else {
			return fmt.Errorf("free value %d is nil", i)
		}
		vm.stack[start+i] = nil
	}
	vm.sp = start
	return vm.push(&object.Closure{Fn: function, Free: free})
}

func (vm *VM) executeCall(argumentCount int) error {
	calleeIndex := vm.sp - argumentCount - 1
	if argumentCount < 0 || calleeIndex < 0 {
		return fmt.Errorf("call stack underflow")
	}
	callee := vm.stack[calleeIndex]
	if callee == nil {
		return fmt.Errorf("callee is nil")
	}
	switch function := callee.(type) {
	case *object.Closure:
		if function.Fn == nil {
			return fmt.Errorf("cannot call closure without a compiled function")
		}
		if argumentCount != function.Fn.NumParameters {
			return fmt.Errorf("wrong number of arguments: got=%d, want=%d", argumentCount, function.Fn.NumParameters)
		}
		if len(vm.frames) >= maxFrames {
			return fmt.Errorf("maximum call frame depth exceeded (limit %d)", maxFrames-1)
		}
		if function.Fn.NumLocals < argumentCount {
			return fmt.Errorf("compiled function has fewer locals than parameters")
		}
		locals := make([]*object.Cell, function.Fn.NumLocals)
		for i := range locals {
			locals[i] = &object.Cell{}
		}
		for i := 0; i < argumentCount; i++ {
			locals[i].Value = vm.stack[calleeIndex+1+i]
		}
		vm.clearStack(calleeIndex)
		vm.frames = append(vm.frames, &frame{
			closure:     function,
			basePointer: calleeIndex,
			locals:      locals,
		})
		return nil
	case *object.Builtin:
		if function.Fn == nil {
			return fmt.Errorf("cannot call builtin without an implementation")
		}
		arguments := append([]object.Object(nil), vm.stack[calleeIndex+1:vm.sp]...)
		vm.clearStack(calleeIndex)
		result := function.Fn(arguments...)
		if result == nil {
			return fmt.Errorf("builtin returned a nil object")
		}
		if runtimeError, ok := result.(*object.Error); ok {
			return fmt.Errorf("%s", runtimeError.Inspect())
		}
		return vm.push(result)
	default:
		return fmt.Errorf("not a function: %s", callee.Type())
	}
}

func (vm *VM) clearStack(from int) {
	for i := from; i < vm.sp; i++ {
		vm.stack[i] = nil
	}
	vm.sp = from
}

func (vm *VM) returnFromFrame(value object.Object) error {
	if len(vm.frames) <= 1 {
		return fmt.Errorf("return instruction outside a function")
	}
	current := vm.frames[len(vm.frames)-1]
	vm.frames = vm.frames[:len(vm.frames)-1]
	vm.clearStack(current.basePointer)
	return vm.push(value)
}

func (vm *VM) pushConstant(index int) error {
	if index < 0 || index >= len(vm.constants) {
		return fmt.Errorf("constant index %d is out of range", index)
	}
	return vm.push(vm.constants[index])
}

func (vm *VM) push(value object.Object) error {
	if value == nil {
		return fmt.Errorf("cannot push a nil object")
	}
	if vm.sp >= len(vm.stack) {
		return fmt.Errorf("stack overflow")
	}
	vm.stack[vm.sp] = value
	vm.sp++
	return nil
}

func (vm *VM) pop() (object.Object, error) {
	if vm.sp == 0 {
		return nil, fmt.Errorf("stack underflow")
	}
	vm.sp--
	value := vm.stack[vm.sp]
	vm.stack[vm.sp] = nil
	return value, nil
}

func (vm *VM) executeBinaryOperation(op code.Opcode) error {
	right, err := vm.pop()
	if err != nil {
		return err
	}
	left, err := vm.pop()
	if err != nil {
		return err
	}

	if left.Type() == "INTEGER" && right.Type() == "INTEGER" {
		l, r := left.(*object.Integer), right.(*object.Integer)
		var result int64
		switch op {
		case code.OpAdd:
			result = l.Value + r.Value
		case code.OpSub:
			result = l.Value - r.Value
		case code.OpMul:
			result = l.Value * r.Value
		case code.OpDiv:
			if r.Value == 0 {
				return fmt.Errorf("division by zero")
			}
			result = l.Value / r.Value
		}
		return vm.push(&object.Integer{Value: result})
	}

	if op == code.OpAdd && left.Type() == "STRING" && right.Type() == "STRING" {
		l, r := left.(*object.String), right.(*object.String)
		return vm.push(&object.String{Value: l.Value + r.Value})
	}
	return fmt.Errorf("unsupported operands for %s: %s and %s", opcodeName(op), left.Type(), right.Type())
}

func (vm *VM) executeComparison(op code.Opcode) error {
	right, err := vm.pop()
	if err != nil {
		return err
	}
	left, err := vm.pop()
	if err != nil {
		return err
	}

	switch {
	case left.Type() == "INTEGER" && right.Type() == "INTEGER":
		l, r := left.(*object.Integer), right.(*object.Integer)
		switch op {
		case code.OpEqual:
			return vm.push(&object.Boolean{Value: l.Value == r.Value})
		case code.OpNotEqual:
			return vm.push(&object.Boolean{Value: l.Value != r.Value})
		case code.OpGreaterThan:
			return vm.push(&object.Boolean{Value: l.Value > r.Value})
		case code.OpLessThan:
			return vm.push(&object.Boolean{Value: l.Value < r.Value})
		}
	case op == code.OpEqual || op == code.OpNotEqual:
		equal := objectsEqual(left, right)
		if op == code.OpNotEqual {
			equal = !equal
		}
		return vm.push(&object.Boolean{Value: equal})
	}
	return fmt.Errorf("unsupported operands for %s: %s and %s", opcodeName(op), left.Type(), right.Type())
}

func (vm *VM) executeMinusOperator() error {
	value, err := vm.pop()
	if err != nil {
		return err
	}
	integer, ok := value.(*object.Integer)
	if !ok {
		return fmt.Errorf("unsupported operand for OpMinus: %s", value.Type())
	}
	return vm.push(&object.Integer{Value: -integer.Value})
}

func (vm *VM) executeBangOperator() error {
	value, err := vm.pop()
	if err != nil {
		return err
	}
	return vm.push(&object.Boolean{Value: !isTruthy(value)})
}

func isTruthy(value object.Object) bool {
	switch value := value.(type) {
	case *object.Null:
		return false
	case *object.Boolean:
		return value.Value
	default:
		return true
	}
}

func objectsEqual(left, right object.Object) bool {
	if left.Type() != right.Type() {
		return false
	}
	switch left := left.(type) {
	case *object.Integer:
		return left.Value == right.(*object.Integer).Value
	case *object.String:
		return left.Value == right.(*object.String).Value
	case *object.Boolean:
		return left.Value == right.(*object.Boolean).Value
	case *object.Null:
		return true
	default:
		return left == right
	}
}

func opcodeName(op code.Opcode) string {
	definition, err := code.Lookup(op)
	if err != nil {
		return fmt.Sprintf("opcode %d", op)
	}
	return definition.Name
}
