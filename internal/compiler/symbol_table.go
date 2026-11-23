package compiler

import "fmt"

type symbolScope string

const (
	globalScope   symbolScope = "GLOBAL"
	localScope    symbolScope = "LOCAL"
	builtinScope  symbolScope = "BUILTIN"
	freeScope     symbolScope = "FREE"
	functionScope symbolScope = "FUNCTION"
)

type symbol struct {
	name  string
	scope symbolScope
	index int
}

type symbolTable struct {
	outer          *symbolTable
	store          map[string]symbol
	numDefinitions int
	freeSymbols    []symbol
}

func newSymbolTable() *symbolTable {
	return &symbolTable{store: make(map[string]symbol)}
}

func newEnclosedSymbolTable(outer *symbolTable) *symbolTable {
	return &symbolTable{outer: outer, store: make(map[string]symbol)}
}

func (table *symbolTable) defineGlobal(name string, index int) symbol {
	symbol := symbol{name: name, scope: globalScope, index: index}
	table.store[name] = symbol
	return symbol
}

func (table *symbolTable) defineLocal(name string) (symbol, error) {
	if existing, ok := table.store[name]; ok && existing.scope != functionScope {
		return existing, nil
	}
	if table.numDefinitions > 0xff {
		return symbol{}, fmt.Errorf("local variable table exceeds 8-bit index limit")
	}
	symbol := symbol{name: name, scope: localScope, index: table.numDefinitions}
	table.store[name] = symbol
	table.numDefinitions++
	return symbol, nil
}

func (table *symbolTable) defineBuiltin(name string, index int) {
	table.store[name] = symbol{name: name, scope: builtinScope, index: index}
}

func (table *symbolTable) defineFunctionName(name string) {
	table.store[name] = symbol{name: name, scope: functionScope}
}

func (table *symbolTable) resolve(name string) (symbol, bool) {
	if value, ok := table.store[name]; ok {
		return value, true
	}
	if table.outer == nil {
		return symbol{}, false
	}
	value, ok := table.outer.resolve(name)
	if !ok || value.scope == globalScope || value.scope == builtinScope {
		return value, ok
	}
	return table.defineFree(value), true
}

func (table *symbolTable) defineFree(original symbol) symbol {
	table.freeSymbols = append(table.freeSymbols, original)
	value := symbol{name: original.name, scope: freeScope, index: len(table.freeSymbols) - 1}
	table.store[original.name] = value
	return value
}
