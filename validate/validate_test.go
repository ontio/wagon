package validate

import (
	"io/ioutil"
	"path/filepath"
	"testing"
)

var testPaths = []string{
	//"../wasm/testdata",
	//"../exec/testdata",
	"../exec/testdata/spec",
}

func TestVerifyModuleFromRust(t *testing.T) {
	for _, dir := range testPaths {
		fnames, err := filepath.Glob(filepath.Join(dir, "*.wasm"))
		if err != nil {
			t.Fatal(err)
		}
		for _, fname := range fnames {
			name := fname
			if filepath.Base(name) == "globals.wasm" { // disable mutable global
				continue
			}
			t.Run(name, func(t *testing.T) {
				raw, err := ioutil.ReadFile(name)
				if err != nil {
					t.Fatal(err)
				}

				err = VerifyWasmCodeFromRust(raw)
				if err != nil {
					t.Fatalf("error to verify module %v", err)
				}
			})
		}
	}
}
