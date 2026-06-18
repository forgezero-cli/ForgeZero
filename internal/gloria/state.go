package gloria

import (
	"errors"
)

type compilerState struct {
	vars        map[string]int
	declared    map[string]bool
	stackOffset int
}

func newCompilerState() *compilerState {
	return &compilerState{
		vars:        make(map[string]int),
		declared:    make(map[string]bool),
		stackOffset: 0,
	}
}

func (s *compilerState) getStackOffset(name string) (int, error) {
	if !s.declared[name] {
		return 0, errors.New("variable '" + name + "' is not declared! Use 'let " + name + " = ...'")
	}
	return s.vars[name], nil
}

func (s *compilerState) declareAndAlloc(name string) (int, error) {
	if s.declared[name] {
		return 0, errors.New("variable '" + name + "' redeclared")
	}
	s.stackOffset -= 8
	s.vars[name] = s.stackOffset
	s.declared[name] = true
	return s.stackOffset, nil
}