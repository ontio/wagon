// Copyright 2018 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/go-interpreter/wagon/wasm"
	"github.com/stretchr/testify/assert"
)

func TestHostCall(t *testing.T) {
	const secretValue = 0xdeadbeef

	var secretVariable int

	// a host function that can be called by WASM code.
	testHostFunction := func(proc *Process) {
		secretVariable = secretValue
	}

	m := wasm.NewModule()
	m.Start = nil

	// A function signature. Both the host and WASM function
	// have the same signature.
	fsig := wasm.FunctionSig{
		Form:        0,
		ParamTypes:  []wasm.ValueType{},
		ReturnTypes: []wasm.ValueType{},
	}

	// List of all function types available in this module.
	// There is only one: (func [] -> [])
	m.Types = &wasm.SectionTypes{
		Entries: []wasm.FunctionSig{fsig, fsig},
	}

	m.Function = &wasm.SectionFunctions{
		Types: []uint32{0, 0},
	}

	// The body of the start function, that should only
	// call the host function
	fb := wasm.FunctionBody{
		Module: m,
		Locals: []wasm.LocalEntry{},
		// code should disassemble to:
		// call 1 (which is host)
		// end
		Code: []byte{0x02, 0x00, 0x10, 0x01, 0x0b},
	}

	// There was no call to `ReadModule` so this part emulates
	// how the module object would look like if the function
	// had been called.
	m.FunctionIndexSpace = []wasm.Function{
		{
			Sig:  &fsig,
			Body: &fb,
		},
		{
			Sig:  &fsig,
			Host: reflect.ValueOf(testHostFunction),
		},
	}

	m.Code = &wasm.SectionCode{
		Bodies: []wasm.FunctionBody{fb},
	}

	// Once called, NewVM will execute the module's main
	// function.
	vm, err := NewVM(m, math.MaxUint64)
	if err != nil {
		fmt.Printf("error is %s\n", err.Error())
		t.Fatalf("Error creating VM: %v", vm)
	}
	GasLimit := uint64(1000000)
	ExecStep := uint64(math.MaxUint64)
	vm.ExecMetrics = &Gas{GasPrice: 500, GasLimit: &GasLimit, GasFactor: 5, ExecStep: &ExecStep}
	vm.CallStackDepth = 1

	vm.ExecCode(0)
	if len(vm.funcs) < 1 {
		t.Fatalf("Need at least a start function!")
	}

	// Only one entry, which should be a function
	if secretVariable != secretValue {
		t.Fatalf("x is %d instead of %d", secretVariable, secretValue)
	}
}

var moduleCallHost = []byte{
	0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00, 0x01, 0x1A, 0x06, 0x60, 0x01, 0x7F, 0x00, 0x60,
	0x01, 0x7F, 0x01, 0x7F, 0x60, 0x00, 0x01, 0x7F, 0x60, 0x00, 0x00, 0x60, 0x00, 0x01, 0x7C, 0x60,
	0x01, 0x7F, 0x01, 0x7F, 0x02, 0x0F, 0x01, 0x03, 0x65, 0x6E, 0x76, 0x07, 0x5F, 0x6E, 0x61, 0x74,
	0x69, 0x76, 0x65, 0x00, 0x05, 0x03, 0x02, 0x01, 0x02, 0x04, 0x04, 0x01, 0x70, 0x00, 0x02, 0x06,
	0x10, 0x03, 0x7F, 0x01, 0x41, 0x00, 0x0B, 0x7F, 0x01, 0x41, 0x00, 0x0B, 0x7F, 0x00, 0x41, 0x01,
	0x0B, 0x07, 0x09, 0x01, 0x05, 0x5F, 0x6D, 0x61, 0x69, 0x6E, 0x00, 0x01, 0x09, 0x01, 0x00, 0x0A,
	0x08, 0x01, 0x06, 0x00, 0x41, 0x00, 0x10, 0x00, 0x0B,
}

func add3(proc *Process, x int32) int32 {
	return x + 3
}

