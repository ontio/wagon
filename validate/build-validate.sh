#!/bin/sh
set -ex

cd rust-wasm-validate
cargo build --release --target wasm32-unknown-unknown
cd target/wasm32-unknown-unknown/release
wasm-prune -e wasm_validate,alloc_buffer wasm_validate.wasm wasm_validate_prune.wasm

wasm2wat --no-debug-names wasm_validate_prune.wasm  -o wasm_validate_prune.wast
wat2wasm wasm_validate_prune.wast  -o wasm_validate_prune_no_custom.wasm
cd -

go run ./scripts/hex2bytes.go  target/wasm32-unknown-unknown/release/wasm_validate_prune_no_custom.wasm > ../validate_code.go

cd ..
