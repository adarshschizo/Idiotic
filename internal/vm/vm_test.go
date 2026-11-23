package vm

import (
	"strings"
	"testing"

	"idiotic/internal/code"
	"idiotic/internal/object"
)

type instructionSpec struct {
	op       code.Opcode
	operands []int
}

func TestVMArithmeticAndStringOperations(t *testing.T) {
	tests := []struct {
		name      string
		constants []object.Object
		ops       []instructionSpec
		want      string
	}{
		{
			name:      "integer addition",
			constants: []object.Object{&object.Integer{Value: 1}, &object.Integer{Value: 2}},
			ops:       []instructionSpec{{code.OpConstant, []int{0}}, {code.OpConstant, []int{1}}, {code.OpAdd, nil}, {code.OpPop, nil}},
			want:      "3",
		},
		{
			name:      "string concatenation",
			constants: []object.Object{&object.String{Value: "hello "}, &object.String{Value: "world"}},
			ops:       []instructionSpec{{code.OpConstant, []int{0}}, {code.OpConstant, []int{1}}, {code.OpAdd, nil}, {code.OpPop, nil}},
			want:      "hello world",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			instructions := makeInstructions(t, test.ops...)
			machine := New(&code.Bytecode{Instructions: instructions, Constants: test.constants})
			if err := machine.Run(); err != nil {
				t.Fatal(err)
			}
			if got := machine.LastPoppedStackElem().Inspect(); got != test.want {
				t.Fatalf("result = %q, want %q", got, test.want)
			}
		})
	}
}

func TestVMComparisonAndUnaryOperations(t *testing.T) {
	tests := []struct {
		name      string
		constants []object.Object
		ops       []instructionSpec
		want      string
	}{
		{
			name:      "less than",
			constants: []object.Object{&object.Integer{Value: 2}, &object.Integer{Value: 3}},
			ops:       []instructionSpec{{code.OpConstant, []int{0}}, {code.OpConstant, []int{1}}, {code.OpLessThan, nil}, {code.OpPop, nil}},
			want:      "true",
		},
		{
			name: "equal booleans",
			ops:  []instructionSpec{{code.OpTrue, nil}, {code.OpTrue, nil}, {code.OpEqual, nil}, {code.OpPop, nil}},
			want: "true",
		},
		{
			name: "negation",
			ops:  []instructionSpec{{code.OpFalse, nil}, {code.OpBang, nil}, {code.OpPop, nil}},
			want: "true",
		},
		{
			name:      "integer minus",
			constants: []object.Object{&object.Integer{Value: 7}},
			ops:       []instructionSpec{{code.OpConstant, []int{0}}, {code.OpMinus, nil}, {code.OpPop, nil}},
			want:      "-7",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			machine := New(&code.Bytecode{Instructions: makeInstructions(t, test.ops...), Constants: test.constants})
			if err := machine.Run(); err != nil {
				t.Fatal(err)
			}
			if got := machine.LastPoppedStackElem().Inspect(); got != test.want {
				t.Fatalf("result = %q, want %q", got, test.want)
			}
		})
	}
}

func TestVMConditionalJumps(t *testing.T) {
	tests := []struct {
		name string
		op   code.Opcode
		want string
	}{
		{name: "truthy branch", op: code.OpTrue, want: "1"},
		{name: "falsey branch", op: code.OpFalse, want: "2"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			instructions := makeInstructions(t,
				instructionSpec{test.op, nil},
				instructionSpec{code.OpJumpNotTruthy, []int{10}},
				instructionSpec{code.OpConstant, []int{0}},
				instructionSpec{code.OpJump, []int{13}},
				instructionSpec{code.OpConstant, []int{1}},
				instructionSpec{code.OpPop, nil},
			)
			machine := New(&code.Bytecode{
				Instructions: instructions,
				Constants: []object.Object{
					&object.Integer{Value: 1},
					&object.Integer{Value: 2},
				},
			})
			if err := machine.Run(); err != nil {
				t.Fatal(err)
			}
			if got := machine.LastPoppedStackElem().Inspect(); got != test.want {
				t.Fatalf("result = %q, want %q", got, test.want)
			}
		})
	}
}

func TestVMGlobals(t *testing.T) {
	machine := New(&code.Bytecode{
		Instructions: makeInstructions(t,
			instructionSpec{code.OpConstant, []int{0}},
			instructionSpec{code.OpDefineGlobal, []int{5}},
			instructionSpec{code.OpGetGlobal, []int{5}},
			instructionSpec{code.OpPop, nil},
		),
		Constants: []object.Object{&object.Integer{Value: 42}},
	})
	if err := machine.Run(); err != nil {
		t.Fatal(err)
	}
	if got := machine.LastPoppedStackElem().Inspect(); got != "42" {
		t.Fatalf("result = %q, want 42", got)
	}
}

func TestVMUsesDeclaredGlobalCountAndChecksBounds(t *testing.T) {
	bytecode := &code.Bytecode{
		Instructions: makeInstructions(t, instructionSpec{code.OpGetGlobal, []int{1}}),
		NumGlobals:   1,
	}
	machine := New(bytecode)
	if got := len(machine.globals); got != 1 {
		t.Fatalf("allocated %d global slots, want 1", got)
	}
	if err := machine.Run(); err == nil || !strings.Contains(err.Error(), "global index 1 is out of range") {
		t.Fatalf("error = %v, want global bounds error", err)
	}
}

func TestVMReportsInvalidBytecodeAndStackErrors(t *testing.T) {
	tests := []struct {
		name         string
		instructions code.Instructions
		constants    []object.Object
		wantError    string
	}{
		{name: "stack underflow", instructions: makeInstructions(t, instructionSpec{code.OpAdd, nil}), wantError: "stack underflow"},
		{name: "constant out of range", instructions: makeInstructions(t, instructionSpec{code.OpConstant, []int{1}}), wantError: "constant index 1"},
		{name: "uninitialized global", instructions: makeInstructions(t, instructionSpec{code.OpGetGlobal, []int{2}}), wantError: "global index 2 is uninitialized"},
		{name: "invalid jump", instructions: makeInstructions(t, instructionSpec{code.OpJump, []int{20}}), wantError: "jump target 20"},
		{name: "truncated operand", instructions: code.Instructions{byte(code.OpConstant)}, wantError: "truncated"},
		{name: "unknown opcode", instructions: code.Instructions{255}, wantError: "unknown opcode"},
		{name: "call stack underflow", instructions: makeInstructions(t, instructionSpec{code.OpCall, []int{0}}), wantError: "call stack underflow"},
		{name: "division by zero", instructions: makeInstructions(t, instructionSpec{code.OpConstant, []int{0}}, instructionSpec{code.OpConstant, []int{1}}, instructionSpec{code.OpDiv, nil}), constants: []object.Object{&object.Integer{Value: 1}, &object.Integer{Value: 0}}, wantError: "division by zero"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			machine := New(&code.Bytecode{Instructions: test.instructions, Constants: test.constants})
			err := machine.Run()
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %v, want text %q", err, test.wantError)
			}
		})
	}
}

func makeInstructions(t *testing.T, ops ...instructionSpec) code.Instructions {
	t.Helper()
	var instructions code.Instructions
	for _, op := range ops {
		instruction, err := code.Make(op.op, op.operands...)
		if err != nil {
			t.Fatal(err)
		}
		instructions = append(instructions, instruction...)
	}
	return instructions
}
