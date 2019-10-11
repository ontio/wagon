// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package exec provides functions for executing WebAssembly bytecode.
package exec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/go-interpreter/wagon/disasm"
	"github.com/go-interpreter/wagon/exec/internal/compile"
	"github.com/go-interpreter/wagon/wasm"
	ops "github.com/go-interpreter/wagon/wasm/operators"
)

var (
	// ErrMultipleLinearMemories is returned by (*VM).NewVM when the module
	// has more then one entries in the linear memory space.
	ErrMultipleLinearMemories = errors.New("exec: more than one linear memories in module")
	// ErrInvalidArgumentCount is returned by (*VM).ExecCode when an invalid
	// number of arguments to the WebAssembly function are passed to it.
	ErrInvalidArgumentCount = errors.New("exec: invalid number of arguments to function")
)

// InvalidReturnTypeError is returned by (*VM).ExecCode when the module
// specifies an invalid return type value for the executed function.
type InvalidReturnTypeError int8

func (e InvalidReturnTypeError) Error() string {
	return fmt.Sprintf("Function has invalid return value_type: %d", int8(e))
}

// InvalidFunctionIndexError is returned by (*VM).ExecCode when the function
// index provided is invalid.
type InvalidFunctionIndexError int64

func (e InvalidFunctionIndexError) Error() string {
	return fmt.Sprintf("Invalid index to function index space: %d", int64(e))
}

type context struct {
	stack   []uint64
	locals  []uint64
	code    []byte
	pc      int64
	curFunc int64
}

type Gas struct {
	GasPrice        uint64
	GasLimit        *uint64
	LocalGasCounter uint64
	GasFactor       uint64
}

// VM is the execution context for executing WebAssembly bytecode.
type VM struct {
	ctx context

	module  *wasm.Module
	globals []uint64
	memory  []byte
	funcs   []function

	funcTable [256]func()

	// RecoverPanic controls whether the `ExecCode` method
	// recovers from a panic and returns it as an error
	// instead.
	// A panic can occur either when executing an invalid VM
	// or encountering an invalid instruction, e.g. `unreachable`.
	RecoverPanic bool

	abort bool // Flag for host functions to terminate execution

	//add for ontology gas limit
	AvaliableGas *Gas

	HostData interface{}

	//memory limitation
	MemoryLimitation uint64
	//call stack depth
	CallStackDepth uint32
}

// As per the WebAssembly spec: https://github.com/WebAssembly/design/blob/27ac254c854994103c24834a994be16f74f54186/Semantics.md#linear-memory
const wasmPageSize = wasm.WasmPageSize

var endianess = binary.LittleEndian

type CompiledModule struct {
	RawModule *wasm.Module
	globals   []uint64
	memory    []byte
	funcs     []function
}

func CompileModule(module *wasm.Module) (*CompiledModule, error) {
	var compiled CompiledModule

	if module.Memory != nil && len(module.Memory.Entries) != 0 {
		if len(module.Memory.Entries) > 1 {
			return nil, ErrMultipleLinearMemories
		}

		memsize := uint(module.Memory.Entries[0].Limits.Initial) * wasmPageSize
		compiled.memory = make([]byte, memsize)
		copy(compiled.memory, module.LinearMemoryIndexSpace[0])
	}

	compiled.funcs = make([]function, len(module.FunctionIndexSpace))
	compiled.globals = make([]uint64, len(module.GlobalIndexSpace))
	compiled.RawModule = module

	nNatives := 0
	for i, fn := range module.FunctionIndexSpace {
		// Skip native methods as they need not be
		// disassembled; simply add them at the end
		// of the `funcs` array as is, as specified
		// in the spec. See the "host functions"
		// section of:
		// https://webassembly.github.io/spec/core/exec/modules.html#allocation
		if fn.IsHost() {
			compiled.funcs[i] = goFunction{
				typ: fn.Host.Type(),
				val: fn.Host,
			}
			nNatives++
			continue
		}

		disassembly, err := disasm.NewDisassembly(fn, module)
		if err != nil {
			return nil, err
		}

		totalLocalVars := 0
		totalLocalVars += len(fn.Sig.ParamTypes)
		for _, entry := range fn.Body.Locals {
			totalLocalVars += int(entry.Count)
		}
		code, table := compile.Compile(disassembly.Code)
		compiled.funcs[i] = compiledFunction{
			code:           code,
			branchTables:   table,
			maxDepth:       disassembly.MaxDepth,
			totalLocalVars: totalLocalVars,
			args:           len(fn.Sig.ParamTypes),
			returns:        len(fn.Sig.ReturnTypes) != 0,
		}
	}

	for i, global := range module.GlobalIndexSpace {
		val, err := module.ExecInitExpr(global.Init)
		if err != nil {
			return nil, err
		}
		switch v := val.(type) {
		case int32:
			compiled.globals[i] = uint64(v)
		case int64:
			compiled.globals[i] = uint64(v)
			//case float32:
			//	compiled.globals[i] = uint64(math.Float32bits(v))
			//case float64:
			//	compiled.globals[i] = uint64(math.Float64bits(v))
		}
	}

	if module.Start != nil {
		//_, err := compiled.ExecCode(int64(module.Start.Index))
		//if err != nil {
		//	return nil, err
		//}
		return nil, errors.New("start entry is not supported in smart contract")
	}

	return &compiled, nil
}