func importer(name string, f func(*Process, int32) int32) (*wasm.Module, error) {
	m := wasm.NewModule()
	m.Types = &wasm.SectionTypes{
		// List of all function types available in this module.
		// There is only one: (func [int32] -> [int32])
		Entries: []wasm.FunctionSig{
			{
				Form:        0,
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
		},
	}
	m.FunctionIndexSpace = []wasm.Function{
		{
			Sig:  &m.Types.Entries[0],
			Host: reflect.ValueOf(f),
			Body: &wasm.FunctionBody{},
		},
	}
	m.Export = &wasm.SectionExports{
		Entries: map[string]wasm.ExportEntry{
			"_native": {
				FieldStr: "_naive",
				Kind:     wasm.ExternalFunction,
				Index:    0,
			},
		},
	}

	return m, nil
}

func invalidAdd3(x int32) int32 {
	return x + 3
}

func invalidImporter(name string) (*wasm.Module, error) {
	m := wasm.NewModule()
	m.Types = &wasm.SectionTypes{
		// List of all function types available in this module.
		// There is only one: (func [int32] -> [int32])
		Entries: []wasm.FunctionSig{
			{
				Form:        0,
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
		},
	}
	m.FunctionIndexSpace = []wasm.Function{
		{
			Sig:  &m.Types.Entries[0],
			Host: reflect.ValueOf(invalidAdd3),
			Body: &wasm.FunctionBody{},
		},
	}
	m.Export = &wasm.SectionExports{
		Entries: map[string]wasm.ExportEntry{
			"_native": {
				FieldStr: "_naive",
				Kind:     wasm.ExternalFunction,
				Index:    0,
			},
		},
	}

	return m, nil
}

func TestHostSymbolCall(t *testing.T) {
	m, err := wasm.ReadModule(bytes.NewReader(moduleCallHost), func(n string) (*wasm.Module, error) { return importer(n, add3) })
	if err != nil {
		t.Fatalf("Could not read module: %v", err)
	}
	vm, err := NewVM(m, math.MaxUint64)
	if err != nil {
		t.Fatalf("Could not instantiate vm: %v", err)
	}
	GasLimit := uint64(1000000)
	ExecStep := uint64(math.MaxUint64)
	vm.ExecMetrics = &Gas{GasPrice: 500, GasLimit: &GasLimit, GasFactor: 5, ExecStep: &ExecStep}
	vm.CallStackDepth = 1

	rtrns, err := vm.ExecCode(1)
	if err != nil {
		t.Fatalf("Error executing the default function: %v", err)
	}
	if int(rtrns.(uint32)) != 3 {
		t.Fatalf("Did not get the right value. Got %d, wanted %d", rtrns, 3)
	}
}

func TestGoFunctionCallChecksForFirstArgument(t *testing.T) {
	m, err := wasm.ReadModule(bytes.NewReader(moduleCallHost), invalidImporter)
	if err != nil {
		t.Fatalf("Could not read module: %v", err)
	}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("This code should have panicked.")
		} else {
			if r != "exec: the first argument of a host function was int32, expected ptr" {
				t.Errorf("This should have panicked because of the wrong type being used as a first argument, and it panicked because of %v", r)
			}
		}
	}()
	vm, err := NewVM(m, math.MaxUint64)
	if err != nil {
		t.Fatalf("Could not instantiate vm: %v", err)
	}
	GasLimit := uint64(1000000)
	ExecStep := uint64(math.MaxUint64)
	vm.ExecMetrics = &Gas{GasPrice: 500, GasLimit: &GasLimit, GasFactor: 5, ExecStep: &ExecStep}
	vm.CallStackDepth = 10000

	_, err = vm.ExecCode(1)
	if err != nil {
		t.Fatalf("Error executing the default function: %v", err)
	}
}

func terminate(proc *Process, x int32) int32 {
	proc.Terminate()
	return 3
}

func TestHostTerminate(t *testing.T) {
	m, err := wasm.ReadModule(bytes.NewReader(moduleCallHost), func(n string) (*wasm.Module, error) { return importer(n, terminate) })
	if err != nil {
		t.Fatalf("Could not read module: %v", err)
	}
	vm, err := NewVM(m, math.MaxUint64)
	if err != nil {
		t.Fatalf("Could not instantiate vm: %v", err)
	}
	GasLimit := uint64(1000000)
	ExecStep := uint64(math.MaxUint64)
	vm.ExecMetrics = &Gas{GasPrice: 500, GasLimit: &GasLimit, GasFactor: 5, ExecStep: &ExecStep}
	vm.CallStackDepth = 1

	_, err = vm.ExecCode(1)
	if err != nil {
		t.Fatalf("Error executing the default function: %v", err)
	}
	if vm.abort == false || vm.ctx.pc > 0x13 {
		t.Fatalf("Terminate did not abort execution: abort=%v, pc=%#x", vm.abort, vm.ctx.pc)
	}
}

//
func TestInfiniteRecursion(t *testing.T) {
	/*
		(module
		  (type $t0 (func (param i32) (result i32)))
		  (func $invoke (export "invoke") (type $t0) (param $p0 i32) (result i32)
		    get_local $p0
		    call $invoke))
	*/
	byteCode, err := hex.DecodeString("0061736d0100000001060160017f017f03020100070a0106696e766f6b6500000a08010600200010000b")
	if err != nil {
		panic(err)
	}
	m, err := wasm.ReadModule(bytes.NewReader(byteCode), func(name string) (*wasm.Module, error) {
		return nil, fmt.Errorf("module %q unknown", name)
	})
	if err != nil {
		panic(err)
	}
	// this code follows ontology/smartcontract/wasmvm/wasm_service.go:Invoke(), with
	// some error checking removed for brevity.
	compiled, err := CompileModule(m)
	if err != nil {
		panic(err)
	}
	vm, err := NewVMWithCompiled(compiled, 10*1024*1024)
	if err != nil {
		t.Fatalf("Could not instantiate vm: %v", err)
	}
	vm.RecoverPanic = true
	GasLimit := uint64(100000000)
	ExecStep := uint64(math.MaxUint64)
	vm.ExecMetrics = &Gas{GasLimit: &GasLimit, GasPrice: 100, GasFactor: 5, ExecStep: &ExecStep}
	vm.CallStackDepth = 100000
	entry, _ := compiled.RawModule.Export.Entries["invoke"]
	index := int64(entry.Index)
	_, err = vm.ExecCode(index, 0)
	assert.Equal(t, err, ErrCallStackDepthExceed)
}
