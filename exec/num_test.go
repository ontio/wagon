// Copyright 2020 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"fmt"
	"testing"

	"github.com/ontio/wagon/wasm/operators"
)

func TestI32BinOps(t *testing.T) {
	for _, tc := range []struct {
		opcode byte
		z1     int32
		z2     int32
		want   int32
	}{
		{operators.I32Shl, -1, 8, -256},
		{operators.I32Shl, -1, 200, -256},
		{operators.I32ShrS, -1, 8, -1},
		{operators.I32ShrS, -1, 200, -1},
		{operators.I32ShrU, -1, 8, 0xFFFFFF},
		{operators.I32ShrU, -1, 200, 0xFFFFFF},
		{operators.I32Rotl, 0x12345678, 8, 0x34567812},
		{operators.I32Rotl, 0x12345678, 200, 0x34567812},
	} {
		name, err := operators.New(tc.opcode)
		if err != nil {
			t.Fatalf("could not lookup operator 0x%x: %+v", tc.opcode, name)
		}
		t.Run(fmt.Sprintf("%v(%v,%v)", name, tc.z1, tc.z2), func(t *testing.T) {
			vm := new(VM)
			vm.newFuncTable()
			vm.pushInt32(tc.z1)
			vm.pushInt32(tc.z2)
			vm.funcTable[tc.opcode]()
			if got, want := vm.popInt32(), tc.want; got != want {
				t.Fatalf("got=%v, want=%v", got, want)
			}
		})
	}
}

func TestI64BinOps(t *testing.T) {
	for _, tc := range []struct {
		opcode byte
		z1     int64
		z2     int64
		want   int64
	}{
		{operators.I64Shl, -1, 8, -256},
		{operators.I64Shl, -1, 200, -256},
		{operators.I64ShrS, -1, 8, -1},
		{operators.I64ShrS, -1, 200, -1},
		{operators.I64ShrU, -1, 8, 0xFFFFFFFFFFFFFF},
		{operators.I64ShrU, -1, 200, 0xFFFFFFFFFFFFFF},
		{operators.I64Rotl, 0x1234567812345678, 8, 0x3456781234567812},
		{operators.I64Rotl, 0x1234567812345678, 200, 0x3456781234567812},
	} {
		name, err := operators.New(tc.opcode)
		if err != nil {
			t.Fatalf("could not lookup operator 0x%x: %+v", tc.opcode, name)
		}
		t.Run(fmt.Sprintf("%v(%v,%v)", name, tc.z1, tc.z2), func(t *testing.T) {
			vm := new(VM)
			vm.newFuncTable()
			vm.pushInt64(tc.z1)
			vm.pushInt64(tc.z2)
			vm.funcTable[tc.opcode]()
			if got, want := vm.popInt64(), tc.want; got != want {
				t.Fatalf("got=%v, want=%v", got, want)
			}
		})
	}
}

func TestF32BinOps(t *testing.T) {
	for _, tc := range []struct {
		opcode byte
		z1     float32
		z2     float32
		want   float32
	}{
		{operators.F32Sub, 3.0, 2.0, 1.0},
		{operators.F32Copysign, 3.0, 2.0, 3.0},
		{operators.F32Copysign, 3.0, -2.0, -3.0},
	} {
		name, err := operators.New(tc.opcode)
		if err != nil {
			t.Fatalf("could not lookup operator 0x%x: %+v", tc.opcode, name)
		}
		t.Run(fmt.Sprintf("%v(%v,%v)", name, tc.z1, tc.z2), func(t *testing.T) {
			vm := new(VM)
			vm.newFuncTable()
			vm.pushFloat32(tc.z1)
			vm.pushFloat32(tc.z2)
			vm.funcTable[tc.opcode]()
			if got, want := vm.popFloat32(), tc.want; got != want {
				t.Fatalf("got=%v, want=%v", got, want)
			}
		})
	}
}

func TestF64BinOps(t *testing.T) {
	for _, tc := range []struct {
		opcode byte
		z1     float64
		z2     float64
		want   float64
	}{
		{operators.F64Sub, 3.0, 2.0, 1.0},
		{operators.F64Copysign, 3.0, 2.0, 3.0},
		{operators.F64Copysign, 3.0, -2.0, -3.0},
	} {
		name, err := operators.New(tc.opcode)
		if err != nil {
			t.Fatalf("could not lookup operator 0x%x: %+v", tc.opcode, name)
		}
		t.Run(fmt.Sprintf("%v(%v,%v)", name, tc.z1, tc.z2), func(t *testing.T) {
			vm := new(VM)
			vm.newFuncTable()
			vm.pushFloat64(tc.z1)
			vm.pushFloat64(tc.z2)
			vm.funcTable[tc.opcode]()
			if got, want := vm.popFloat64(), tc.want; got != want {
				t.Fatalf("got=%v, want=%v", got, want)
			}
		})
	}
}
