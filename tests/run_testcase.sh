#!/usr/bin/env bash

set -ex
rm -rf temp
mkdir temp
cd temp

if ! which wast2json ; then
    wget https://github.com/WebAssembly/wabt/releases/download/1.0.13/wabt-1.0.13-linux.tar.gz
    tar -xzvf wabt-1.0.13-linux.tar.gz
    WAST2JOSN=wabt-1.0.13/wast2json
else
    WAST2JOSN=wast2json
fi


go build -o spec_test ../spec_test_runner.go

for file in ../spectestcase/*.wast ; do
    ${WAST2JOSN} ${file}
done

for json in *.json ; do
    ./spec_test ${json}
done
