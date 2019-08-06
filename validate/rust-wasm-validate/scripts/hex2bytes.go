package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("usage: %s binfile", os.Args[0])
		os.Exit(1)
	}

	name := os.Args[1]
	buff, err := ioutil.ReadFile(name)
	if err != nil {
		panic(err)
	}
	hexStr := hex.EncodeToString(buff)

	fmt.Printf(`package validate

import (
	"encoding/hex"
)

var verifyCode = func () []byte {
	code, err := hex.DecodeString("%s")
	if err != nil {
		panic(err)
	}
	return code
}()
`, hexStr)
}
