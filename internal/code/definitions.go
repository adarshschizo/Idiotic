package code

import "fmt"

type Opcode byte

const (
	OpConstant Opcode = iota
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpPop
	OpTrue
	OpFalse
	OpEqual
	OpNotEqual
	OpGreaterThan
	OpLessThan
	OpMinus
	OpBang
	OpJumpNotTruthy
	OpJump
	OpNull
	OpGetGlobal
	OpSetGlobal
	OpCall
	OpReturnValue
	OpReturn
	OpGetLocal
	OpSetLocal
	OpGetBuiltin
	OpClosure
	OpGetFree
	OpCurrentClosure
	OpToBoolean
	OpDefineGlobal
	OpGetLocalRef
	OpGetFreeRef
	OpSetFree
)

type Definition struct {
	Name          string
	OperandWidths []int
}

var definitions = map[Opcode]Definition{
	OpConstant:       {Name: "OpConstant", OperandWidths: []int{2}},
	OpAdd:            {Name: "OpAdd"},
	OpSub:            {Name: "OpSub"},
	OpMul:            {Name: "OpMul"},
	OpDiv:            {Name: "OpDiv"},
	OpPop:            {Name: "OpPop"},
	OpTrue:           {Name: "OpTrue"},
	OpFalse:          {Name: "OpFalse"},
	OpEqual:          {Name: "OpEqual"},
	OpNotEqual:       {Name: "OpNotEqual"},
	OpGreaterThan:    {Name: "OpGreaterThan"},
	OpLessThan:       {Name: "OpLessThan"},
	OpMinus:          {Name: "OpMinus"},
	OpBang:           {Name: "OpBang"},
	OpJumpNotTruthy:  {Name: "OpJumpNotTruthy", OperandWidths: []int{2}},
	OpJump:           {Name: "OpJump", OperandWidths: []int{2}},
	OpNull:           {Name: "OpNull"},
	OpGetGlobal:      {Name: "OpGetGlobal", OperandWidths: []int{2}},
	OpSetGlobal:      {Name: "OpSetGlobal", OperandWidths: []int{2}},
	OpCall:           {Name: "OpCall", OperandWidths: []int{1}},
	OpReturnValue:    {Name: "OpReturnValue"},
	OpReturn:         {Name: "OpReturn"},
	OpGetLocal:       {Name: "OpGetLocal", OperandWidths: []int{1}},
	OpSetLocal:       {Name: "OpSetLocal", OperandWidths: []int{1}},
	OpGetBuiltin:     {Name: "OpGetBuiltin", OperandWidths: []int{1}},
	OpClosure:        {Name: "OpClosure", OperandWidths: []int{2, 1}},
	OpGetFree:        {Name: "OpGetFree", OperandWidths: []int{1}},
	OpCurrentClosure: {Name: "OpCurrentClosure"},
	OpToBoolean:      {Name: "OpToBoolean"},
	OpDefineGlobal:   {Name: "OpDefineGlobal", OperandWidths: []int{2}},
	OpGetLocalRef:    {Name: "OpGetLocalRef", OperandWidths: []int{1}},
	OpGetFreeRef:     {Name: "OpGetFreeRef", OperandWidths: []int{1}},
	OpSetFree:        {Name: "OpSetFree", OperandWidths: []int{1}},
}

func Lookup(op Opcode) (*Definition, error) {
	definition, ok := definitions[op]
	if !ok {
		return nil, fmt.Errorf("unknown opcode: %d", op)
	}
	definition.OperandWidths = append([]int(nil), definition.OperandWidths...)
	return &definition, nil
}
