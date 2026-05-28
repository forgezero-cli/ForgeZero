package gloria

import (
	"fmt"
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
		return 0, fmt.Errorf("variable '%s' is not declared! Use 'let %s = ...'", name, name)
	}
	return s.vars[name], nil
}

func (s *compilerState) declareAndAlloc(name string) (int, error) {
	if s.declared[name] {
		return 0, fmt.Errorf("variable '%s' redeclared", name)
	}
	s.stackOffset -= 8
	s.vars[name] = s.stackOffset
	s.declared[name] = true
	return s.stackOffset, nil
}