func NewVMWithCompiled(module *CompiledModule, memLimit uint64) (*VM, error) {
	var vm VM

	memsize := len(module.memory)
	if uint64(memsize) > memLimit {
		return nil, fmt.Errorf("memory is exceed the limitation of %d", memLimit)
	}
	vm.MemoryLimitation = memLimit
	vm.memory = make([]byte, memsize)
	copy(vm.memory, module.memory)

	vm.funcs = module.funcs
	vm.globals = make([]uint64, len(module.RawModule.GlobalIndexSpace))
	copy(vm.globals, module.globals)
	vm.newFuncTable()
	vm.module = module.RawModule

	return &vm, nil
}

// NewVM creates a new VM from a given module. If the module defines a
// start function, it will be executed.
func NewVM(module *wasm.Module, memLimit uint64) (*VM, error) {
	compiled, err := CompileModule(module)
	if err != nil {
		return nil, err
	}

	return NewVMWithCompiled(compiled, memLimit)
}

// Memory returns the linear memory space for the VM.
func (vm *VM) Memory() []byte {
	return vm.memory
}

func (vm *VM) pushBool(v bool) {
	if v {
		vm.pushUint64(1)
	} else {
		vm.pushUint64(0)
	}
}

func (vm *VM) fetchBool() bool {
	return vm.fetchInt8() != 0
}

func (vm *VM) fetchInt8() int8 {
	i := int8(vm.ctx.code[vm.ctx.pc])
	vm.ctx.pc++
	return i
}

func (vm *VM) fetchUint32() uint32 {
	v := endianess.Uint32(vm.ctx.code[vm.ctx.pc:])
	vm.ctx.pc += 4
	return v
}

func (vm *VM) fetchInt32() int32 {
	return int32(vm.fetchUint32())
}

func (vm *VM) fetchFloat32() float32 {
	return math.Float32frombits(vm.fetchUint32())
}

func (vm *VM) fetchUint64() uint64 {
	v := endianess.Uint64(vm.ctx.code[vm.ctx.pc:])
	vm.ctx.pc += 8
	return v
}

func (vm *VM) fetchInt64() int64 {
	return int64(vm.fetchUint64())
}

func (vm *VM) fetchFloat64() float64 {
	return math.Float64frombits(vm.fetchUint64())
}

func (vm *VM) popUint64() uint64 {
	i := vm.ctx.stack[len(vm.ctx.stack)-1]
	vm.ctx.stack = vm.ctx.stack[:len(vm.ctx.stack)-1]
	return i
}

func (vm *VM) popInt64() int64 {
	return int64(vm.popUint64())
}

func (vm *VM) popFloat64() float64 {
	return math.Float64frombits(vm.popUint64())
}

func (vm *VM) popUint32() uint32 {
	return uint32(vm.popUint64())
}

func (vm *VM) popInt32() int32 {
	return int32(vm.popUint32())
}

func (vm *VM) popFloat32() float32 {
	return math.Float32frombits(vm.popUint32())
}

