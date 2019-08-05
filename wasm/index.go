// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wasm

import (
	"fmt"
	"reflect"
)

type SizeOverFlowError struct {
	ImmType string
	Size    uint64
	Max     uint64
}

func (e SizeOverFlowError) Error() string {
	return fmt.Sprintf("validate: %s size overflow (%v), max (%v)", e.ImmType, e.Size, e.Max)
}

type InvalidTableIndexError uint32

func (e InvalidTableIndexError) Error() string {
	return fmt.Sprintf("wasm: Invalid table to table index space: %d", uint32(e))
}

type InvalidValueTypeInitExprError struct {
	Wanted reflect.Kind
	Got    reflect.Kind
}

func (e InvalidValueTypeInitExprError) Error() string {
	return fmt.Sprintf("wasm: Wanted initializer expression to return %v value, got %v", e.Wanted, e.Got)
}

type InvalidLinearMemoryIndexError uint32

func (e InvalidLinearMemoryIndexError) Error() string {
	return fmt.Sprintf("wasm: Invalid linear memory index: %d", uint32(e))
}

// Functions for populating and looking up entries in a module's index space.
// More info: http://webassembly.org/docs/modules/#function-index-space

func (m *Module) populateFunctions() error {
	if m.Types == nil || m.Function == nil {
		return nil
	}

	for codeIndex, typeIndex := range m.Function.Types {
		if int(typeIndex) >= len(m.Types.Entries) {
			return InvalidFunctionIndexError(typeIndex)
		}

		fn := Function{
			Sig:  &m.Types.Entries[typeIndex],
			Body: &m.Code.Bodies[codeIndex],
		}

		m.FunctionIndexSpace = append(m.FunctionIndexSpace, fn)
	}

	funcs := make([]uint32, 0, len(m.Function.Types)+len(m.imports.Funcs))
	funcs = append(funcs, m.imports.Funcs...)
	funcs = append(funcs, m.Function.Types...)
	m.Function.Types = funcs
	return nil
}

// GetFunction returns a *Function, based on the function's index in
// the function index space. Returns nil when the index is invalid
func (m *Module) GetFunction(i int) *Function {
	if i >= len(m.FunctionIndexSpace) || i < 0 {
		return nil
	}

	return &m.FunctionIndexSpace[i]
}

func (m *Module) populateGlobals() error {
	if m.Global == nil {
		return nil
	}

	m.GlobalIndexSpace = append(m.GlobalIndexSpace, m.Global.Globals...)
	logger.Printf("There are %d entries in the global index spaces.", len(m.GlobalIndexSpace))
	return nil
}

// GetGlobal returns a *GlobalEntry, based on the global index space.
// Returns nil when the index is invalid
func (m *Module) GetGlobal(i int) *GlobalEntry {
	if i >= len(m.GlobalIndexSpace) || i < 0 {
		return nil
	}

	return &m.GlobalIndexSpace[i]
}

func (m *Module) populateTables() error {
	if m.Table == nil || len(m.Table.Entries) == 0 || m.Elements == nil || len(m.Elements.Entries) == 0 {
		return nil
	}

	for _, elem := range m.Elements.Entries {
		// the MVP dictates that index should always be zero, we should
		// probably check this
		if elem.Index >= uint32(len(m.TableIndexSpace)) {
			return InvalidTableIndexError(elem.Index)
		}

		val, err := m.ExecInitExpr(elem.Offset)
		if err != nil {
			return err
		}
		off, ok := val.(int32)
		if !ok {
			return InvalidValueTypeInitExprError{reflect.Int32, reflect.TypeOf(val).Kind()}
		}
		offset := uint32(off)

		table := m.TableIndexSpace[elem.Index]
		//use uint64 to avoid overflow
		if uint64(offset)+uint64(len(elem.Elems)) > uint64(len(table)) {
			if uint64(offset)+uint64(len(elem.Elems)) > uint64(m.Table.Entries[elem.Index].Limits.Maximum) {
				return SizeOverFlowError{"Table", uint64(offset) + uint64(len(elem.Elems)), uint64(m.Table.Entries[elem.Index].Limits.Maximum)}
			}
			data := make([]uint32, uint64(offset)+uint64(len(elem.Elems)))
			copy(data[offset:], elem.Elems)
			copy(data, table)
			m.TableIndexSpace[elem.Index] = data
		} else {
			copy(table[offset:], elem.Elems)
		}
	}

	logger.Printf("There are %d entries in the table index space.", len(m.TableIndexSpace))
	return nil
}

// GetTableElement returns an element from the tableindex space indexed
// by the integer index. It returns an error if index is invalid.
func (m *Module) GetTableElement(index int) (uint32, error) {
	if index >= len(m.TableIndexSpace[0]) {
		return 0, InvalidTableIndexError(index)
	}

	return m.TableIndexSpace[0][index], nil
}

func (m *Module) populateLinearMemory() error {
	if m.Data == nil || len(m.Data.Entries) == 0 {
		return nil
	}
	// each module can only have a single linear memory in the MVP

	for _, entry := range m.Data.Entries {
		if entry.Index != 0 {
			return InvalidLinearMemoryIndexError(entry.Index)
		}

		val, err := m.ExecInitExpr(entry.Offset)
		if err != nil {
			return err
		}
		off, ok := val.(int32)
		if !ok {
			return InvalidValueTypeInitExprError{reflect.Int32, reflect.TypeOf(val).Kind()}
		}
		offset := uint32(off)

		memory := m.LinearMemoryIndexSpace[entry.Index]
		if uint64(offset)+uint64(len(entry.Data)) > uint64(len(memory)) {
			bound := uint64(m.Memory.Entries[entry.Index].Limits.Maximum * uint32(WASM_PAGE_SIZE))
			if uint64(offset)+uint64(len(entry.Data)) > bound {
				return SizeOverFlowError{"Memory", uint64(offset) + uint64(len(entry.Data)), bound}
			}
			data := make([]byte, uint64(offset)+uint64(len(entry.Data)))
			copy(data, memory)
			copy(data[offset:], entry.Data)
			m.LinearMemoryIndexSpace[int(entry.Index)] = data
		} else {
			copy(memory[offset:], entry.Data)
		}
	}

	return nil
}

func (m *Module) GetLinearMemoryData(index int) (byte, error) {
	if index >= len(m.LinearMemoryIndexSpace[0]) {
		return 0, InvalidLinearMemoryIndexError(uint32(index))

	}

	return m.LinearMemoryIndexSpace[0][index], nil
}
