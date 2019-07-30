// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wasm

import (
	"encoding/binary"
	"io"

	"github.com/go-interpreter/wagon/wasm/leb128"
	"bytes"
	"errors"
)
var ErrEof = errors.New("got EOF, can not get the next byte")

func readBytes(r io.Reader, n uint64) ([]byte, error) {
	if n == 0 {
		return nil, nil
	}
	if n < 2*1024*1024 {
		p := make([]byte, n)
		_, err := io.ReadFull(r, p)
		if err != nil {
			return nil, err
		}
		return p, nil
	}
	// normal path to avoid attack
	limited := io.LimitReader(r, int64(n))
	buf := &bytes.Buffer{}
	num, _ := buf.ReadFrom(limited)
	if num == int64(n) {
		return buf.Bytes(), nil
	}
	return nil, ErrEof
}

func readBytesUint(r io.Reader) ([]byte, error) {
	n, err := leb128.ReadVarUint32(r)
	if err != nil {
		return nil, err
	}
	return readBytes(r, int(n))
}

func readString(r io.Reader, n int) (string, error) {
	bytes, err := readBytes(r, n)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func readStringUint(r io.Reader) (string, error) {
	n, err := leb128.ReadVarUint32(r)
	if err != nil {
		return "", err
	}
	return readString(r, int(n))
}

func readU32(r io.Reader) (uint32, error) {
	var buf [4]byte
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf[:]), nil
}

func readU64(r io.Reader) (uint64, error) {
	var buf [8]byte
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(buf[:]), nil
}