func (vm *VM) pushUint64(i uint64) {
	if debugStackDepth {
		if len(vm.ctx.stack) >= cap(vm.ctx.stack) {
			panic("stack exceeding max depth: " + fmt.Sprintf("len=%d,cap=%d", len(vm.ctx.stack), cap(vm.ctx.stack)))
		}
	}
	vm.ctx.stack = append(vm.ctx.stack, i)
}

func (vm *VM) pushInt64(i int64) {
	vm.pushUint64(uint64(i))
}

func (vm *VM) pushFloat64(f float64) {
	vm.pushUint64(math.Float64bits(f))
}

func (vm *VM) pushUint32(i uint32) {
	vm.pushUint64(uint64(i))
}

func (vm *VM) pushInt32(i int32) {
	vm.pushUint64(uint64(i))
}

func (vm *VM) pushFloat32(f float32) {
	vm.pushUint32(math.Float32bits(f))
}

// ExecCode calls the function with the given index and arguments.
// fnIndex should be a valid index into the function index space of
// the VM's module.
func (vm *VM) ExecCode(fnIndex int64, args ...uint64) (rtrn interface{}, err error) {
	// If used as a library, client code should set vm.RecoverPanic to true
	// in order to have an error returned.
	if vm.RecoverPanic {
		defer func() {
			if r := recover(); r != nil {
				switch e := r.(type) {
				case error:
					err = e
				default:
					err = fmt.Errorf("exec: %v", e)
				}
			}
		}()
	}
	if int(fnIndex) > len(vm.funcs) {
		return nil, InvalidFunctionIndexError(fnIndex)
	}
	if len(vm.module.GetFunction(int(fnIndex)).Sig.ParamTypes) != len(args) {
		return nil, ErrInvalidArgumentCount
	}
	compiled, ok := vm.funcs[fnIndex].(compiledFunction)
	if !ok {
		panic(fmt.Sprintf("exec: function at index %d is not a compiled function", fnIndex))
	}

	depth := compiled.maxDepth + 1
	if cap(vm.ctx.stack) < depth {
		vm.ctx.stack = make([]uint64, 0, depth)
	} else {
		vm.ctx.stack = vm.ctx.stack[:0]
	}

	vm.ctx.locals = make([]uint64, compiled.totalLocalVars)
	vm.ctx.pc = 0
	vm.ctx.code = compiled.code
	vm.ctx.curFunc = fnIndex

	for i, arg := range args {
		vm.ctx.locals[i] = arg
	}

	res, err := vm.execCode(compiled)
	if err != nil {
		return nil, fmt.Errorf("exec:%v", err)
	}
	if compiled.returns {
		rtrnType := vm.module.GetFunction(int(fnIndex)).Sig.ReturnTypes[0]
		switch rtrnType {
		case wasm.ValueTypeI32:
			rtrn = uint32(res)
		case wasm.ValueTypeI64:
			rtrn = uint64(res)
		//case wasm.ValueTypeF32:
		//	rtrn = math.Float32frombits(uint32(res))
		//case wasm.ValueTypeF64:
		//	rtrn = math.Float64frombits(res)
		default:
			return nil, InvalidReturnTypeError(rtrnType)
		}
	}

	return rtrn, nil
}

