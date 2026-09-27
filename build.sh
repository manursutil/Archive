#!/bin/sh
# Build everything into dist/: 

set -eu
cd "$(dirname "$0")"

go generate
go build -o dist/archive
(cd desktop && deno task compile)

echo "built dist/archive and dist/archive-desktop"
