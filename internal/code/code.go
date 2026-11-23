package code

import (
	"encoding/binary"
	"fmt"
	"strings"

	"idiotic/internal/object"
)

type Instructions []byte

type Bytecode struct {
	Instructions Instructions
	Constants    []object.Object
	NumGlobals   int
}

func Make(op Opcode, operands ...int) ([]byte, error) {
	definition, err := Lookup(op)
	if err != nil {
		return nil, err
	}
	if len(operands) != len(definition.OperandWidths) {
		return nil, fmt.Errorf("%s expects %d operands, got %d", definition.Name, len(definition.OperandWidths), len(operands))
	}

	length := 1
	for _, width := range definition.OperandWidths {
		if width != 1 && width != 2 {
			return nil, fmt.Errorf("%s has unsupported operand width %d", definition.Name, width)
		}
		length += width
	}

	instruction := make([]byte, length)
	instruction[0] = byte(op)
	offset := 1
	for i, operand := range operands {
		width := definition.OperandWidths[i]
		if operand < 0 || (width == 1 && operand > 0xff) || (width == 2 && operand > 0xffff) {
			return nil, fmt.Errorf("%s operand %d does not fit in %d byte(s)", definition.Name, operand, width)
		}
		switch width {
		case 1:
			instruction[offset] = byte(operand)
		case 2:
			binary.BigEndian.PutUint16(instruction[offset:], uint16(operand))
		}
		offset += width
	}
	return instruction, nil
}

func ReadOperands(definition *Definition, instruction []byte) ([]int, int, error) {
	if definition == nil {
		return nil, 0, fmt.Errorf("cannot read operands without an instruction definition")
	}
	operands := make([]int, len(definition.OperandWidths))
	offset := 0
	for i, width := range definition.OperandWidths {
		if width != 1 && width != 2 {
			return nil, offset, fmt.Errorf("%s has unsupported operand width %d", definition.Name, width)
		}
		if len(instruction)-offset < width {
			return nil, offset, fmt.Errorf("%s operand %d is truncated", definition.Name, i)
		}
		switch width {
		case 1:
			operands[i] = int(instruction[offset])
		case 2:
			operands[i] = int(binary.BigEndian.Uint16(instruction[offset:]))
		}
		offset += width
	}
	return operands, offset, nil
}

func (instructions Instructions) Disassemble() (string, error) {
	var out strings.Builder
	for offset := 0; offset < len(instructions); {
		op := Opcode(instructions[offset])
		definition, err := Lookup(op)
		if err != nil {
			return "", fmt.Errorf("instruction at %d: %w", offset, err)
		}
		operands, bytesRead, err := ReadOperands(definition, instructions[offset+1:])
		if err != nil {
			return "", fmt.Errorf("instruction at %d: %w", offset, err)
		}
		if _, err := fmt.Fprintf(&out, "%04d %s", offset, definition.Name); err != nil {
			return "", err
		}
		for _, operand := range operands {
			if _, err := fmt.Fprintf(&out, " %d", operand); err != nil {
				return "", err
			}
		}
		out.WriteByte('\n')
		offset += 1 + bytesRead
	}
	return out.String(), nil
}

func (instructions Instructions) String() string {
	disassembly, err := instructions.Disassemble()
	if err != nil {
		return "ERROR: " + err.Error()
	}
	return disassembly
}