func (vm *VM) execCode(compiled compiledFunction) (uint64, error) {
outer:
	for int(vm.ctx.pc) < len(vm.ctx.code) && !vm.abort {
		if !vm.checkGas(1) {
			return 0, fmt.Errorf("exec:reach the gas limit")
		}
		op := vm.ctx.code[vm.ctx.pc]
		vm.ctx.pc++
		switch op {
		case ops.Return:
			break outer
		case compile.OpJmp:
			vm.ctx.pc = vm.fetchInt64()
			continue
		case compile.OpJmpZ:
			target := vm.fetchInt64()
			if vm.popUint32() == 0 {
				vm.ctx.pc = target
				continue
			}
		case compile.OpJmpNz:
			target := vm.fetchInt64()
			preserveTop := vm.fetchBool()
			discard := vm.fetchInt64()
			if vm.popUint32() != 0 {
				vm.ctx.pc = target
				var top uint64
				if preserveTop {
					top = vm.ctx.stack[len(vm.ctx.stack)-1]
				}
				vm.ctx.stack = vm.ctx.stack[:len(vm.ctx.stack)-int(discard)]
				if preserveTop {
					vm.pushUint64(top)
				}
				continue
			}
		case ops.BrTable:
			index := vm.fetchInt64()
			label := vm.popInt32()
			cf, ok := vm.funcs[vm.ctx.curFunc].(compiledFunction)
			if !ok {
				panic(fmt.Sprintf("exec: function at index %d is not a compiled function", vm.ctx.curFunc))
			}
			table := cf.branchTables[index]
			var target compile.Target
			if label >= 0 && label < int32(len(table.Targets)) {
				target = table.Targets[int32(label)]
			} else {
				target = table.DefaultTarget
			}

			if target.Return {
				break outer
			}
			vm.ctx.pc = target.Addr
			var top uint64
			if target.PreserveTop {
				top = vm.ctx.stack[len(vm.ctx.stack)-1]
			}
			vm.ctx.stack = vm.ctx.stack[:len(vm.ctx.stack)-int(target.Discard)]
			if target.PreserveTop {
				vm.pushUint64(top)
			}
			continue
		case compile.OpDiscard:
			place := vm.fetchInt64()
			vm.ctx.stack = vm.ctx.stack[:len(vm.ctx.stack)-int(place)]
		case compile.OpDiscardPreserveTop:
			top := vm.ctx.stack[len(vm.ctx.stack)-1]
			place := vm.fetchInt64()
			vm.ctx.stack = vm.ctx.stack[:len(vm.ctx.stack)-int(place)]
			vm.pushUint64(top)
		default:
			vm.funcTable[op]()
		}
	}

	if compiled.returns && !vm.abort {
		return vm.ctx.stack[len(vm.ctx.stack)-1], nil
	}
	return 0, nil
}

//check gas
func (vm *VM) checkGas(gaslimit uint64) bool {
	vm.AvaliableGas.LocalGasCounter += gaslimit
	normalizationGasLimit := vm.AvaliableGas.LocalGasCounter / vm.AvaliableGas.GasFactor

	vm.AvaliableGas.LocalGasCounter = vm.AvaliableGas.LocalGasCounter % vm.AvaliableGas.GasFactor
	if normalizationGasLimit == 0 {
		return true
	}

	if *vm.AvaliableGas.GasLimit >= normalizationGasLimit {
		*vm.AvaliableGas.GasLimit -= normalizationGasLimit
		return true
	}
	return false
}

func (vm *VM) checkCallStackDepth() {
	if vm.CallStackDepth <= 0 {
		panic(ErrCallStackDepthExceed)
	}
	vm.CallStackDepth--

}

// Process is a proxy passed to host functions in order to access
// things such as memory and control.
type Process struct {
	vm *VM
}

// NewProcess creates a VM interface object for host functions
func NewProcess(vm *VM) *Process {
	return &Process{vm: vm}
}

// ReadAt implements the ReaderAt interface: it copies into p
// the content of memory at offset off.
func (proc *Process) ReadAt(p []byte, off int64) (int, error) {
	mem := proc.vm.Memory()

	var length int
	if len(mem) < len(p)+int(off) {
		length = len(mem) - int(off)
	} else {
		length = len(p)
	}

	copy(p, mem[off:off+int64(length)])

	var err error
	if length < len(p) {
		err = io.ErrShortBuffer
	}

	return length, err
}

// WriteAt implements the WriterAt interface: it writes the content of p
// into the VM memory at offset off.
func (proc *Process) WriteAt(p []byte, off int64) (int, error) {
	mem := proc.vm.Memory()

	var length int
	if len(mem) < len(p)+int(off) {
		length = len(mem) - int(off)
	} else {
		length = len(p)
	}

	copy(mem[off:], p[:length])

	var err error
	if length < len(p) {
		err = io.ErrShortWrite
	}

	return length, err
}

// MemSize returns the current allocated memory size in bytes.
func (proc *Process) MemSize() int {
	return len(proc.vm.Memory())
}

// Terminate stops the execution of the current module.
func (proc *Process) Terminate() {
	proc.vm.abort = true
}

func (proc *Process) HostData() interface{} {
	return proc.vm.HostData
}
