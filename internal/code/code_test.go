package code

import (
	"slices"
	"strings"
	"testing"
)

func TestMakeAndReadOperands(t *testing.T) {
	tests := []struct {
		name     string
		op       Opcode
		operands []int
	}{
		{name: "constant", op: OpConstant, operands: []int{65534}},
		{name: "local", op: OpGetLocal, operands: []int{255}},
		{name: "closure", op: OpClosure, operands: []int{513, 7}},
		{name: "no operands", op: OpAdd},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			instruction, err := Make(test.op, test.operands...)
			if err != nil {
				t.Fatal(err)
			}
			definition, err := Lookup(test.op)
			if err != nil {
				t.Fatal(err)
			}
			got, bytesRead, err := ReadOperands(definition, instruction[1:])
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, test.operands) {
				t.Fatalf("operands = %v, want %v", got, test.operands)
			}
			if want := len(instruction) - 1; bytesRead != want {
				t.Fatalf("bytes read = %d, want %d", bytesRead, want)
			}
		})
	}
}

func TestMakeRejectsInvalidOperands(t *testing.T) {
	tests := []struct {
		name     string
		op       Opcode
		operands []int
	}{
		{name: "unknown opcode", op: Opcode(255)},
		{name: "missing operand", op: OpConstant},
		{name: "extra operand", op: OpAdd, operands: []int{1}},
		{name: "negative operand", op: OpConstant, operands: []int{-1}},
		{name: "one-byte overflow", op: OpCall, operands: []int{256}},
		{name: "two-byte overflow", op: OpJump, operands: []int{65536}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Make(test.op, test.operands...); err == nil {
				t.Fatal("expected operand validation error")
			}
		})
	}
}

func TestReadOperandsRejectsTruncatedInstruction(t *testing.T) {
	definition, err := Lookup(OpClosure)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadOperands(definition, []byte{0, 1}); err == nil {
		t.Fatal("expected truncated operand error")
	}
}

func TestDisassembleInstructions(t *testing.T) {
	constant, err := Make(OpConstant, 2)
	if err != nil {
		t.Fatal(err)
	}
	call, err := Make(OpCall, 1)
	if err != nil {
		t.Fatal(err)
	}
	instructions := append(Instructions{}, constant...)
	instructions = append(instructions, call...)

	got, err := instructions.Disassemble()
	if err != nil {
		t.Fatal(err)
	}
	want := "0000 OpConstant 2\n0003 OpCall 1\n"
	if got != want {
		t.Fatalf("disassembly = %q, want %q", got, want)
	}
}

func TestDisassembleReportsMalformedBytecode(t *testing.T) {
	if _, err := (Instructions{255}).Disassemble(); err == nil || !strings.Contains(err.Error(), "unknown opcode") {
		t.Fatalf("expected unknown opcode error, got %v", err)
	}
	if _, err := (Instructions{byte(OpConstant)}).Disassemble(); err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("expected truncated operand error, got %v", err)
	}
}
