set -ex

find . -name "*.go" | xargs sed -i 's/github.com\/go-interpreter/github.com\/ontio/g'
sed -i 's/github.com\/go-interpreter/github.com\/ontio/g' go.mod
